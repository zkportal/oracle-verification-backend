package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"
)

type ReqContextValue string

const (
	ContextLogger      ReqContextValue = "log"
	ContextRequestID   ReqContextValue = "req_id"
	ContextHandlerName ReqContextValue = "handler"
)

func GetContextLogger(ctx context.Context) *log.Logger {
	logVal := ctx.Value(ContextLogger)
	if logVal == nil {
		log.Println("Warning: expected to find logger in request context")
		return log.Default()
	}

	logger, ok := logVal.(*log.Logger)
	if !ok {
		log.Println("Warning: expected to find logger in request context")
		return log.Default()
	}

	return logger
}

func GetContextRequestId(ctx context.Context) string {
	reqIdVal := ctx.Value(ContextRequestID)
	if reqIdVal == nil {
		log.Println("Warning: expected to find request ID in request context")
		return ""
	}

	reqId, ok := reqIdVal.(string)
	if !ok {
		log.Println("Warning: expected to find request ID in request context")
		return ""
	}

	return reqId
}

func GetContextHandlerName(ctx context.Context) string {
	handlerNameVal := ctx.Value(ContextHandlerName)
	if handlerNameVal == nil {
		log.Println("Warning: expected to find handler name in request context")
		return ""
	}

	handlerName, ok := handlerNameVal.(string)
	if !ok {
		log.Println("Warning: expected to find handler name in request context")
		return ""
	}

	return handlerName
}

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

type capturingResponseWriter struct {
	http.ResponseWriter

	statusCode  int
	wroteHeader bool
}

func (crw *capturingResponseWriter) WriteHeader(code int) {
	if crw.wroteHeader {
		return
	}

	crw.statusCode = code
	crw.wroteHeader = true
	crw.ResponseWriter.WriteHeader(code)
}

func (crw *capturingResponseWriter) Write(body []byte) (int, error) {
	if !crw.wroteHeader {
		crw.statusCode = http.StatusOK
		crw.wroteHeader = true
	}

	return crw.ResponseWriter.Write(body)
}

func LogAndTraceMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqIdBuf := make([]byte, 16)
		_, err := rand.Read(reqIdBuf)
		if err != nil {
			log.Println("failed to create random request hash, falling back to simple hash")
			timestamp := time.Now().Unix()
			binary.LittleEndian.PutUint64(reqIdBuf, uint64(timestamp))
			hash := sha256.Sum256(reqIdBuf)
			reqIdBuf = hash[:16]
		}
		requestId := hex.EncodeToString(reqIdBuf)

		handlerName := r.URL.Path
		logger := log.New(os.Stderr, fmt.Sprintf("%s: %s: ", requestId, handlerName), log.LstdFlags|log.Lmsgprefix)

		ctx := context.WithValue(r.Context(), ContextLogger, logger)
		ctx = context.WithValue(ctx, ContextRequestID, requestId)
		ctx = context.WithValue(ctx, ContextHandlerName, handlerName)

		crw := &capturingResponseWriter{ResponseWriter: w}

		next.ServeHTTP(crw, r.WithContext(ctx))

		handleVerb := "finished"
		if crw.statusCode != http.StatusOK {
			handleVerb = "failed"
		}

		logger.Println(handleVerb, "HTTP", crw.statusCode)
	}
}
