package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	aleo_signer "github.com/zkportal/aleo-utils-go"

	"github.com/zkportal/oracle-verification-backend/attestation"
	"github.com/zkportal/oracle-verification-backend/u128"

	"github.com/edgelesssys/ego/attestation/tcbstatus"
)

var (
	ErrEncodingUpdateSuggestion = errors.New("user data may be using a different version of encoder, please check for updates")
)

type decodeVerifyHandler struct {
	signer         aleo_signer.Wrapper
	targetUniqueId string
}

type DecodeVerifyRequest struct {
	UserData string `json:"userData"`
	Report   string `json:"report"`
}

type DecodedReport struct {
	Data            []byte           `json:"data"`            // The report data that has been included in the report.
	SecurityVersion uint             `json:"securityVersion"` // Security version of the enclave. For SGX enclaves, this is the ISVSVN value.
	Debug           bool             `json:"debug"`           // If true, the report is for a debug enclave.
	UniqueID        []byte           `json:"uniqueId"`        // The unique ID for the enclave. For SGX enclaves, this is the MRENCLAVE value.
	AleoUniqueID    [2]string        `json:"aleoUniqueId"`    // Same as UniqueID but encoded for Aleo as 2 uint128
	SignerID        []byte           `json:"signerId"`        // The signer ID for the enclave. For SGX enclaves, this is the MRSIGNER value.
	AleoSignerID    [2]string        `json:"aleoSignerId"`    // Same as SignerID but encoded for Aleo as 2 uint128
	ProductID       []byte           `json:"productId"`       // The Product ID for the enclave. For SGX enclaves, this is the ISVPRODID value.
	AleoProductID   string           `json:"aleoProductId"`   // Same as ProductID but encoded for Aleo as 1 uint128
	TCBStatus       tcbstatus.Status `json:"tcbStatus"`       // The status of the enclave's TCB level.
}

type DecodeVerifyResponse struct {
	DecodedData   *attestation.DecodedProofData `json:"decodedData,omitempty"`
	DecodedReport *DecodedReport                `json:"decodedReport,omitempty"`
	ReportValid   bool                          `json:"reportValid"`
	ErrorMessage  string                        `json:"errorMessage,omitempty"`
}

func respondDecodeVerify(w http.ResponseWriter, decodedData *attestation.DecodedProofData, decodedReport *DecodedReport, err error) {
	r := &DecodeVerifyResponse{
		DecodedData:   decodedData,
		DecodedReport: decodedReport,
		ReportValid:   decodedData != nil && decodedReport != nil,
	}

	if err != nil {
		r.ErrorMessage = err.Error()
	}

	msg, err := json.Marshal(r)
	if err != nil {
		log.Println("failed to marshal response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(msg)
}

func CreateDecodeVerifyHandler(signer aleo_signer.Wrapper, uniqueId string) http.Handler {
	return &decodeVerifyHandler{
		signer:         signer,
		targetUniqueId: uniqueId,
	}
}

func (dvh *decodeVerifyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("handling /decodeReport")

	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	request := new(DecodeVerifyRequest)
	err = json.Unmarshal(body, request)
	if err != nil {
		log.Println("error reading request", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(request.Report) == 0 || len(request.UserData) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	signerSession, err := dvh.signer.NewSession()
	if err != nil {
		log.Println("error creating new signer session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer signerSession.Close()

	// technically, we don't even need to recover and decode the data here since it's
	// already in the same format as we do for verification right before hashing.
	// we do it anyway as an extra check that the report follows known encoding scheme.
	recoveredMessage, err := signerSession.RecoverMessage([]byte(request.UserData))
	if err != nil {
		log.Println("error recovering formatted message:", err)
		respondDecodeVerify(w, nil, nil, err)
		return
	}

	recoveredReport, err := signerSession.RecoverMessage([]byte(request.Report))
	if err != nil {
		log.Println("error recovering formatted report:", err)
		respondDecodeVerify(w, nil, nil, err)
		return
	}

	decodedData, err := attestation.DecodeProofData(recoveredMessage)
	if err != nil {
		log.Println("error recovering formatted message:", err)
		respondDecodeVerify(w, nil, nil, errors.Join(err, ErrEncodingUpdateSuggestion))
		return
	}

	report, err := attestation.VerifySgxReport(base64.StdEncoding.EncodeToString(recoveredReport))
	if err != nil {
		respondDecodeVerify(w, decodedData, nil, err)
		return
	}

	parsedUniqueId := hex.EncodeToString(report.UniqueID)
	if parsedUniqueId != dvh.targetUniqueId {
		log.Printf("reporting enclave unique ID doesn't match the expected one, expected=%s, got=%s", dvh.targetUniqueId, parsedUniqueId)
		respondDecodeVerify(w, nil, nil, errors.New("report unique ID doesn't match target"))
		return
	}

	dataBytes, err := attestation.PrepareProofData(decodedData.ResponseStatusCode, decodedData.AttestationData, decodedData.Timestamp, &decodedData.AttestationRequest)
	if err != nil {
		log.Printf("prepareProofData: %v", err)
		respondDecodeVerify(w, decodedData, nil, errors.Join(attestation.ErrVerificationFailedToPrepare, ErrEncodingUpdateSuggestion))
		return
	}

	formattedData, err := signerSession.FormatMessage(dataBytes, attestation.ALEO_STRUCT_REPORT_DATA_SIZE)
	if err != nil {
		log.Printf("aleo.FormatMessage(): %v\n", err)
		respondDecodeVerify(w, decodedData, nil, attestation.ErrVerificationFailedToFormat)
		return
	}

	attestationHash, err := signerSession.HashMessage(formattedData)
	if err != nil {
		log.Printf("aleo.HashMessage(): %v\n", err)
		respondDecodeVerify(w, decodedData, nil, attestation.ErrVerificationFailedToHash)
		return
	}

	// Poseidon8 hash is 16 bytes when represented in bytes so here we compare
	// the resulting hash only with 16 out of 64 bytes of the report's user data.
	// IMPORTANT! this needs to be adjusted if we put more data in the report
	if !bytes.Equal(attestationHash, report.Data[:16]) {
		respondDecodeVerify(w, decodedData, nil, attestation.ErrVerificationFailedToMatchData)
		return
	}

	// encode some of the enclave measurements to Leo values so that the user
	// could compare them with some of the "magic" numbers in a contract
	aleoUniqueId1, _ := u128.SliceToU128(report.UniqueID[0:16])
	aleoUniqueId2, _ := u128.SliceToU128(report.UniqueID[16:32])

	aleoSignerId1, _ := u128.SliceToU128(report.SignerID[0:16])
	aleoSignerId2, _ := u128.SliceToU128(report.SignerID[16:32])

	aleoProductId, _ := u128.SliceToU128(report.ProductID[0:16])

	decodedReport := &DecodedReport{
		Data:            report.Data,
		SecurityVersion: report.SecurityVersion,
		Debug:           report.Debug,
		UniqueID:        report.UniqueID,
		SignerID:        report.SignerID,
		ProductID:       report.ProductID,
		TCBStatus:       report.TCBStatus,

		AleoUniqueID:  [2]string{fmt.Sprintf("%su128", aleoUniqueId1.String()), fmt.Sprintf("%su128", aleoUniqueId2.String())},
		AleoSignerID:  [2]string{fmt.Sprintf("%su128", aleoSignerId1.String()), fmt.Sprintf("%su128", aleoSignerId2.String())},
		AleoProductID: fmt.Sprintf("%su128", aleoProductId.String()),
	}

	respondDecodeVerify(w, decodedData, decodedReport, nil)
}
