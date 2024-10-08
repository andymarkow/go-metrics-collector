// Package tlsutils provides TLS utilities.
package tlsutils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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
