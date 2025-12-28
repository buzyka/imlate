package isams

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_GetStudents_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/students", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "10", r.URL.Query().Get("pageSize"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"count": 1,
			"page": 1,
			"pageSize": 10,
			"totalCount": 1,
			"totalPages": 1,
			"students": [
				{
					"id": 123,
					"fullName": "John Doe",
					"schoolId": "S123"
				}
			]
		}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	resp, err := client.GetStudents(1, 10)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Count)
	assert.Len(t, resp.Students, 1)
	assert.Equal(t, int64(123), resp.Students[0].ID)
	assert.Equal(t, "John Doe", *resp.Students[0].FullName)
}

func TestClient_GetStudents_HTTPError(t *testing.T) {
	client := &Client{
		BaseURL:    "http://invalid-url",
		HTTPClient: &http.Client{},
	}

	resp, err := client.GetStudents(1, 10)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClient_GetStudents_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	resp, err := client.GetStudents(1, 10)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get students")
}

func TestClient_GetStudents_JSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	resp, err := client.GetStudents(1, 10)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClient_GetStudents_NewRequestError(t *testing.T) {
	// To trigger http.NewRequest error, we can use a control character in the URL
	client := &Client{
		BaseURL:    "http://example.com" + string(rune(0x7f)),
		HTTPClient: &http.Client{},
	}

	resp, err := client.GetStudents(1, 10)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestClient_GetStudentByID_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/students/123", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 123,
			"fullName": "John Doe",
			"schoolId": "S123"
		}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	student, err := client.GetStudentByID(123)
	assert.NoError(t, err)
	assert.NotNil(t, student)
	assert.Equal(t, int64(123), student.ID)
	assert.Equal(t, "John Doe", *student.FullName)
}

func TestClient_GetStudentByID_HTTPError(t *testing.T) {
	client := &Client{
		BaseURL:    "http://invalid-url",
		HTTPClient: &http.Client{},
	}

	student, err := client.GetStudentByID(123)
	assert.Error(t, err)
	assert.Nil(t, student)
}

func TestClient_GetStudentByID_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	student, err := client.GetStudentByID(123)
	assert.Error(t, err)
	assert.Nil(t, student)
	assert.Contains(t, err.Error(), "failed to get student")
}

func TestClient_GetStudentByID_JSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	student, err := client.GetStudentByID(123)
	assert.Error(t, err)
	assert.Nil(t, student)
}

func TestClient_GetStudentByID_NewRequestError(t *testing.T) {
	client := &Client{
		BaseURL:    "http://example.com" + string(rune(0x7f)),
		HTTPClient: &http.Client{},
	}

	student, err := client.GetStudentByID(123)
	assert.Error(t, err)
	assert.Nil(t, student)
}
