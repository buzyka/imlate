package adminapi

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var openUploadedFile = func(fileHeader *multipart.FileHeader) (multipart.File, error) {
	return fileHeader.Open()
}

var readUploadedFile = io.ReadAll

func readUploadedFormFile(c *gin.Context, fieldName string, maxSize int64) (filename string, data []byte, ok bool) {
	fileHeader, err := c.FormFile(fieldName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s file is required", fieldName)})
		return "", nil, false
	}

	if fileHeader.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("image file is too large (max %s)", formatUploadSizeLimit(maxSize))})
		return "", nil, false
	}

	file, err := openUploadedFile(fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return "", nil, false
	}
	defer func() { _ = file.Close() }()

	data, err = readUploadedFile(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
		return "", nil, false
	}

	return fileHeader.Filename, data, true
}

// parseVisitorID reads the ":id" path segment as a visitor ID. Visitor IDs are
// int32, so the value is parsed at that width: anything outside the range is
// not a valid ID and is reported as a bad request.
func parseVisitorID(c *gin.Context) (int32, bool) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visitor ID"})
		return 0, false
	}
	return int32(id), true
}

func parseUserUUID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return uuid.Nil, false
	}
	return id, true
}

func currentAdminUser(c *gin.Context) (*entity.User, bool) {
	data, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in auth context"})
		return nil, false
	}

	user, ok := data.(*entity.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user data in auth context"})
		return nil, false
	}

	return user, true
}

func formatUploadSizeLimit(maxSize int64) string {
	const megabyte = int64(1024 * 1024)
	if maxSize > 0 && maxSize%megabyte == 0 {
		return fmt.Sprintf("%dMB", maxSize/megabyte)
	}
	return fmt.Sprintf("%d bytes", maxSize)
}
