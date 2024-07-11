package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestNewLoggerInDebugModeWillSetDebugConfiguration(t *testing.T) {
	logger := NewLogger(true)

	assert.NotNil(t, logger)
	assert.Equal(t, zapcore.DebugLevel, logger.Desugar().Level())
}

func TestNewLoggerInProductionModeWillSetProductionConfiguration(t *testing.T) {
	logger := NewLogger(false)

	assert.NotNil(t, logger)
	assert.Equal(t, zapcore.InfoLevel, logger.Desugar().Level())
}

func TestCreateSugarLoggerWillReturnNormalLogger(t *testing.T) {
	loggerCfg := zap.NewDevelopmentConfig()

	logger := createSugarLoggerForConfig(loggerCfg, nil)

	assert.NotNil(t, logger)
	assert.False(t, reflect.DeepEqual(logger.Desugar(), zap.NewNop()))
}

func TestCreateSugarLoggerForConfigWhenStandardLoggerNotBuiltWillReturnNopLogger(t *testing.T) {
	loggerCfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		Encoding:         "xml",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger := createSugarLoggerForConfig(loggerCfg, nil)

	assert.NotNil(t, logger)
	assert.True(t, reflect.DeepEqual(logger.Desugar(), zap.NewNop()))
}

func TestNewLoggingMiddlewareWillSetLoggerToTheContext(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	logger, err := zap.NewDevelopmentConfig().Build()
	assert.Nil(t, err)
	sLogger := logger.Sugar()
	_, okBefore := c.Request.Context().Value(loggerKey).(*zap.SugaredLogger)
	assert.False(t, okBefore)

	middleware := NewLoggingMiddleware(sLogger)
	middleware(c)

	l, ok := c.Request.Context().Value(loggerKey).(*zap.SugaredLogger)
	assert.True(t, ok)
	assert.NotNil(t, l)
}

func TestFallbackWithNotDefinedFallbackLoggerWillReturnExistingSugaredLogger(t *testing.T) {
	fallbackLoggerBackup := fallbackLogger
	restoreFallbackLogger := func() {
		fallbackLogger = fallbackLoggerBackup
	}
	defer restoreFallbackLogger()

	fallbackLogger = nil

	assert.Nil(t, fallbackLogger)
	assert.Equal(t, fallbackLogger, Fallback())
}

func TestFallbackWithDefinedFallbackLoggerWillReturnExistingSugaredLogger(t *testing.T) {
	exLogger := zaptest.NewLogger(t).Sugar()
	fallbackLoggerBackup := fallbackLogger
	restoreFallbackLogger := func() {
		fallbackLogger = fallbackLoggerBackup
	}
	defer restoreFallbackLogger()

	fallbackLogger = exLogger

	assert.Equal(t, exLogger, Fallback())
}

func TestFromContextWithDefinedLoggerInContextWillReturnLoggerFormContext(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.WithValue(context.Background(), loggerKey, logger)

	assert.Equal(t, logger, FromContext(ctx))
}

func TestFromContextWithNotDefinedLoggerInContextWillReturnFallbackLogger(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	ctx := context.Background()
	fallbackLoggerBackup := fallbackLogger
	restoreFallbackLogger := func() {
		fallbackLogger = fallbackLoggerBackup
	}
	defer restoreFallbackLogger()

	fallbackLogger = logger

	assert.Equal(t, logger, FromContext(ctx))
}

func TestTimeEncoder(t *testing.T) {
	encMock := &MockPrimitiveArrayEncoder{}
	encMock.On("AppendString", "2022-01-01T12:00:00Z").Once()
	encoder := timeEncoder()

	expectedTime := time.Date(2022, time.January, 1, 12, 0, 0, 0, time.UTC)
	encoder(expectedTime, encMock)
	encMock.AssertExpectations(t)
}

type MockPrimitiveArrayEncoder struct {
	mock.Mock
}

func (m *MockPrimitiveArrayEncoder) AppendBool(b bool) {
	m.Called(b)
}

func (m *MockPrimitiveArrayEncoder) AppendByteString(s []byte) {
	m.Called(s)
}

func (m *MockPrimitiveArrayEncoder) AppendComplex128(c complex128) {
	m.Called(c)
}

func (m *MockPrimitiveArrayEncoder) AppendComplex64(c complex64) {
	m.Called(c)
}

func (m *MockPrimitiveArrayEncoder) AppendFloat64(f float64) {
	m.Called(f)
}

func (m *MockPrimitiveArrayEncoder) AppendFloat32(f float32) {
	m.Called(f)
}

func (m *MockPrimitiveArrayEncoder) AppendInt(i int) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendInt64(i int64) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendInt32(i int32) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendInt16(i int16) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendInt8(i int8) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendString(s string) {
	m.Called(s)
}

func (m *MockPrimitiveArrayEncoder) AppendUint(i uint) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendUint64(i uint64) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendUint32(i uint32) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendUint16(i uint16) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendUint8(i uint8) {
	m.Called(i)
}

func (m *MockPrimitiveArrayEncoder) AppendUintptr(u uintptr) {
	m.Called(u)
}

func (m *MockPrimitiveArrayEncoder) AppendDuration(d time.Duration) {
	m.Called(d)
}

func (m *MockPrimitiveArrayEncoder) AppendTime(t time.Time) {
	m.Called(t)
}

func (m *MockPrimitiveArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *MockPrimitiveArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *MockPrimitiveArrayEncoder) AppendReflected(v interface{}) error {
	args := m.Called(v)
	return args.Error(0)
}
