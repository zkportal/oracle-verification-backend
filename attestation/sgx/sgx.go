package sgx

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/zkportal/oracle-verification-backend/u128"

	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/attestation/tcbstatus"
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

type DecodedReport struct {
	Data            []byte           `json:"data"`            // The report data that has been included in the report.
	SecurityVersion uint             `json:"securityVersion"` // Security version of the enclave. For SGX enclaves, this is the ISVSVN value.
	Debug           bool             `json:"debug"`           // If true, the report is for a debug enclave.
	UniqueID        []byte           `json:"uniqueId"`        // The unique ID for the enclave. For SGX enclaves, this is the MRENCLAVE value.
	AleoUniqueID    string           `json:"aleoUniqueId"`    // Same as UniqueID but encoded for Aleo as 2 uint128
	SignerID        []byte           `json:"signerId"`        // The signer ID for the enclave. For SGX enclaves, this is the MRSIGNER value.
	AleoSignerID    string           `json:"aleoSignerId"`    // Same as SignerID but encoded for Aleo as 2 uint128
	ProductID       []byte           `json:"productId"`       // The Product ID for the enclave. For SGX enclaves, this is the ISVPRODID value.
	AleoProductID   string           `json:"aleoProductId"`   // Same as ProductID but encoded for Aleo as 1 uint128
	TCBStatus       tcbstatus.Status `json:"tcbStatus"`       // The status of the enclave's TCB level.
}

func FormatReport(reportVar interface{}) (interface{}, error) {
	report, ok := reportVar.(*attestation.Report)
	if !ok {
		return nil, errors.New("unexpected report type")
	}

	// encode some of the enclave measurements to Leo values so that the user
	// could compare them with some of the "magic" numbers in a contract
	aleoUniqueId1, _ := u128.SliceToU128(report.UniqueID[0:16])
	aleoUniqueId2, _ := u128.SliceToU128(report.UniqueID[16:32])

	aleoSignerId1, _ := u128.SliceToU128(report.SignerID[0:16])
	aleoSignerId2, _ := u128.SliceToU128(report.SignerID[16:32])

	aleoProductId, _ := u128.SliceToU128(report.ProductID[0:16])

	aleoStructFormat := "{ chunk_1: %su128, chunk_2: %su128 }"

	decodedReport := &DecodedReport{
		Data:            report.Data,
		SecurityVersion: report.SecurityVersion,
		Debug:           report.Debug,
		UniqueID:        report.UniqueID,
		SignerID:        report.SignerID,
		ProductID:       report.ProductID,
		TCBStatus:       report.TCBStatus,

		AleoUniqueID:  fmt.Sprintf(aleoStructFormat, aleoUniqueId1.String(), aleoUniqueId2.String()),
		AleoSignerID:  fmt.Sprintf(aleoStructFormat, aleoSignerId1.String(), aleoSignerId2.String()),
		AleoProductID: aleoProductId.String() + "u128",
	}

	return decodedReport, nil
}
