// Package cryptutils provides a set of encryption/decryption functions.
package cryptutils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash"
	"io"
	"os"
)

// LoadRSAPublicKey loads an RSA public key from a file.
func LoadRSAPublicKey(keyfile string) (*rsa.PublicKey, error) {
	keyPEM, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("reading key file: %w", err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key with block type: %s", block.Type)
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	rsaPubKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("the key is not a RSA public key")
	}

	return rsaPubKey, nil
}

// LoadRSAPrivateKey loads an RSA private key from a file.
func LoadRSAPrivateKey(keyfile string) (*rsa.PrivateKey, error) {
	keyPEM, err := os.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("reading key file: %w", err)
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing private key with block type: %s", block.Type)
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	rsaPrivKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("the key is not a RSA private key")
	}

	return rsaPrivKey, nil
}

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
