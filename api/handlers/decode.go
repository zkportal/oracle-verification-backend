package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	aleo_signer "github.com/zkportal/aleo-utils-go"

	"github.com/zkportal/oracle-verification-backend/attestation"
)

type DecodeProofDataRequest struct {
	UserData string `json:"userData"`
}

type DecodeProofDataResponse struct {
	DecodedData  *attestation.DecodedProofData `json:"decodedData,omitempty"`
	Success      bool                          `json:"success"`
	ErrorMessage string                        `json:"errorMessage,omitempty"`
}

func respondDecode(w http.ResponseWriter, decodedData *attestation.DecodedProofData, err error) {
	r := &DecodeProofDataResponse{
		DecodedData: decodedData,
		Success:     decodedData != nil,
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

func CreateDecodeHandler(aleo aleo_signer.Wrapper) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if req.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("handling /decode")

		body, err := io.ReadAll(req.Body)
		defer req.Body.Close()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		request := new(DecodeProofDataRequest)
		err = json.Unmarshal(body, request)
		if err != nil {
			log.Println("error reading request", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if request.UserData == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		signerSession, err := aleo.NewSession()
		if err != nil {
			log.Println("error creating new signer session:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer signerSession.Close()

		recoveredMessage, err := signerSession.RecoverMessage([]byte(request.UserData))
		if err != nil {
			log.Println("error recovering formatted message:", err)
			respondDecode(w, nil, err)
			return
		}

		decodedData, err := attestation.DecodeProofData(recoveredMessage)
		if err != nil {
			log.Println("error decoding proof data:", err)
			respondDecode(w, nil, err)
			return
		}

		respondDecode(w, decodedData, nil)
	}
}
