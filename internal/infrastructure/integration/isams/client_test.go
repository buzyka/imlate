package isams

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

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
