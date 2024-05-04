package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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
