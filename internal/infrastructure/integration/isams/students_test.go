package isams

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testDomain = "https://example.test"

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

func TestClient_GetStudentPhoto_Success(t *testing.T) {
	var tests = []struct {
		name		   string
		contentType   string
		expectedExt   string
	} {
		{"JPEG Image", "image/jpeg", "jpg"},
		{"PNG Image", "image/png", "png"},
		{"GIF Image", "image/gif", "gif"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schoolId := "123"
			testBody := []byte("test image content")
			httpClient := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, fmt.Sprintf("%s/api/students/%s/photos/current", testDomain, schoolId), r.URL.String())
					assert.Equal(t, "*/*", r.Header.Get("Accept"))
					
					return testResponse(t, http.StatusOK, WithBodyBinary(testBody), WithHeader("Content-Type", tt.contentType)), nil
				}),
			}

			client := &Client{
				BaseURL:    testDomain,
				HTTPClient: httpClient,
			}
			
			photoResp, err := client.GetStudentPhoto(schoolId)
			assert.NoError(t, err)
			assert.NotNil(t, photoResp)
			assert.Equal(t, testBody, photoResp.Data)
			assert.Equal(t, tt.contentType, photoResp.ContentType)
			assert.Equal(t, tt.expectedExt, photoResp.Extension)
		})
	}
}


type tErrReadCloser struct{ err error }

func (e tErrReadCloser) Read(p []byte) (n int, err error) { return 0, e.err }
func (e tErrReadCloser) Close() error               { return nil }

func TestClient_GetStudentPhoto_HTTPClientError(t *testing.T) {
	schoolId := "321"

	client := &Client{
		BaseURL:    testDomain,
	}

	t.Run("HTTP Client Error", func(t *testing.T){
		httpClient := &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return testResponse(t, http.StatusNotFound), assert.AnError
			}),
		}

		client.HTTPClient = httpClient

		photoResp, err := client.GetStudentPhoto(schoolId)
		assert.Error(t, err)
		assert.Nil(t, photoResp)
	})

	t.Run("Non-200 Status Code", func(t *testing.T){
		httpClient := &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return testResponse(t, http.StatusInternalServerError), nil
			}),
		}

		client.HTTPClient = httpClient

		photoResp, err := client.GetStudentPhoto(schoolId)
		assert.Error(t, err)
		assert.Nil(t, photoResp)
	})

	t.Run("404 Status Code", func(t *testing.T){
		httpClient := &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return testResponse(t, http.StatusNotFound), nil
			}),
		}

		client.HTTPClient = httpClient

		_, err := client.GetStudentPhoto(schoolId)
		assert.ErrorIs(t, err, ErrStudentPhotoNotFound)
	})

	t.Run("Response Body Read Error", func(t *testing.T){
		httpClient := &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				resp := testResponse(t, http.StatusOK)
				resp.Body = tErrReadCloser{err: assert.AnError}
				return resp, nil
			}),
		}

		client.HTTPClient = httpClient

		_, err := client.GetStudentPhoto(schoolId)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAPIResponseBody)
	})
}
