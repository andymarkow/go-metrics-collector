package middlewares

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/andymarkow/go-metrics-collector/internal/cryptutils"
)

func (m *Middlewares) Cryptography(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			m.log.Error("read body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		m.log.Debug("encrypted request body", zap.Any("body", body))

		decryptedBody, err := cryptutils.DecryptOAEP(sha256.New(), rand.Reader, m.cryptoPrivKey, body, nil)
		if err != nil {
			m.log.Error("failed to decrypt body", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		m.log.Debug("decrypted request body", zap.Any("body", decryptedBody))

		r.Body = io.NopCloser(bytes.NewBuffer(decryptedBody))

		next.ServeHTTP(w, r)
	})
}
