// Package logger provides a logger implementation.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapLogger creates a new logger with the given log level.
//
// The logger is configured with the ProductionEncoder, which is a JSON
// encoder that outputs logs in the following format:
//
//	{
//	  "level": "INFO",
//	  "ts": 1598059584.776,
//	  "logger": "example",
//	  "caller": "example/main.go:11",
//	  "msg": "Hello, world!",
//	  "number": 42,
//	  "null": null,
//	  "object": {
//	    "bar": "baz"
//	  },
//	  "bool": true,
//	  "string": "hello"
//	}
//
// The log level must be one of: "debug", "info", "warn", "error", "fatal".
func NewZapLogger(level string) (*zap.Logger, error) {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, fmt.Errorf("zap.ParseAtomicLevel: %w", err)
	}

	logCfg := zap.NewProductionConfig()
	logCfg.DisableCaller = true
	logCfg.DisableStacktrace = true
	logCfg.Level = lvl
	logCfg.Encoding = "console"
	logCfg.EncoderConfig = encoderCfg

	return zap.Must(logCfg.Build()), nil
}
