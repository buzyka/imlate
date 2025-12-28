package isams

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_GetYearGroupDivisions_Success(t *testing.T) {
	yearGroupID := int32(10)
	expectedPath := fmt.Sprintf("/api/school/yeargroups/%d/divisions", yearGroupID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedPath, r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"divisions": [
				{
					"id": 1,
					"code": "DIV1",
					"lastUpdated": "2023-01-01T00:00:00Z",
					"name": "Division 1",
					"order": 1
				}
			]
		}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	resp, err := client.GetYearGroupDivisions(yearGroupID)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Divisions, 1)
	assert.Equal(t, int32(1), resp.Divisions[0].ID)
	assert.Equal(t, "DIV1", resp.Divisions[0].Code)
	assert.Equal(t, "Division 1", resp.Divisions[0].Name)
}

func TestClient_GetYearGroupsDivisions_HTTPError(t *testing.T) {
	client := &Client{
		BaseURL:    "http://invalid-url",
		HTTPClient: &http.Client{},
	}

	resp, err := client.GetYearGroupDivisions(10)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClient_GetYearGroupsDivisions_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	resp, err := client.GetYearGroupDivisions(10)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get year group divisions")
}

func TestClient_GetYearGroupsDivisions_JSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	resp, err := client.GetYearGroupDivisions(10)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClient_GetYearGroupsDivisions_NewRequestError(t *testing.T) {
	// To trigger http.NewRequest error, we can use a control character in the URL
	client := &Client{
		BaseURL:    "http://example.com" + string(rune(0x7f)),
		HTTPClient: &http.Client{},
	}

	resp, err := client.GetYearGroupDivisions(10)
	assert.Error(t, err)
	assert.Nil(t, resp)
}
