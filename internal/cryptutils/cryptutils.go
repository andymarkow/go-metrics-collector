// Package cryptutils provides a set of encryption/decryption functions.
package cryptutils

import (
	"crypto/rsa"
	"fmt"
	"hash"
	"io"
)

// EncryptOAEP encrypts data using RSA-OAEP encryption method.
func EncryptOAEP(hash hash.Hash, random io.Reader, key *rsa.PublicKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)

	chunkSize := key.Size() - 2*hash.Size() - 2

	encryptedChunks := make([]byte, 0)

	for i := 0; i < msgLen; i += chunkSize {
		end := i + chunkSize
		if end > msgLen {
			end = msgLen
		}

		encryptedChunk, err := rsa.EncryptOAEP(hash, random, key, msg[i:end], label)
		if err != nil {
			return nil, fmt.Errorf("rsa.EncryptOAEP: %w", err)
		}

		encryptedChunks = append(encryptedChunks, encryptedChunk...)
	}

	return encryptedChunks, nil
}

// DecryptOAEP decrypts data using RSA-OAEP decryption method.
func DecryptOAEP(hash hash.Hash, random io.Reader, key *rsa.PrivateKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)

	chunkSize := key.PublicKey.Size()

	decryptedChunks := make([]byte, 0)

	for i := 0; i < msgLen; i += chunkSize {
		end := i + chunkSize
		if end > msgLen {
			end = msgLen
		}

		decryptedChunk, err := rsa.DecryptOAEP(hash, random, key, msg[i:end], label)
		if err != nil {
			return nil, fmt.Errorf("rsa.DecryptOAEP: %w", err)
		}

		decryptedChunks = append(decryptedChunks, decryptedChunk...)
	}

	return decryptedChunks, nil
}
