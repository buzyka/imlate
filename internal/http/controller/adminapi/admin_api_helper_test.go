package adminapi

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadUploadedFormFile_MissingFile(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/upload", bytes.NewBuffer(nil))
	c.Request.Header.Set("Content-Type", "multipart/form-data; boundary=abc")

	filename, data, ok := readUploadedFormFile(c, "image", maxVisitorImageUploadSizeBytes)

	assert.False(t, ok)
	assert.Empty(t, filename)
	assert.Nil(t, data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "image file is required")
}

func TestReadUploadedFormFile_FileTooLarge(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "too-large.jpg")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("a"), int(maxVisitorImageUploadSizeBytes)+1))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c.Request = httptest.NewRequest(http.MethodPost, "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	filename, data, ok := readUploadedFormFile(c, "image", maxVisitorImageUploadSizeBytes)

	assert.False(t, ok)
	assert.Empty(t, filename)
	assert.Nil(t, data)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "max 5MB")
}

func TestReadUploadedFormFile_OpenFileError(t *testing.T) {
	origOpen := openUploadedFile
	t.Cleanup(func() {
		openUploadedFile = origOpen
	})
	openUploadedFile = func(_ *multipart.FileHeader) (multipart.File, error) {
		return nil, errors.New("open error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "ok.jpg")
	require.NoError(t, err)
	_, err = part.Write([]byte("abc"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c.Request = httptest.NewRequest(http.MethodPost, "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	filename, data, ok := readUploadedFormFile(c, "image", maxVisitorImageUploadSizeBytes)

	assert.False(t, ok)
	assert.Empty(t, filename)
	assert.Nil(t, data)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to open uploaded file")
}

func TestReadUploadedFormFile_ReadFileError(t *testing.T) {
	origRead := readUploadedFile
	t.Cleanup(func() {
		readUploadedFile = origRead
	})
	readUploadedFile = func(_ io.Reader) ([]byte, error) {
		return nil, errors.New("read error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "ok.jpg")
	require.NoError(t, err)
	_, err = part.Write([]byte("abc"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c.Request = httptest.NewRequest(http.MethodPost, "/upload", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	filename, data, ok := readUploadedFormFile(c, "image", maxVisitorImageUploadSizeBytes)

	assert.False(t, ok)
	assert.Empty(t, filename)
	assert.Nil(t, data)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to read uploaded file")
}

func TestParseVisitorID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "15"}}

		id, ok := parseVisitorID(c)

		assert.True(t, ok)
		assert.Equal(t, int32(15), id)
	})

	t.Run("invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		id, ok := parseVisitorID(c)

		assert.False(t, ok)
		assert.Zero(t, id)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid visitor ID")
	})
}

func TestParseUserUUID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := uuid.New()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: expected.String()}}

		id, ok := parseUserUUID(c)

		assert.True(t, ok)
		assert.Equal(t, expected, id)
	})

	t.Run("invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "id", Value: "bad-id"}}

		id, ok := parseUserUUID(c)

		assert.False(t, ok)
		assert.Equal(t, uuid.Nil, id)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user ID")
	})
}

func TestCurrentAdminUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := &entity.User{ID: uuid.New(), UserName: "admin"}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("id", expected)

		user, ok := currentAdminUser(c)

		assert.True(t, ok)
		assert.Equal(t, expected, user)
	})

	t.Run("missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		user, ok := currentAdminUser(c)

		assert.False(t, ok)
		assert.Nil(t, user)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "user not found in auth context")
	})

	t.Run("invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("id", "not-a-user")

		user, ok := currentAdminUser(c)

		assert.False(t, ok)
		assert.Nil(t, user)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "invalid user data in auth context")
	})
}
