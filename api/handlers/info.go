package handlers

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/zkportal/oracle-verification-backend/u128"
)

type infoHandler struct {
	uniqueId         string
	liveCheckProgram string
	startTime        time.Time
}

func CreateInfoHandler(uniqueId string, liveCheckProgram string) http.Handler {
	return &infoHandler{
		uniqueId:         uniqueId,
		liveCheckProgram: liveCheckProgram,
		startTime:        time.Now().UTC(),
	}
}

type uniqueIdInfo struct {
	Hex    string    `json:"hexEncoded"`
	Base64 string    `json:"base64Encoded"`
	Aleo   [2]string `json:"aleoEncoded"`
}

type InfoResponse struct {
	TargetUniqueId   uniqueIdInfo `json:"targetUniqueId"`
	LiveCheckProgram string       `json:"liveCheckProgram"`
	StartTime        string       `json:"startTimeUTC"`
}

func (h *infoHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log.Println("handling /info")

	response := new(InfoResponse)

	uniqueIdBytes, _ := hex.DecodeString(h.uniqueId)

	uniqueIdAleo1, _ := u128.SliceToU128(uniqueIdBytes[0:16])
	uniqueIdAleo2, _ := u128.SliceToU128(uniqueIdBytes[16:32])

	response.TargetUniqueId = uniqueIdInfo{
		Hex:    h.uniqueId,
		Base64: base64.StdEncoding.EncodeToString(uniqueIdBytes),
		Aleo: [2]string{
			uniqueIdAleo1.String() + "u128",
			uniqueIdAleo2.String() + "u128",
		},
	}
	response.LiveCheckProgram = h.liveCheckProgram
	response.StartTime = h.startTime.Format(time.DateTime)

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Println("failed to marshal response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(responseBody)
}
