package config

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

const expectedUniqueIdLength = 32
const expectedPcrValueLength = 48

type Configuration struct {
	Port            uint16   `json:"port"`
	UseTls          bool     `json:"useTls"`
	TlsKeyFile      string   `json:"tlsKey"`
	TlsCertFile     string   `json:"tlsCert"`
	UniqueIdTarget  string   `json:"uniqueIdTarget"`
	PcrValuesTarget []string `json:"pcrValuesTarget"`
	LiveCheck       struct {
		Skip         bool   `json:"skip"`
		ApiBaseUrl   string `json:"apiBaseUrl"`
		ContractName string `json:"contractName"`
	} `json:"liveCheck"`
}

func validateAndNormalizeUniqueId(conf *Configuration) error {
	// check the unique ID for correctness, if it's base64 then convert to hex
	if len(conf.UniqueIdTarget) != 0 {
		var uniqueIdBytes []byte
		var err error

		uniqueIdBytes, err = hex.DecodeString(conf.UniqueIdTarget)
		isHex := err == nil

		// now try decoding as base64
		if !isHex {
			uniqueIdBytes, err = base64.StdEncoding.DecodeString(conf.UniqueIdTarget)
			if err != nil {
				log.Printf("config: invalid SGX Unique ID: \"%s\"\n", conf.UniqueIdTarget)
				return fmt.Errorf("config \"uniqueIdTarget\" must be %d bytes hex- or base64-encoded", expectedUniqueIdLength)
			}

			// convert the unique ID to a hex string
			conf.UniqueIdTarget = hex.EncodeToString(uniqueIdBytes)
		}

		if len(uniqueIdBytes) != expectedUniqueIdLength {
			log.Printf("config: invalid SGX Unique ID: \"%s\"\n", conf.UniqueIdTarget)
			return fmt.Errorf("config \"uniqueIdTarget\" must be %d bytes", expectedUniqueIdLength)
		}
	}

	return nil
}

func validateAndNormalizePcrValues(conf *Configuration) error {
	for pcrIdx, pcr := range conf.PcrValuesTarget {
		var pcrBytes []byte
		var err error

		pcrBytes, err = hex.DecodeString(pcr)
		isHex := err == nil

		// now try decoding as base64
		if !isHex {
			pcrBytes, err = base64.StdEncoding.DecodeString(pcr)
			if err != nil {
				log.Printf("config: invalid Nitro PCR value: \"%s\"\n", pcr)
				return fmt.Errorf("config \"pcrValuesTarget\" values must be %d bytes hex- or base64-encoded", expectedPcrValueLength)
			}

			// convert the PCR value to a hex string
			conf.PcrValuesTarget[pcrIdx] = hex.EncodeToString(pcrBytes)
		}

		if len(pcrBytes) != expectedPcrValueLength {
			log.Printf("config: invalid Nitro PCR value: \"%s\"\n", pcr)
			return fmt.Errorf("config \"pcrValuesTarget\" values must be %d bytes", expectedPcrValueLength)
		}
	}

	return nil
}

func LoadConfig(confContent []byte) (*Configuration, error) {
	conf := new(Configuration)

	err := json.Unmarshal(confContent, conf)
	if err != nil {
		return nil, err
	}

	if conf.LiveCheck.ApiBaseUrl == "" || conf.LiveCheck.ContractName == "" {
		return nil, errors.New("config \"liveCheck\" is not configured correctly, must have \"apiBaseUrl\" and \"contractName\"")
	}

	if !strings.HasSuffix(conf.LiveCheck.ContractName, ".aleo") {
		conf.LiveCheck.ContractName = conf.LiveCheck.ContractName + ".aleo"
	}

	err = validateAndNormalizeUniqueId(conf)
	if err != nil {
		return nil, err
	}

	err = validateAndNormalizePcrValues(conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
