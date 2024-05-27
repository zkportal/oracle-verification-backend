package api

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/zkportal/oracle-verification-backend/config"

	"github.com/zkportal/oracle-verification-backend/api/handlers"

	aleo_signer "github.com/zkportal/aleo-utils-go"

	"github.com/rs/cors"
)

func HeaderMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		next.ServeHTTP(w, r)
	}
}

func PanicMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				log.Println("Panic:", err)
				log.Printf("%s", debug.Stack())

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}
}

func CreateApi(signer aleo_signer.Wrapper, conf *config.Configuration) http.Handler {
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodPost},
	})

	addMiddleware := func(h http.Handler) http.Handler {
		return PanicMiddleware(corsMiddleware.Handler(HeaderMiddleware(h)))
	}

	mux := http.NewServeMux()

	mux.Handle("/info", addMiddleware(handlers.CreateInfoHandler(conf.UniqueIdTarget, conf.LiveCheck.ContractName)))
	mux.Handle("/verify", addMiddleware(handlers.CreateVerifyHandler(signer, conf.UniqueIdTarget)))
	mux.Handle("/decode", addMiddleware(handlers.CreateDecodeHandler(signer)))
	mux.Handle("/decodeReport", addMiddleware(handlers.CreateDecodeVerifyHandler(signer, conf.UniqueIdTarget)))

	return mux
}
