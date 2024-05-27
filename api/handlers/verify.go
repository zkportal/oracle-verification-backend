package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	aleo_signer "github.com/zkportal/aleo-utils-go"

	"github.com/zkportal/oracle-verification-backend/attestation"
)

type verifyHandler struct {
	signer         aleo_signer.Wrapper
	targetUniqueId string
}

type VerifyReportsRequest struct {
	Reports []attestation.AttestationResponse `json:"reports"`
}

type VerifyReportsResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func respondVerify(w http.ResponseWriter, status bool, err error) {
	r := &VerifyReportsResponse{
		Success: status,
	}

	if err != nil {
		r.ErrorMessage = err.Error()
	}

	msg, err := json.Marshal(r)
	if err != nil {
		log.Println("/verify: failed to marshal response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(msg)
}

func CreateVerifyHandler(signer aleo_signer.Wrapper, uniqueId string) http.Handler {
	return &verifyHandler{
		signer:         signer,
		targetUniqueId: uniqueId,
	}
}

func (vh *verifyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("handling /verify")

	body, err := io.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		log.Println("/verify: failed to read request body: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	request := new(VerifyReportsRequest)
	err = json.Unmarshal(body, request)
	if err != nil {
		log.Println("/verify: error reading request", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(request.Reports) == 0 {
		log.Println("/verify: no reports to verify")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	signerSession, err := vh.signer.NewSession()
	if err != nil {
		log.Println("/verify: error creating new signer session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer signerSession.Close()

	for _, v := range request.Reports {
		err := attestation.VerifyReport(signerSession, v, vh.targetUniqueId)
		if err != nil {
			log.Println("/verify: error verifying report:", err)
			respondVerify(w, false, err)
			return
		}
	}

	respondVerify(w, true, nil)
}
