package httpauth

import (
	"net/http"
	"strings"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TerminalUserContextKey = "terminal_user"

type TerminalAuthMiddleware struct {
	UserRepo provider.UserRepository `container:"type"`
}

func (m *TerminalAuthMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		terminalName := c.GetHeader("Terminal-Name")
		if terminalName == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "missing Terminal-Name header",
			})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "missing Authorization header",
			})
			return
		}

		token, ok := parseBearerToken(authHeader)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "invalid Authorization header format, expected 'Bearer <token>'",
			})
			return
		}

		user, err := m.UserRepo.FindByUsername(terminalName)
		if err != nil || user == nil || user.ID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "terminal not found",
			})
			return
		}

		if !user.IsActive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "terminal is not active",
			})
			return
		}

		if user.Role != entity.UserRoleTerminal {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "user is not a terminal",
			})
			return
		}

		if !user.PasswordValidate(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "invalid terminal credentials",
			})
			return
		}

		c.Set(TerminalUserContextKey, user)
		c.Next()
	}
}

func parseBearerToken(header string) (string, bool) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
