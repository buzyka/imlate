package adminapi

import (
	"errors"
	"net/http"

	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	"github.com/gin-gonic/gin"
)

// RegisterTerminalRequest defines the payload for terminal registration.
type RegisterTerminalRequest struct {
	AdminLogin    string `json:"admin_login" binding:"required"`
	AdminPassword string `json:"admin_password" binding:"required"`
	TerminalName  string `json:"terminal_name" binding:"required"`
	ForceUpdate   bool   `json:"force_update"`
}

// RegisterTerminalResponse is the successful response for terminal registration.
// Note: auth_token contains the original generated password for the terminal,
// returned only at creation/regeneration time.
type RegisterTerminalResponse struct {
	AuthToken    string `json:"auth_token"`
	TerminalName string `json:"terminal_name"`
}

// RegisterTerminalHandler godoc
// @Summary      Register a terminal
// @Description  Validates admin credentials and creates (or force-updates) a terminal user.
// @Description  Returns the generated password as `auth_token`. This token is only returned
// @Description  at creation or regeneration time and should be stored securely by the client.
// @Tags         terminal
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterTerminalRequest  true  "Terminal registration request"
// @Success      200      {object}  RegisterTerminalResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse  "Terminal with this name already exists"
// @Failure      500      {object}  ErrorResponse
// @Router       /register-terminal [post]
func (ac *AdminAPIController) RegisterTerminalHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterTerminalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := ac.AdminAPI.RegisterTerminal(
			req.AdminLogin,
			req.AdminPassword,
			req.TerminalName,
			req.ForceUpdate,
		)
		if err != nil {
			if errors.Is(err, usecase.ErrTerminalAlreadyExists) {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			if errors.Is(err, usecase.ErrAuthenticationFailed) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, RegisterTerminalResponse{
			AuthToken:    result.AuthToken,
			TerminalName: result.TerminalName,
		})
	}
}
