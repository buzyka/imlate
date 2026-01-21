package util

import (
	"bytes"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewTestLogger() (*zap.SugaredLogger, *bytes.Buffer) {
	// Create a buffer to capture logs
    logBuffer := new(bytes.Buffer)

	encoderCfg := zap.NewDevelopmentEncoderConfig()
    core := zapcore.NewCore(
        zapcore.NewConsoleEncoder(encoderCfg),
        zapcore.AddSync(logBuffer),
        zap.DebugLevel,
    )
    return zap.New(core).Sugar(), logBuffer
}
