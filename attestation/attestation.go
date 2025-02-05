package attestation

import (
	"bytes"
	"errors"
	"log"

	"github.com/zkportal/oracle-verification-backend/attestation/nitro"
	"github.com/zkportal/oracle-verification-backend/attestation/sgx"

	encoding "github.com/zkportal/aleo-oracle-encoding"
	aleo_wrapper "github.com/zkportal/aleo-utils-go"
)

// Tee types
const (
	// AWS Nitro enclave
	TEE_TYPE_NITRO string = "nitro"
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
	ErrUnsupportedReportType         = errors.New("unsupported report type")
)

func VerifyReport(reportType string, report []byte, nonce string, targetUniqueId string, targetPcrValues [3]string) (interface{}, []byte, error) {
	switch reportType {
	case TEE_TYPE_SGX:
		parsedReport, err := sgx.VerifySgxReport(report, targetUniqueId)
		if err != nil {
			return nil, nil, err
		}

		return parsedReport, parsedReport.Data, nil

	case TEE_TYPE_NITRO:
		parsedReport, err := nitro.VerifyNitroReport(report, nonce, targetPcrValues)
		if err != nil {
			return nil, nil, err
		}

		return parsedReport, parsedReport.UserData, nil

	default:
		return nil, nil, ErrUnsupportedReportType
	}
}

func VerifyReportData(aleoSession aleo_wrapper.Session, userData []byte, resp *AttestationResponse) error {
	dataBytes, err := PrepareProofData(resp.ResponseStatusCode, resp.AttestationData, resp.Timestamp, &resp.AttestationRequest)
	if err != nil {
		log.Printf("prepareProofData: %v", err)
		return ErrVerificationFailedToPrepare
	}

	if resp.AttestationRequest.Url == PriceFeedAleoUrl {
		dataBytes[0] = 8
	} else if resp.AttestationRequest.Url == PriceFeedBtcUrl {
		dataBytes[0] = 12
	} else if resp.AttestationRequest.Url == PriceFeedEthUrl {
		dataBytes[0] = 11
	}

	formattedData, err := aleoSession.FormatMessage(dataBytes, ALEO_STRUCT_REPORT_DATA_SIZE)
	if err != nil {
		log.Printf("aleo.FormatMessage(): %v\n", err)
		return ErrVerificationFailedToFormat
	}

	attestationHash, err := aleoSession.HashMessage(formattedData)
	if err != nil {
		log.Printf("aleo.HashMessage(): %v\n", err)
		return ErrVerificationFailedToHash
	}

	// Poseidon8 hash is 16 bytes when represented in bytes so here we compare
	// the resulting hash only with 16 out of 64 bytes of the report's user data.
	// IMPORTANT! this needs to be adjusted if we put more data in the report
	if !bytes.Equal(attestationHash, userData[:16]) {
		return ErrVerificationFailedToMatchData
	}

	return nil
}
