package sgx

import (
	"encoding/hex"
	"errors"
	"log"

	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/eclient"
)

func VerifySgxReport(reportBytes []byte, targetUniqueId string) (*attestation.Report, error) {
	report, err := eclient.VerifyRemoteReport(reportBytes)
	if err != nil {
		return nil, err
	}

	uniqueId := hex.EncodeToString(report.UniqueID)

	if uniqueId != targetUniqueId {
		log.Printf("reporting enclave unique ID doesn't match the expected one, expected=%s, got=%s", targetUniqueId, uniqueId)
		return nil, errors.New("report unique ID doesn't match target")
	}

	return &report, nil
}
