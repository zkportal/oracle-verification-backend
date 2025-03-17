package handlers

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zkportal/oracle-verification-backend/attestation/nitro"
	"github.com/zkportal/oracle-verification-backend/u128"
)

type infoHandler struct {
	uniqueId         string
	pcrValues        [3]string
	liveCheckProgram string
	startTime        time.Time
}

func CreateInfoHandler(uniqueId string, pcrValues [3]string, liveCheckProgram string) http.Handler {
	return &infoHandler{
		uniqueId:         uniqueId,
		pcrValues:        pcrValues,
		liveCheckProgram: liveCheckProgram,
		startTime:        time.Now().UTC(),
	}
}

type uniqueIdInfo struct {
	Hex    string `json:"hexEncoded"`
	Base64 string `json:"base64Encoded"`
	Aleo   string `json:"aleoEncoded"`
}

type pcrValuesInfo struct {
	Hex    [3]string `json:"hexEncoded"`
	Base64 [3]string `json:"base64Encoded"`
	Aleo   string    `json:"aleoEncoded"`
}

type InfoResponse struct {
	TargetUniqueId   uniqueIdInfo  `json:"targetUniqueId"`
	TargetPcrValues  pcrValuesInfo `json:"targetPcrValues"`
	LiveCheckProgram string        `json:"liveCheckProgram"`
	StartTime        string        `json:"startTimeUTC"`
}

func (h *infoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log := GetContextLogger(req.Context())

	response := new(InfoResponse)

	uniqueIdBytes, _ := hex.DecodeString(h.uniqueId)

	uniqueIdAleo1, _ := u128.SliceToU128(uniqueIdBytes[0:16])
	uniqueIdAleo2, _ := u128.SliceToU128(uniqueIdBytes[16:32])

	response.TargetUniqueId = uniqueIdInfo{
		Hex:    h.uniqueId,
		Base64: base64.StdEncoding.EncodeToString(uniqueIdBytes),
		Aleo:   fmt.Sprintf("{ chunk_1: %su128, chunk_2: %su128 }", uniqueIdAleo1.String(), uniqueIdAleo2.String()),
	}

	var pcrBytes [3][48]byte

	for idx, pcr := range h.pcrValues {
		buf, _ := hex.DecodeString(pcr)
		pcrBytes[idx] = ([48]byte)(buf)
	}

	response.TargetPcrValues = pcrValuesInfo{
		Hex: h.pcrValues,
		Base64: [3]string{
			base64.StdEncoding.EncodeToString(pcrBytes[0][:]),
			base64.StdEncoding.EncodeToString(pcrBytes[1][:]),
			base64.StdEncoding.EncodeToString(pcrBytes[2][:]),
		},
		Aleo: nitro.FormatPcrValues(pcrBytes),
	}

	response.LiveCheckProgram = h.liveCheckProgram
	response.StartTime = h.startTime.Format(time.DateTime)

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Println("failed to marshal response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(responseBody)
	if err != nil {
		log.Println("failed to write response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
