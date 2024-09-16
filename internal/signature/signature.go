// Package signature provides functions to calculate SHA256 hash sum with a key.
package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// CalculateHashSum calculate SHA256 hash sum with a key.
func CalculateHashSum(key, payload []byte) ([]byte, error) {
	h := hmac.New(sha256.New, key)

	if _, err := h.Write(payload); err != nil {
		return nil, fmt.Errorf("hmac.Write: %w", err)
	}

	return h.Sum(nil), nil
}
