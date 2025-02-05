package nitro

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"

	"github.com/blocky/nitrite"
	"github.com/zkportal/oracle-verification-backend/u128"
)

var verifier *nitrite.Verifier
var initErr error
var initOnce sync.Once

type Document struct {
	ModuleID    string `cbor:"module_id" json:"module_id"`
	Timestamp   uint64 `cbor:"timestamp" json:"timestamp"`
	Digest      string `cbor:"digest" json:"digest"`
	Certificate string `cbor:"certificate" json:"certificate"`

	PCRs     map[uint]string `cbor:"pcrs" json:"pcrs"`
	CABundle []string        `cbor:"cabundle" json:"cabundle"`

	PublicKey string `cbor:"public_key" json:"public_key,omitempty"`
	UserData  string `cbor:"user_data" json:"user_data,omitempty"`
	Nonce     string `cbor:"nonce" json:"nonce,omitempty"`
}

func Init() error {
	initOnce.Do(func() {
		log.Println("nitro: initializing verifier...")
		verifier, initErr = nitrite.New(nitrite.WithVerificationTime(nitrite.AttestationTime))
	})

	return initErr
}

func VerifyNitroReport(reportBytes []byte, nonceString string, targetPcrValues [3]string) (*nitrite.Document, error) {
	if verifier == nil {
		panic("nitro verifier is not initialized")
	}

	report, err := verifier.Verify(reportBytes)
	if err != nil {
		return nil, err
	}

	nonce := hex.EncodeToString(report.Nonce)

	if nonceString != "" && nonceString != nonce {
		return nil, errors.New("error verifying nitro report: nonce missmatched")
	}

	var pcrValues [3]string

	for i := uint(0); i < 3; i++ {
		pcrValues[i] = hex.EncodeToString(report.PCRs[i])
	}

	if !slices.Equal(pcrValues[:], targetPcrValues[:]) {
		log.Printf("reporting enclave PCR values don't match the expected ones, expected=[%s], got=[%s]", strings.Join(targetPcrValues[:], ", "), strings.Join(pcrValues[:], ", "))
		return nil, errors.New("report PCR values don't match target")
	}

	if len(report.UserData) != 16 {
		return nil, errors.New("unexpected length of the attestation report data")
	}

	nitriteDocument := nitrite.Document(report)

	return &nitriteDocument, nil
}

func FormatPcrValues(pcrs [3][48]byte) string {
	// struct PCR_values {
	//   pcr_1_chunk_1: u128,
	//   pcr_1_chunk_2: u128,
	//   pcr_1_chunk_3: u128,
	//   pcr_2_chunk_1: u128,
	//   pcr_2_chunk_2: u128,
	//   pcr_2_chunk_3: u128,
	//   pcr_3_chunk_1: u128,
	//   pcr_3_chunk_2: u128,
	//   pcr_3_chunk_3: u128
	// }

	// Building a one long string for the type above

	pairs := make([]string, 0, 9)

	for pcrIdx := 0; pcrIdx < 3; pcrIdx++ {
		pcr := pcrs[pcrIdx]
		for chunkIdx := 0; chunkIdx < 3; chunkIdx++ {
			value, _ := u128.SliceToU128(pcr[chunkIdx*16 : (chunkIdx+1)*16])
			pairs = append(pairs, fmt.Sprintf("pcr_%d_chunk_%d: %su128", pcrIdx, chunkIdx+1, value.String()))
		}
	}

	return "{ " + strings.Join(pairs, ", ") + " }"
}
