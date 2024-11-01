package nitro

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/zkportal/oracle-verification-backend/u128"

	"github.com/blocky/nitrite"
)

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

func VerifyNitroReport(reportBytes []byte, timestamp int64, nonceString string, targetPcrValues [3]string) (*nitrite.Result, error) {
	report, err := nitrite.Verify(reportBytes, nitrite.VerifyOptions{CurrentTime: time.Unix(timestamp, 0)})
	if err != nil {
		return nil, err
	}

	nonce := hex.EncodeToString(report.Document.Nonce)

	if nonceString != "" && nonceString != nonce {
		return nil, errors.New("error verifying nitro report: nonce missmatched")
	}

	var pcrValues [3]string

	for i := uint(0); i < 3; i++ {
		pcrValues[i] = hex.EncodeToString(report.Document.PCRs[i])
	}

	if !slices.Equal(pcrValues[:], targetPcrValues[:]) {
		log.Printf("reporting enclave PCR values don't match the expected ones, expected=[%s], got=[%s]", strings.Join(targetPcrValues[:], ", "), strings.Join(pcrValues[:], ", "))
		return nil, errors.New("report PCR values don't match target")
	}

	if len(report.Document.UserData) != 16 {
		return nil, errors.New("unexpected length of the attestation report data")
	}

	return report, nil
}

type DecodedReport struct {
	ModuleID         string          `json:"moduleID"`
	Timestamp        uint64          `json:"timestamp"`
	Digest           string          `json:"digest"`
	PCRs             map[uint][]byte `json:"pcrs"`
	AleoPCRs         string          `json:"aleoPcrs"`
	Certificate      []byte          `json:"certificate"`
	CABundle         [][]byte        `json:"cabundle"`
	PublicKey        []byte          `json:"publicKey,omitempty"`
	UserData         []byte          `json:"userData"`
	Nonce            []byte          `json:"nonce"`
	ProtectedSection []byte          `json:"protectedCose"` // Protected section from the COSE Sign1 payload
	Signature        []byte          `json:"signature"`     // Attestation document signature
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

func FormatReport(reportVar interface{}) (interface{}, error) {
	report, ok := reportVar.(*nitrite.Result)
	if !ok {
		return nil, errors.New("unexpected report type")
	}

	pcr0 := report.Document.PCRs[0]
	pcr1 := report.Document.PCRs[1]
	pcr2 := report.Document.PCRs[2]

	return &DecodedReport{
		ModuleID:         report.Document.ModuleID,
		Timestamp:        report.Document.Timestamp,
		Digest:           report.Document.Digest,
		PCRs:             report.Document.PCRs,
		AleoPCRs:         FormatPcrValues([3][48]byte{[48]byte(pcr0), [48]byte(pcr1), [48]byte(pcr2)}),
		Certificate:      report.Document.Certificate,
		CABundle:         report.Document.CABundle,
		PublicKey:        report.Document.PublicKey,
		UserData:         report.Document.UserData,
		Nonce:            report.Document.Nonce,
		ProtectedSection: report.Protected,
		Signature:        report.Signature,
	}, nil
}
