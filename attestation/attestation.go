package attestation

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log"

	encoding "github.com/zkportal/aleo-oracle-encoding"
	aleo_signer "github.com/zkportal/aleo-utils-go"

	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/eclient"
)

// Tee types
const (
	// Intel SGX
	TEE_TYPE_SGX string = "sgx"

	ALEO_STRUCT_REPORT_DATA_SIZE = 8
)

type AttestationRequest struct {
	Url string `json:"url"`

	RequestMethod  string  `json:"requestMethod"`
	Selector       string  `json:"selector,omitempty"`
	ResponseFormat string  `json:"responseFormat"`
	HTMLResultType *string `json:"htmlResultType,omitempty"`

	RequestBody        *string `json:"requestBody,omitempty"`
	RequestContentType *string `json:"requestContentType,omitempty"`

	RequestHeaders map[string]string `json:"requestHeaders,omitempty"`

	EncodingOptions encoding.EncodingOptions `json:"encodingOptions"`

	DebugRequest bool `json:"debugRequest,omitempty"`
}

type AttestationResponse struct {
	AttestationReport  string             `json:"attestationReport"`
	ReportType         string             `json:"reportType"`
	AttestationData    string             `json:"attestationData"`
	ResponseBody       string             `json:"responseBody"`
	ResponseStatusCode int                `json:"responseStatusCode"`
	Nonce              string             `json:"nonce,omitempty"`
	Timestamp          int64              `json:"timestamp"`
	AttestationRequest AttestationRequest `json:"attestationRequest"`
}

var (
	ErrVerificationFailedToPrepare   = errors.New("verification error: failed to prepare data for report verification")
	ErrVerificationFailedToFormat    = errors.New("verification error: failed to format message for report verification")
	ErrVerificationFailedToHash      = errors.New("verification error: failed to hash message for report verification")
	ErrVerificationFailedToMatchData = errors.New("verification error: userData hashes don't match")
)

func VerifySgxReport(reportString string) (*attestation.Report, error) {
	reportBytes, err := base64.StdEncoding.DecodeString(reportString)
	if err != nil {
		return nil, errors.New("error verifying sgx report: error decoding report bytes")
	}

	report, err := eclient.VerifyRemoteReport(reportBytes)
	if err != nil {
		return nil, err
	}

	return &report, nil
}

func VerifyReport(signerSession aleo_signer.Session, resp AttestationResponse, targetUniqueId string) error {
	var usrData []byte
	var err error

	parsedUniqueId := ""
	switch resp.ReportType {
	case TEE_TYPE_SGX:
		var report *attestation.Report
		report, err = VerifySgxReport(resp.AttestationReport)
		if err != nil {
			return err
		}
		usrData = report.Data
		parsedUniqueId = hex.EncodeToString(report.UniqueID)
	default:
		err = errors.New("unknown TEE type")
	}
	if err != nil {
		return err
	}

	if parsedUniqueId != targetUniqueId {
		log.Printf("reporting enclave unique ID doesn't match the expected one, expected=%s, got=%s", targetUniqueId, parsedUniqueId)
		return errors.New("report unique ID doesn't match target")
	}

	dataBytes, err := PrepareProofData(resp.ResponseStatusCode, resp.AttestationData, resp.Timestamp, &resp.AttestationRequest)
	if err != nil {
		log.Printf("prepareProofData: %v", err)
		return ErrVerificationFailedToPrepare
	}

	formattedData, err := signerSession.FormatMessage(dataBytes, ALEO_STRUCT_REPORT_DATA_SIZE)
	if err != nil {
		log.Printf("aleo.FormatMessage(): %v\n", err)
		return ErrVerificationFailedToFormat
	}

	attestationHash, err := signerSession.HashMessage(formattedData)
	if err != nil {
		log.Printf("aleo.HashMessage(): %v\n", err)
		return ErrVerificationFailedToHash
	}

	// Poseidon8 hash is 16 bytes when represented in bytes so here we compare
	// the resulting hash only with 16 out of 64 bytes of the report's user data.
	// IMPORTANT! this needs to be adjusted if we put more data in the report
	if !bytes.Equal(attestationHash, usrData[:16]) {
		return ErrVerificationFailedToMatchData
	}

	return nil
}
