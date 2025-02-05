package api

import (
	"net/http"

	"github.com/zkportal/oracle-verification-backend/api/handlers"
	"github.com/zkportal/oracle-verification-backend/config"

	aleo_wrapper "github.com/zkportal/aleo-utils-go"

	"github.com/rs/cors"
)

func CreateApi(aleoWrapper aleo_wrapper.Wrapper, conf *config.Configuration) http.Handler {
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodPost},
	})

	addMiddleware := func(h http.Handler) http.Handler {
		return handlers.LogAndTraceMiddleware(handlers.PanicMiddleware(corsMiddleware.Handler(handlers.HeaderMiddleware(h))))
	}

	mux := http.NewServeMux()

	targetPcrs := [3]string{conf.PcrValuesTarget[0], conf.PcrValuesTarget[1], conf.PcrValuesTarget[2]}

	mux.Handle("/info", addMiddleware(handlers.CreateInfoHandler(conf.UniqueIdTarget, targetPcrs, conf.LiveCheck.ContractName)))
	mux.Handle("/verify", addMiddleware(handlers.CreateVerifyHandler(aleoWrapper, conf.UniqueIdTarget, targetPcrs)))
	mux.Handle("/decode", addMiddleware(handlers.CreateDecodeHandler(aleoWrapper)))

	return mux
}
