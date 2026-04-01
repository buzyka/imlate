package adminapi

import (
	"net/http"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	themeview "github.com/buzyka/imlate/internal/usecase/theme"
	"github.com/buzyka/imlate/internal/usecase/tracking"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminAPIController struct {
	AdminAPI       *usecase.AdminAPI               `container:"type"`
	TrackRepo      provider.VisitorTrackRepository `container:"type"`
	StudentTracker *tracking.StudentTracker        `container:"type"`
	ThemeService   *themeview.Service              `container:"type"`
}

// CurrentUserHandler godoc
// @Summary      Get current authenticated admin user
// @Description  Returns the user resolved from the JWT identity (`id`) in the request context.
// @Tags         admin-users
// @Produce      json
// @Success      200  {object}  entity.User
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/current-user [get]
func (ac *AdminAPIController) CurrentUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, exists := c.Get("id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user not found in auth context",
			})
			return
		}

		user, ok := data.(*entity.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "invalid user data in auth context",
			})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

// ListUsersHandler godoc
// @Summary      List admin users
// @Description  Returns all users available in the administration area.
// @Tags         admin-users
// @Produce      json
// @Success      200  {array}   entity.User
// @Failure      500  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/users [get]
func (ac *AdminAPIController) ListUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := ac.AdminAPI.GetAllUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

// GetUserHandler godoc
// @Summary      Get user by ID
// @Description  Returns one user by UUID.
// @Tags         admin-users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  entity.User
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/users/{id} [get]
func (ac *AdminAPIController) GetUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
			return
		}

		user, err := ac.AdminAPI.GetUser(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

type CreateUserRequest struct {
	Username string          `json:"username" binding:"required"`
	Password string          `json:"password" binding:"required"`
	Name     string          `json:"name" binding:"required"`
	Surname  string          `json:"surname" binding:"required"`
	Role     entity.UserRole `json:"role" binding:"required"`
}

// CreateUserHandler godoc
// @Summary      Create admin user
// @Description  Creates a new user account for the admin panel.
// @Tags         admin-users
// @Accept       json
// @Produce      json
// @Param        request  body      CreateUserRequest  true  "Create user request"
// @Success      201      {object}  entity.User
// @Failure      400      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/users [post]
func (ac *AdminAPIController) CreateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := ac.AdminAPI.CreateUser(req.Username, req.Password, req.Name, req.Surname, req.Role)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, user)
	}
}

type UpdateUserRequest struct {
	Name     string          `json:"name" binding:"required"`
	Surname  string          `json:"surname" binding:"required"`
	Role     entity.UserRole `json:"role" binding:"required"`
	IsActive bool            `json:"is_active"`
}

// UpdateUserHandler godoc
// @Summary      Update admin user
// @Description  Updates profile and role fields for the user identified by UUID.
// @Tags         admin-users
// @Accept       json
// @Produce      json
// @Param        id       path      string             true  "User UUID"
// @Param        request  body      UpdateUserRequest  true  "Update user request"
// @Success      200      {object}  entity.User
// @Failure      400      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/users/{id} [put]
func (ac *AdminAPIController) UpdateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
			return
		}

		var req UpdateUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := ac.AdminAPI.UpdateUser(id, req.Name, req.Surname, req.Role, req.IsActive)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

type UpdatePasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// UpdatePasswordHandler godoc
// @Summary      Update user password
// @Description  Replaces the password of the user identified by UUID.
// @Tags         admin-users
// @Accept       json
// @Produce      json
// @Param        id       path      string                 true  "User UUID"
// @Param        request  body      UpdatePasswordRequest  true  "Update password request"
// @Success      200      {object}  MessageResponse
// @Failure      400      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/users/{id}/password [put]
func (ac *AdminAPIController) UpdatePasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
			return
		}

		var req UpdatePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := ac.AdminAPI.UpdatePassword(id, req.Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "password updated successfully"})
	}
}

// DeleteUserHandler godoc
// @Summary      Delete admin user
// @Description  Deletes the user identified by UUID.
// @Tags         admin-users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/users/{id} [delete]
func (ac *AdminAPIController) DeleteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
			return
		}

		if err := ac.AdminAPI.DeleteUser(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
	}
}
