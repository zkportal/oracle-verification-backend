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

func requestAndParseString(c *http.Client, url string, retry bool) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if retry && (resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusNotFound) {
		log.Printf("contract: requesting %s returned %d, trying again\n", url, resp.StatusCode)
		time.Sleep(3 * time.Second)
		return requestAndParseString(c, url, false)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("contract: did not get an OK response")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result string

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	if result == "null" {
		return nil, errors.New("contract: value is not set")
	}

	result = strings.TrimSuffix(result, "u128")

	byteResult := bigIntStrToBytes(result)
	if byteResult == nil {
		return nil, errors.New("contract: invalid value")
	}

	return byteResult, nil
}

// Retrieves the unique ID from the contract that it uses to verify reports.
// The contract must have a mapping called unique_id, where the values are stored as u128 with keys "1" and "2".
func GetUniqueIDAssert(apiBaseUrl, contractName string) (string, error) {
	apiBaseUrl = strings.TrimSuffix(apiBaseUrl, "/")

	requestUrl1 := apiBaseUrl + "/program/" + url.PathEscape(contractName) + "/mapping/unique_id/1u8"
	requestUrl2 := apiBaseUrl + "/program/" + url.PathEscape(contractName) + "/mapping/unique_id/2u8"

	// the urls below contain some live contract, which has u8 keys and u128 values so it works for testing here
	// requestUrl1 := "https://api.explorer.aleo.org/v1/testnet3/program/aleo_name_service_registry_v3.aleo/mapping/general_settings/2u8"
	// requestUrl2 := "https://api.explorer.aleo.org/v1/testnet3/program/aleo_name_service_registry_v3.aleo/mapping/general_settings/3u8"

	client := &http.Client{
		Timeout: time.Second * 30,
	}

	uniqueIdPart1, err := requestAndParseString(client, requestUrl1, true)
	if err != nil {
		return "", err
	}

	uniqueIdPart2, err := requestAndParseString(client, requestUrl2, true)
	if err != nil {
		return "", err
	}

	uniqueId := append(uniqueIdPart1, uniqueIdPart2...)
	if len(uniqueId) != 32 {
		return "", errors.New("malformed unique id in the contract")
	}

	return hex.EncodeToString(uniqueId), nil
}
