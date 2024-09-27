// Package middlewares provides router middlewares.
package middlewares

import (
	"crypto/rsa"

	"go.uber.org/zap"
)

// Middlewares is a collection of router middlewares.
type Middlewares struct {
	log           *zap.Logger
	cryptoPrivKey *rsa.PrivateKey
	signKey       []byte
}

// New creates new Middlewares instance.
func New(opts ...Option) *Middlewares {
	// Default Middleware options.
	mw := &Middlewares{
		log: zap.Must(zap.NewDevelopment()),
	}

	// Apply options
	for _, opt := range opts {
		opt(mw)
	}

	return mw
}

// Option is a router middleware option.
type Option func(m *Middlewares)

// WithLogger is a router middleware option that sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(m *Middlewares) {
		m.log = logger
	}
}

// WithSignKey is a router middleware option that sets sign key.
func WithSignKey(signKey []byte) Option {
	return func(m *Middlewares) {
		m.signKey = signKey
	}
}

func WithCryptoPrivateKey(key *rsa.PrivateKey) Option {
	return func(m *Middlewares) {
		m.cryptoPrivKey = key
	}
}
