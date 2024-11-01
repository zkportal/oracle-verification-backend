package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/zkportal/oracle-verification-backend/attestation"

	aleo_wrapper "github.com/zkportal/aleo-utils-go"
)

var (
	ErrEncodingUpdateSuggestion = errors.New("user data may be using a different version of encoder, please check for updates")
)

type decodeVerifyHandler struct {
	aleoWrapper     aleo_wrapper.Wrapper
	targetUniqueId  string
	targetPcrValues [3]string
}

type DecodeVerifyRequest struct {
	UserData string `json:"userData"`
	Report   string `json:"report"`
}

type DecodeVerifyResponse struct {
	DecodedData   *attestation.DecodedProofData `json:"decodedData,omitempty"`
	DecodedReport interface{}                   `json:"decodedReport,omitempty"`
	ReportValid   bool                          `json:"reportValid"`
	ErrorMessage  string                        `json:"errorMessage,omitempty"`
}

func respondDecodeVerify(w http.ResponseWriter, decodedData *attestation.DecodedProofData, decodedReport interface{}, err error) {
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

func CreateDecodeVerifyHandler(aleoWrapper aleo_wrapper.Wrapper, uniqueId string, pcrValues [3]string) http.Handler {
	return &decodeVerifyHandler{
		aleoWrapper:     aleoWrapper,
		targetUniqueId:  uniqueId,
		targetPcrValues: pcrValues,
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

	aleoSession, err := dvh.aleoWrapper.NewSession()
	if err != nil {
		log.Println("error creating new aleo session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer aleoSession.Close()

	// technically, we don't even need to recover and decode the data here since it's
	// already in the same format as we do for verification right before hashing.
	// we do it anyway as an extra check that the report follows known encoding scheme.
	recoveredMessage, err := aleoSession.RecoverMessage([]byte(request.UserData))
	if err != nil {
		log.Println("error recovering formatted message:", err)
		respondDecodeVerify(w, nil, nil, err)
		return
	}

	recoveredReport, err := aleoSession.RecoverMessage([]byte(request.Report))
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

	teeType, normalizedReport, err := attestation.DetectReportTypeNormalize(recoveredReport)
	if err != nil {
		respondDecodeVerify(w, decodedData, nil, err)
		return
	}

	report, userData, err := attestation.VerifyReport(teeType, normalizedReport, decodedData.Timestamp, "", dvh.targetUniqueId, dvh.targetPcrValues)
	if err != nil {
		respondDecodeVerify(w, decodedData, nil, err)
		return
	}

	dataBytes, err := attestation.PrepareProofData(decodedData.ResponseStatusCode, decodedData.AttestationData, decodedData.Timestamp, &decodedData.AttestationRequest)
	if err != nil {
		log.Printf("prepareProofData: %v", err)
		respondDecodeVerify(w, decodedData, nil, errors.Join(attestation.ErrVerificationFailedToPrepare, ErrEncodingUpdateSuggestion))
		return
	}

	formattedData, err := aleoSession.FormatMessage(dataBytes, attestation.ALEO_STRUCT_REPORT_DATA_SIZE)
	if err != nil {
		log.Printf("aleo.FormatMessage(): %v\n", err)
		respondDecodeVerify(w, decodedData, nil, attestation.ErrVerificationFailedToFormat)
		return
	}

	attestationHash, err := aleoSession.HashMessage(formattedData)
	if err != nil {
		log.Printf("aleo.HashMessage(): %v\n", err)
		respondDecodeVerify(w, decodedData, nil, attestation.ErrVerificationFailedToHash)
		return
	}

	// Poseidon8 hash is 16 bytes when represented in bytes so here we compare
	// the resulting hash only with 16 out of 64 bytes of the report's user data.
	// IMPORTANT! this needs to be adjusted if we put more data in the report
	if !bytes.Equal(attestationHash, userData[:16]) {
		respondDecodeVerify(w, decodedData, nil, attestation.ErrVerificationFailedToMatchData)
		return
	}

	formattedReport, err := attestation.FormatReport(teeType, report)
	if err != nil {
		respondDecodeVerify(w, decodedData, nil, err)
		return
	}

	respondDecodeVerify(w, decodedData, formattedReport, nil)
}
