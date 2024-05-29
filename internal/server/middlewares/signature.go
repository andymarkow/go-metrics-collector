package middlewares

import (
	"bytes"
	"crypto/hmac"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/signature"
	"go.uber.org/zap"
)

// type hashResponseWriter struct {
// 	w    http.ResponseWriter
// 	body *bytes.Buffer
// }

// func newHashResponseWriter(w http.ResponseWriter) *hashResponseWriter {
// 	return &hashResponseWriter{
// 		w:    w,
// 		body: new(bytes.Buffer),
// 	}
// }

// func (h *hashResponseWriter) Write(b []byte) (int, error) {

// 	bw, err := h.body.Write(b)
// 	if err != nil {
// 		return 0, err
// 	}

// 	fmt.Printf("body: %v\n", h.body.String())

// 	// Compute the hash sum of the captured response body
// 	hash := sha256.New()
// 	if _, err := io.Copy(hash, bytes.NewReader(b)); err != nil {
// 		// m.log.Error("compute hash", zap.Error(err))
// 		// http.Error(w, err.Error(), http.StatusInternalServerError)

// 		log.Printf("compute hash: %v\n", err)

// 		// return
// 	}

// 	hashSum := hash.Sum(nil)
// 	hashHex := hex.EncodeToString(hashSum)

// 	// Set the hash in the response headers
// 	h.w.Header().Set("HashSHA256", hashHex)

// 	fmt.Printf("Hash: %s\n", hashHex)
// 	fmt.Printf("body: %v\n", h.body.String())

// 	return bw, nil
// }

// func (h *hashResponseWriter) WriteHeader(statusCode int) {
// 	h.w.WriteHeader(statusCode)
// }

// func (h *hashResponseWriter) Header() http.Header {
// 	return h.w.Header()
// }

func (m *Middlewares) HashSumValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			m.log.Error("read body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		r.Body = io.NopCloser(bytes.NewBuffer(body))

		sign, err := signature.CalculateHashSum(m.signKey, body)
		if err != nil {
			m.log.Error("calculate signature", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		m.log.Debug("signature orig", zap.Any("sign", sign))

		signHeader, err := hex.DecodeString(r.Header.Get("HashSHA256"))
		if err != nil {
			m.log.Error("decode signature", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		m.log.Debug("signature head", zap.Any("sign", signHeader))

		if !hmac.Equal(sign, signHeader) {
			m.log.Error("signature mismatch", zap.Error(errormsg.ErrHashSumValueMismatch))
			http.Error(w, errormsg.ErrHashSumValueMismatch.Error(), http.StatusBadRequest)

			return
		}

		next.ServeHTTP(w, r)
	})
}
