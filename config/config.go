package config

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

type Configuration struct {
	Port           uint16 `json:"port"`
	UseTls         bool   `json:"useTls"`
	TlsKeyFile     string `json:"tlsKey"`
	TlsCertFile    string `json:"tlsCert"`
	UniqueIdTarget string `json:"uniqueIdTarget"`
	LiveCheck      struct {
		Skip         bool   `json:"skip"`
		ApiBaseUrl   string `json:"apiBaseUrl"`
		ContractName string `json:"contractName"`
	} `json:"liveCheck"`
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
		return nil, errors.New("config \"liveCheck.contractName\" is not configured correctly, must end with \".aleo\"")
	}

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
				return nil, errors.New("config \"uniqueIdTarget\" must be 32 bytes hex- or base64-encoded")
			}

			// convert the unique ID to a hex string
			conf.UniqueIdTarget = hex.EncodeToString(uniqueIdBytes)
		}

		if len(uniqueIdBytes) != 32 {
			return nil, errors.New("config \"uniqueIdTarget\" must be 32 bytes")
		}
	}

	return conf, nil
}
