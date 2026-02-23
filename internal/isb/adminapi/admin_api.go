package adminapi

import (
	"net/http"

	"github.com/buzyka/imlate/internal/domain/entity"
	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminAPIController struct {
	AdminAPI *usecase.AdminAPI `container:"type"`
}

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
