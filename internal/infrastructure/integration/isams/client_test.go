package isams

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

func TestClient_Do_RedactsSensitiveHeadersInLogs(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		zapcore.DebugLevel,
	)).Sugar()

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer testServer.Close()

	client := &Client{
		BaseURL:    testServer.URL,
		HTTPClient: &http.Client{},
		Logger:     logger,
	}

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/test", strings.NewReader("payload"))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer very-secret-token")
	req.Header.Set("Cookie", "session=secret-cookie")
	req.Header.Set("X-API-Key", "secret-api-key")

	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	_ = resp.Body.Close()

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "isams outgoing request")
	assert.Contains(t, logOutput, "REDACTED")
	assert.NotContains(t, logOutput, "very-secret-token")
	assert.NotContains(t, logOutput, "secret-cookie")
	assert.NotContains(t, logOutput, "secret-api-key")
	assert.Contains(t, logOutput, "payload")
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

func TestClient_logOutgoingRequest_BodyReadError(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		zapcore.DebugLevel,
	)).Sugar()

	client := &Client{Logger: logger}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/students"},
		Header: make(http.Header),
		Body:   errReadCloser{},
	}

	client.logOutgoingRequest(req)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "isams outgoing request")
	assert.Contains(t, logOutput, "dumpError")
	assert.Contains(t, logOutput, "read failed")
}

func TestClient_logOutgoingRequest_DumpRequestOutError(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(logBuffer),
		zapcore.DebugLevel,
	)).Sugar()

	client := &Client{Logger: logger}

	req := &http.Request{
		Method: "BAD METHOD",
		URL:    &url.URL{Scheme: "https", Host: "example.com", Path: "/api/students"},
		Header: make(http.Header),
	}

	client.logOutgoingRequest(req)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "isams outgoing request")
	assert.Contains(t, logOutput, "dumpError")
	assert.Contains(t, logOutput, "invalid method")
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
