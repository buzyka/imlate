package isams

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type errReadCloser struct{}

func (e errReadCloser) Read(_ []byte) (int, error) {
	return 0, errors.New("read failed")
}

func (e errReadCloser) Close() error {
	return nil
}

func TestClientFactory_NewClient_Success(t *testing.T) {
	// Create a test server for OAuth2 token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"test_token","token_type":"Bearer","expires_in":3600}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	// Reset global tokenSource for test isolation
	tokenSource = nil

	factory := &ClientFactory{
		BaseURL:      tokenServer.URL,
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	ctx := context.Background()
	client, err := factory.NewClient(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, tokenServer.URL, client.BaseURL)
	assert.NotNil(t, client.HTTPClient)
	assert.Nil(t, client.Logger)
}

func TestClientFactory_NewClient_WithLogger(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"test_token","token_type":"Bearer","expires_in":3600}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	tokenSource = nil

	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		zapcore.DebugLevel,
	)).Sugar()

	factory := &ClientFactory{
		BaseURL:      tokenServer.URL,
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		Logger:       logger,
	}

	ctx := context.Background()
	client, err := factory.NewClient(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, logger, client.Logger)
}

func TestClientFactory_NewClient_TokenError(t *testing.T) {
	// Create a test server that returns error for token request
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer tokenServer.Close()

	// Reset global tokenSource for test isolation
	tokenSource = nil

	factory := &ClientFactory{
		BaseURL:      tokenServer.URL,
		ClientID:     "invalid_client",
		ClientSecret: "invalid_secret",
	}

	ctx := context.Background()
	client, err := factory.NewClient(ctx)

	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClientFactory_NewClient_TrimsTrailingSlash(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/connect/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"test_token","token_type":"Bearer","expires_in":3600}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tokenServer.Close()

	// Reset global tokenSource for test isolation
	tokenSource = nil

	factory := &ClientFactory{
		BaseURL:      tokenServer.URL + "/",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	ctx := context.Background()
	client, err := factory.NewClient(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, tokenServer.URL, client.BaseURL)
}

func TestClientFactory_getTokenSource_ReusesExistingTokenSource(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"test_token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	// Set a mock token source
	mockTokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "existing_token"})
	tokenSource = mockTokenSource

	factory := &ClientFactory{
		BaseURL:      tokenServer.URL,
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	ctx := context.Background()
	baseURL := tokenServer.URL
	tokenURL := baseURL + TokenEndpoint
	cfg := clientcredentials.Config{
		ClientID:     factory.ClientID,
		ClientSecret: factory.ClientSecret,
		TokenURL:     tokenURL,
	}

	// First call should reuse existing tokenSource
	ts := factory.getTokenSource(ctx, cfg, baseURL)
	assert.Equal(t, mockTokenSource, ts)

	// Reset for cleanup
	tokenSource = nil
}

func TestClientFactory_getTokenSource_CreatesNewTokenSource(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"test_token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	// Reset global tokenSource
	tokenSource = nil

	factory := &ClientFactory{
		BaseURL:      tokenServer.URL,
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
	}

	ctx := context.Background()
	baseURL := tokenServer.URL
	tokenURL := baseURL + TokenEndpoint
	cfg := clientcredentials.Config{
		ClientID:     factory.ClientID,
		ClientSecret: factory.ClientSecret,
		TokenURL:     tokenURL,
	}

	// Should create a new token source
	ts := factory.getTokenSource(ctx, cfg, baseURL)
	assert.NotNil(t, ts)
	assert.NotNil(t, tokenSource)

	// Reset for cleanup
	tokenSource = nil
}

func TestClient_Do_Success(t *testing.T) {
	// Create a test server for the actual request
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer testServer.Close()

	client := &Client{
		BaseURL:    testServer.URL,
		HTTPClient: &http.Client{},
	}

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/test", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
}

func newTestLogger(level zapcore.Level) (*zap.SugaredLogger, *bytes.Buffer) {
	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		level,
	)).Sugar()
	return logger, logBuffer
}

func TestClient_Do_LogsResultWithoutHeadersQueryOrBody(t *testing.T) {
	logger, logBuffer := newTestLogger(zapcore.DebugLevel)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Response-Secret", "response-secret")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"status":"slow down"}`))
	}))
	defer testServer.Close()

	client := &Client{
		BaseURL:    testServer.URL,
		HTTPClient: &http.Client{},
		Logger:     logger,
	}

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/api/students/123?secret=query-secret", strings.NewReader("payload"))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer very-secret-token")
	req.Header.Set("Cookie", "session=secret-cookie")
	req.Header.Set("X-API-Key", "secret-api-key")
	req.Header.Set("X-Custom-Secret", "custom-secret")

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	_ = resp.Body.Close()

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(logBuffer.Bytes(), &entry))
	assert.Equal(t, "isams request", entry["msg"])
	assert.Equal(t, "POST", entry["method"])
	assert.Equal(t, "/api/students/123", entry["path"])
	assert.Equal(t, float64(http.StatusTooManyRequests), entry["status"])
	assert.Equal(t, float64(len(`{"status":"slow down"}`)), entry["content_length"])
	assert.Contains(t, entry, "duration_ms")

	logOutput := logBuffer.String()
	for _, leaked := range []string{"very-secret-token", "secret-cookie", "secret-api-key", "custom-secret", "query-secret", "payload", "response-secret", "Authorization"} {
		assert.NotContains(t, logOutput, leaked)
	}
}

func TestClient_Do_LogsFailureWithoutURLQuery(t *testing.T) {
	logger, logBuffer := newTestLogger(zapcore.DebugLevel)
	client := &Client{HTTPClient: &http.Client{}, Logger: logger}

	req, err := http.NewRequest(http.MethodGet, "http://invalid-server-that-does-not-exist.test/api/students?secret=query-secret", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	var entry map[string]any
	assert.NoError(t, json.Unmarshal(logBuffer.Bytes(), &entry))
	assert.Equal(t, "isams request failed", entry["msg"])
	assert.Equal(t, "/api/students", entry["path"])
	assert.NotEmpty(t, entry["error"])
	assert.NotContains(t, logBuffer.String(), "query-secret")
}

func TestRequestErrorMessage(t *testing.T) {
	urlErr := &url.Error{Op: "Get", URL: "https://example.com/x?token=secret", Err: errors.New("connection refused")}
	assert.Equal(t, "Get: connection refused", requestErrorMessage(urlErr))
	assert.Equal(t, "plain error", requestErrorMessage(errors.New("plain error")))
}

func TestClient_logRequestResult_IsDebugLevel(t *testing.T) {
	logger, logBuffer := newTestLogger(zapcore.InfoLevel)
	client := &Client{Logger: logger}

	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/students"},
		Header: make(http.Header),
	}

	client.logRequestResult(req, &http.Response{StatusCode: http.StatusOK}, nil, time.Millisecond)

	assert.Empty(t, logBuffer.String())
}

func TestClient_Do_ErrorRequest(t *testing.T) {
	client := &Client{
		BaseURL:    "http://invalid-server-that-does-not-exist.test",
		HTTPClient: &http.Client{},
	}

	req, err := http.NewRequest(http.MethodGet, "http://invalid-server-that-does-not-exist.test/test", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClient_logRequestResult_DoesNotReadBody(t *testing.T) {
	logger, logBuffer := newTestLogger(zapcore.DebugLevel)
	client := &Client{Logger: logger}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/students"},
		Header: make(http.Header),
		Body:   errReadCloser{},
	}
	resp := &http.Response{StatusCode: http.StatusOK, ContentLength: -1, Body: errReadCloser{}}

	client.logRequestResult(req, resp, nil, time.Millisecond)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "isams request")
	assert.Contains(t, logOutput, `"host":"example.com"`)
	assert.NotContains(t, logOutput, "read failed")
}

func TestConstants(t *testing.T) {
	// Test endpoint constants
	assert.Equal(t, "/auth/connect/token", TokenEndpoint)
	assert.Equal(t, "/api/students", StudentsEndpoint)
	assert.Equal(t, "/api/students/{schoolId}", StudentByIDEndpoint)
	assert.Equal(t, "/api/registration/register/{registrationPeriodId}/students/{schoolId}", RegisterEndpoint)
	assert.Equal(t, "/api/registration/periods", RegistrationPeriodsEndpoint)
	assert.Equal(t, "/api/registration/register/{registrationPeriodId}/students/{schoolId}", RegistrationStatusEndpoint)
	assert.Equal(t, "/api/registration/absencecodes", AbsenceCodesEndpoint)

	// Test default registration period ID
	assert.Equal(t, "22947", defaultRegistrationPeriodID)
}
