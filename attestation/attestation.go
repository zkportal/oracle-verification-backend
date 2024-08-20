package attestation

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/blocky/nitrite"
	encoding "github.com/zkportal/aleo-oracle-encoding"
	aleo_signer "github.com/zkportal/aleo-utils-go"

	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/eclient"
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

func verifyNitroReport(reportString, nonceString string) ([]byte, error) {
	reportBytes, err := base64.StdEncoding.DecodeString(reportString)
	if err != nil {
		return nil, errors.New("error verifying nitro report: error decoding report")
	}

	log.Println("Report bytes:", hex.EncodeToString(reportBytes))
	log.Println()
	log.Println()
	log.Println()

	report, err := nitrite.Verify(reportBytes, nitrite.VerifyOptions{CurrentTime: time.Date(2024, time.March, 20, 15, 0, 0, 0, time.UTC)})
	if err != nil {
		return nil, err
	}

	log.Println("Protected begin:", hex.EncodeToString(report.Protected))
	// log.Println("Protected length:", len(report.Protected))

	log.Println("Unprotected begin:", hex.EncodeToString(report.Unprotected))
	log.Println("Unprotected length:", len(report.Unprotected))

	log.Println("Payload begin:", hex.EncodeToString(report.Payload[0:32]))
	log.Println("Payload length:", len(report.Payload))

	log.Println("Signature:", hex.EncodeToString(report.Signature))

	log.Println("Module ID as bytes", hex.EncodeToString([]byte(report.Document.ModuleID)))

	timestampBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestampBuf, report.Document.Timestamp)

	log.Println("Little endian timestamp:", hex.EncodeToString(timestampBuf))

	binary.BigEndian.PutUint64(timestampBuf, report.Document.Timestamp)
	log.Println("Big endian timestamp:", hex.EncodeToString(timestampBuf))

	document := Document{
		ModuleID:  report.Document.ModuleID,
		Timestamp: report.Document.Timestamp,
		Digest:    report.Document.Digest,

		PCRs:        make(map[uint]string),
		Certificate: hex.EncodeToString(report.Document.Certificate),
		CABundle:    make([]string, len(report.Document.CABundle)),

		PublicKey: hex.EncodeToString(report.Document.PublicKey),
		UserData:  hex.EncodeToString(report.Document.UserData),
		Nonce:     hex.EncodeToString(report.Document.Nonce),
	}

	for k, v := range report.Document.PCRs {
		document.PCRs[k] = hex.EncodeToString(v)
	}

	for idx, v := range report.Document.CABundle {
		document.CABundle[idx] = hex.EncodeToString(v)
	}

	d, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	log.Println("Document:", string(d))

	nonce := hex.EncodeToString(report.Document.Nonce)

	if nonceString != nonce {
		return nil, errors.New("error verifying nitro report: nonce missmatched")
	}

	return report.Document.UserData, nil
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
	case TEE_TYPE_NITRO:
		usrData, err = verifyNitroReport(resp.AttestationReport, resp.Nonce)
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
