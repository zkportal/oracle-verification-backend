package contract

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func bigIntStrToBytes(strBigInt string) []byte {
	num := new(big.Int)

	num, ok := num.SetString(strBigInt, 10)
	if !ok {
		return nil
	}

	bytes := make([]byte, 16)

	// Extract bytes in little-endian order
	for i := 0; i < 16; i++ {
		b := byte(num.Uint64() & 0xff)
		bytes[i] = b

		num.Rsh(num, 8)
	}

	return bytes
}

func requestProgramString(c *http.Client, url string, retry bool) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if retry && (resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusNotFound) {
		log.Printf("contract: requesting %s returned %d, trying again\n", url, resp.StatusCode)
		time.Sleep(3 * time.Second)
		return requestProgramString(c, url, false)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", errors.New("contract: Aleo node API responded with 429 Too Many Requests, try again later")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("contract: did not get an OK response")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result string

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	if result == "null" {
		return "", errors.New("contract: value is not set")
	}

	return result, nil
}

func parseSgxUniqueIdStruct(uniqueIdStructString string) (string, error) {
	// The unique ID is stored using this type:
	// struct Unique_id {
	//   chunk_1: u128,
	//   chunk_2: u128
	// }

	chunks := strings.Split(uniqueIdStructString, "\n")

	uniqueId := make([]byte, 0, 32)

	for _, chunk := range chunks {
		if !strings.Contains(chunk, "chunk_") {
			continue
		}

		valBegin := strings.Index(chunk, ": ")
		valEnd := strings.Index(chunk, "u128")

		if valBegin == -1 || valEnd == -1 {
			return "", errors.New("unexpected type of unique ID chunks in sgx_unique_id mapping")
		}

		uniqueIdPart := chunk[valBegin+2 : valEnd]
		uniqueId = append(uniqueId, bigIntStrToBytes(uniqueIdPart)...)
	}

	if len(uniqueId) != 32 {
		return "", errors.New("malformed unique id in the contract")
	}

	return hex.EncodeToString(uniqueId), nil
}

func parseNitroPcrValues(nitroPcrStructString string) ([]string, error) {
	// The PCR values are stored using this type:
	// struct PCR_values {
	//   pcr_0_chunk_1: u128,
	//   pcr_0_chunk_2: u128,
	//   pcr_0_chunk_3: u128,
	//   pcr_1_chunk_1: u128,
	//   pcr_1_chunk_2: u128,
	//   pcr_1_chunk_3: u128,
	//   pcr_2_chunk_1: u128,
	//   pcr_2_chunk_2: u128,
	//   pcr_2_chunk_3: u128
	// }

	chunks := strings.Split(nitroPcrStructString, "\n")

	pcrValues := make([][]byte, 0, 9)
	for _, chunk := range chunks {
		if !strings.Contains(chunk, "pcr_") {
			continue
		}

		valBegin := strings.Index(chunk, ": ")
		valEnd := strings.Index(chunk, "u128")

		if valBegin == -1 || valEnd == -1 {
			return nil, errors.New("unexpected type of PCR value chunks in nitro_pcr_values mapping")
		}

		pcrValuePart := chunk[valBegin+2 : valEnd]
		pcrValues = append(pcrValues, bigIntStrToBytes(pcrValuePart))
	}

	if len(pcrValues) != 9 {
		return nil, errors.New("unexpected type of PCR values struct in nitro_pcr_values mapping")
	}

	pcrs := make([]string, 3)
	for pcrIdx := 0; pcrIdx < 3; pcrIdx++ {
		pcr := make([]byte, 0, 48)
		pcr = append(pcr, pcrValues[pcrIdx*3]...)
		pcr = append(pcr, pcrValues[pcrIdx*3+1]...)
		pcr = append(pcr, pcrValues[pcrIdx*3+2]...)

		pcrs[pcrIdx] = hex.EncodeToString(pcr)
	}

	return pcrs, nil
}

// Retrieves the SGX unique ID from the contract that it uses to verify reports.
// The contract must have a mapping called sgx_unique_id, where the value us stored as a struct under the "0u8" key.
func GetSgxUniqueIDAssert(apiBaseUrl, contractName string) (string, error) {
	apiBaseUrl = strings.TrimSuffix(apiBaseUrl, "/")

	requestUrl := apiBaseUrl + "/program/" + url.PathEscape(contractName) + "/mapping/sgx_unique_id/0u8"

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	uniqueIdStructString, err := requestProgramString(client, requestUrl, true)
	if err != nil {
		return "", err
	}

	return parseSgxUniqueIdStruct(uniqueIdStructString)
}

// Retrieves the Nitro PCR values from the contract that it uses to verify reports.
// The contract must have a mapping called nitro_pcr_values, where the value us stored as a struct under the "0u8" key.
func GetNitroPcrValuesAssert(apiBaseUrl, contractName string) ([]string, error) {
	apiBaseUrl = strings.TrimSuffix(apiBaseUrl, "/")

	requestUrl := apiBaseUrl + "/program/" + url.PathEscape(contractName) + "/mapping/nitro_pcr_values/0u8"

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	pcrsStructString, err := requestProgramString(client, requestUrl, true)
	if err != nil {
		return nil, err
	}

	return parseNitroPcrValues(pcrsStructString)
}
