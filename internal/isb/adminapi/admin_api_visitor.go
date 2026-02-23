package adminapi

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CreateVisitorRequest struct {
	Name      string   `json:"name" binding:"required"`
	Surname   string   `json:"surname"`
	Email     string   `json:"email"`
	IsStudent bool     `json:"is_student"`
	Grade     *int     `json:"grade"`
	Keys      []string `json:"keys"`
}

type UpdateVisitorRequest struct {
	Name      string   `json:"name" binding:"required"`
	Surname   string   `json:"surname"`
	Email     string   `json:"email"`
	IsStudent bool     `json:"is_student"`
	Grade     *int     `json:"grade"`
	Keys      []string `json:"keys"`
}

type AddKeyRequest struct {
	Key string `json:"key" binding:"required"`
}

func parseVisitorID(c *gin.Context) (int32, bool) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid visitor ID"})
		return 0, false
	}
	return int32(id), true
}

func (ac *AdminAPIController) ListVisitorsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		visitors, err := ac.AdminAPI.GetAllVisitors()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, visitors)
	}
}

func (ac *AdminAPIController) GetVisitorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseVisitorID(c)
		if !ok {
			return
		}

		visitor, err := ac.AdminAPI.GetVisitor(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, visitor)
	}
}

func (ac *AdminAPIController) CreateVisitorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateVisitorRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		visitor, err := ac.AdminAPI.CreateVisitor(req.Name, req.Surname, req.IsStudent, req.Grade, req.Email, req.Keys)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, visitor)
	}
}

func (ac *AdminAPIController) UpdateVisitorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseVisitorID(c)
		if !ok {
			return
		}

		var req UpdateVisitorRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		visitor, err := ac.AdminAPI.UpdateVisitor(id, req.Name, req.Surname, req.IsStudent, req.Grade, req.Email, req.Keys)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, visitor)
	}
}

func (ac *AdminAPIController) UploadVisitorImageHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseVisitorID(c)
		if !ok {
			return
		}

		fileHeader, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
			return
		}
		defer func() { _ = file.Close() }()

		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
			return
		}

		visitor, err := ac.AdminAPI.UploadVisitorImage(id, fileHeader.Filename, data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, visitor)
	}
}

func (ac *AdminAPIController) AddVisitorKeyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseVisitorID(c)
		if !ok {
			return
		}

		var req AddKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := ac.AdminAPI.AddVisitorKey(id, req.Key); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "key added successfully"})
	}
}

func (ac *AdminAPIController) RemoveVisitorKeyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseVisitorID(c)
		if !ok {
			return
		}

		key := c.Param("key")
		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}

		if err := ac.AdminAPI.RemoveVisitorKey(id, key); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "key removed successfully"})
	}
}

func (ac *AdminAPIController) DeleteVisitorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseVisitorID(c)
		if !ok {
			return
		}

		if err := ac.AdminAPI.DeleteVisitor(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "visitor deleted successfully"})
	}
}
