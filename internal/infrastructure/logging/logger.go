package logging

import (
	"context"
	"github.com/gin-gonic/gin"
	"time"

	"go.uber.org/zap/zapcore"

	"go.uber.org/zap"
)

// contextKey is a private string type to prevent collisions in the context map.
type contextKey string

// loggerKey points to the value in the context where the logging is stored.
const loggerKey = contextKey("logging")

var fallbackLogger *zap.SugaredLogger

func NewLogger(debug bool) *zap.SugaredLogger {
	var loggerCfg zap.Config
	var loggerBuildOptions []zap.Option

	if debug {
		loggerCfg = zap.NewDevelopmentConfig()
		loggerBuildOptions = append(loggerBuildOptions, zap.AddStacktrace(zapcore.WarnLevel))
	} else {
		loggerCfg = zap.NewProductionConfig()
		loggerBuildOptions = append(loggerBuildOptions, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	loggerCfg.EncoderConfig.EncodeTime = timeEncoder()
	loggerCfg.EncoderConfig.MessageKey = "message"
	loggerCfg.EncoderConfig.TimeKey = "timestamp"
	loggerCfg.EncoderConfig.EncodeDuration = zapcore.NanosDurationEncoder
	loggerCfg.EncoderConfig.StacktraceKey = "error.stack"
	loggerCfg.EncoderConfig.FunctionKey = "logging.method_name"

	return createSugarLoggerForConfig(loggerCfg, loggerBuildOptions)
}

func createSugarLoggerForConfig(loggerCfg zap.Config, buildOptions []zap.Option) *zap.SugaredLogger {
	logger, err := loggerCfg.Build(buildOptions...)
	if err != nil {
		logger = zap.NewNop()
	}

	return logger.Sugar()
}

// timeEncoder encodes the time as RFC3339 nano.
func timeEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(time.RFC3339Nano))
	}
}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey).(*zap.SugaredLogger); ok {
		return logger
	}
	return Fallback()
}

func Fallback() *zap.SugaredLogger {
	if fallbackLogger == nil {
		loggerCfg := zap.NewProductionConfig()
		logger, _ := loggerCfg.Build()

		fallbackLogger = logger.Sugar()
	}

	return fallbackLogger
}

func NewLoggingMiddleware(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		updatedCtx := context.WithValue(c.Request.Context(), loggerKey, logger)
		c.Request = c.Request.WithContext(updatedCtx)
	}
}
