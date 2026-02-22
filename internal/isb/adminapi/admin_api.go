package adminapi

import (
	"net/http"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
)

type AdminAPIController struct {
	// identityKey used by the auth middleware to store the user in the context
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
