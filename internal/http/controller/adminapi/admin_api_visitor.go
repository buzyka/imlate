package adminapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxVisitorImageUploadSizeBytes int64 = 5 * 1024 * 1024

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

// ListVisitorsHandler godoc
// @Summary      List visitors
// @Description  Returns all visitors available in the administration area.
// @Tags         admin-visitors
// @Produce      json
// @Success      200  {array}   VisitorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors [get]
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

// GetVisitorHandler godoc
// @Summary      Get visitor by ID
// @Description  Returns one visitor by numeric identifier.
// @Tags         admin-visitors
// @Produce      json
// @Param        id   path      int     true  "Visitor ID"
// @Success      200  {object}  VisitorResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors/{id} [get]
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

// CreateVisitorHandler godoc
// @Summary      Create visitor
// @Description  Creates a visitor record with optional keys and student metadata.
// @Tags         admin-visitors
// @Accept       json
// @Produce      json
// @Param        request  body      CreateVisitorRequest  true  "Create visitor request"
// @Success      201      {object}  VisitorResponse
// @Failure      400      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors [post]
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

// UpdateVisitorHandler godoc
// @Summary      Update visitor
// @Description  Updates visitor profile data, including keys and student fields.
// @Tags         admin-visitors
// @Accept       json
// @Produce      json
// @Param        id       path      int                   true  "Visitor ID"
// @Param        request  body      UpdateVisitorRequest  true  "Update visitor request"
// @Success      200      {object}  VisitorResponse
// @Failure      400      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors/{id} [put]
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

// UploadVisitorImageHandler godoc
// @Summary      Upload visitor image
// @Description  Uploads an image file for the specified visitor.
// @Tags         admin-visitors
// @Accept       mpfd
// @Produce      json
// @Param        id     path      int                  true   "Visitor ID"
// @Param        image  formData  file                 true   "Visitor image file"
// @Success      200    {object}  VisitorResponse
// @Failure      400    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors/{id}/image [post]
func (ac *AdminAPIController) UploadVisitorImageHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := parseVisitorID(c)
		if !ok {
			return
		}

		filename, data, ok := readUploadedFormFile(c, "image", maxVisitorImageUploadSizeBytes)
		if !ok {
			return
		}

		visitor, err := ac.AdminAPI.UploadVisitorImage(id, filename, data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, visitor)
	}
}

// AddVisitorKeyHandler godoc
// @Summary      Add visitor key
// @Description  Adds one access key to the specified visitor.
// @Tags         admin-visitors
// @Accept       json
// @Produce      json
// @Param        id       path      int            true  "Visitor ID"
// @Param        request  body      AddKeyRequest  true  "Add key request"
// @Success      200      {object}  MessageResponse
// @Failure      400      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors/{id}/key [post]
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

// RemoveVisitorKeyHandler godoc
// @Summary      Remove visitor key
// @Description  Removes one access key from the specified visitor.
// @Tags         admin-visitors
// @Produce      json
// @Param        id   path      int     true  "Visitor ID"
// @Param        key  path      string  true  "Visitor key"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors/{id}/key/{key} [delete]
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

// DeleteVisitorHandler godoc
// @Summary      Delete visitor
// @Description  Deletes the visitor identified by numeric ID.
// @Tags         admin-visitors
// @Produce      json
// @Param        id   path      int     true  "Visitor ID"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/visitors/{id} [delete]
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
