package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/zkportal/oracle-verification-backend/reproducibleEnclave"

	"github.com/zkportal/oracle-verification-backend/contract"

	"github.com/zkportal/oracle-verification-backend/config"

	"github.com/zkportal/oracle-verification-backend/api"

	aleo_internal "github.com/zkportal/aleo-utils-go"
)

const (
	IdleTimeout      = 30
	ReadWriteTimeout = 5
)

func main() {
	confContent, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalln(err)
	}

	conf, err := config.LoadConfig(confContent)
	if err != nil {
		log.Fatalln(err)
	}

	if conf.UniqueIdTarget == "" {
		log.Println("Unique ID verification target is not provided (\"uniqueIdTarget\" in config.json), reproducing Aleo Oracle backend build")
		expectedUniqueId, err := reproducibleEnclave.GetOracleReproducibleUniqueID()
		if err != nil {
			log.Fatalln(err)
		}

		conf.UniqueIdTarget = expectedUniqueId
	}

	if !conf.LiveCheck.Skip {
		log.Println("Requesting unique ID from", conf.LiveCheck.ContractName, "using", conf.LiveCheck.ApiBaseUrl)
		liveUniqueId, err := contract.GetUniqueIDAssert(conf.LiveCheck.ApiBaseUrl, conf.LiveCheck.ContractName)
		if err != nil {
			log.Fatalln("Failed to fetch live contract's unique ID assertion:", err)
		}

		log.Printf("Fetched unique ID assert from %s: %s", conf.LiveCheck.ContractName, liveUniqueId)

		if liveUniqueId != conf.UniqueIdTarget {
			log.Fatalf("Reproducible build of the oracle backend produced a different unique ID than the live contract.\nLive unique ID: %s\nReproduced unique ID: %s\n", liveUniqueId, conf.UniqueIdTarget)
		}
	} else {
		log.Println("WARNING: skipping Aleo live contract unique ID check")
	}

	log.Println("Expecting Aleo Oracle backend to have unique ID:", conf.UniqueIdTarget)

	aleo, close, err := aleo_internal.NewWrapper()
	if err != nil {
		log.Fatalln("Failed to initialize Aleo signer:", err)
	}
	defer close()

	mux := api.CreateApi(aleo, conf)

	bindAddr := fmt.Sprintf(":%d", conf.Port)

	server := &http.Server{
		IdleTimeout:       time.Second * IdleTimeout,
		ReadHeaderTimeout: time.Second * ReadWriteTimeout,
		WriteTimeout:      time.Second * ReadWriteTimeout,
		Addr:              bindAddr,
		Handler:           mux,
	}

	if conf.UseTls {
		tlsConf := new(tls.Config)
		tlsConf.Certificates = make([]tls.Certificate, 1)

		tlsConf.Certificates[0], err = tls.LoadX509KeyPair(conf.TlsCertFile, conf.TlsKeyFile)
		if err != nil {
			log.Fatalln(err)
		}

		tlsConf.CurvePreferences = []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		}

		tlsConf.MinVersion = tls.VersionTLS12

		tlsConf.CipherSuites = []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}

		server.TLSConfig = tlsConf

		log.Printf("oracle-verification-backend: starting https server on %s\n", bindAddr)
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Printf("oracle-verification-backend: starting http server on %s\n", bindAddr)
		log.Fatalln(server.ListenAndServe())
	}
}
