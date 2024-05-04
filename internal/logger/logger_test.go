package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewZapLogger(t *testing.T) {
	logger, err := NewZapLogger("debug")
	require.NoError(t, err)
	require.NotNil(t, logger)
}

func TestNewZapLoggerInvalid(t *testing.T) {
	_, err := NewZapLogger("invalid")
	require.Error(t, err)
}
