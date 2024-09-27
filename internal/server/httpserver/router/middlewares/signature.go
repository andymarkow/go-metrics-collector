package middlewares

import (
	"bytes"
	"crypto/hmac"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/errormsg"
	"github.com/andymarkow/go-metrics-collector/internal/signature"
)

// HashSumValidator is a router middleware that validates the hash sum of the request body.
//
// The middleware expects the hash sum to be passed in the "HashSHA256" header.
// The hash sum is calculated using the SHA-256 algorithm and the given sign key.
//
// If the hash sum is invalid or the header is missing, the middleware returns a 400 status code.
func (m *Middlewares) HashSumValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			m.log.Error("read body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				m.log.Error("failed to close request body", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}
		}()

		r.Body = io.NopCloser(bytes.NewBuffer(body))

		sign, err := signature.CalculateHashSum(m.signKey, body)
		if err != nil {
			m.log.Error("calculate signature", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		m.log.Debug("body payload calculated signature", zap.Any("hashsum", sign))

		headerHashSum := r.Header.Get("HashSHA256") //nolint:canonicalheader,nolintlint

		signHeader, err := hex.DecodeString(headerHashSum)
		if err != nil {
			m.log.Error("decode signature", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		m.log.Debug("body payload provided signature", zap.Any("hashsum", signHeader))

		m.log.Debug("encoded body payload signature", zap.Any("hashsum", headerHashSum))

		if !hmac.Equal(sign, signHeader) {
			m.log.Error("signature mismatch", zap.Error(errormsg.ErrHashSumValueMismatch))
			http.Error(w, errormsg.ErrHashSumValueMismatch.Error(), http.StatusBadRequest)

			return
		}

		next.ServeHTTP(w, r)
	})
}
