package httpauth

import (
	"fmt"
	"net/http"
	"time"

	gjwt "github.com/appleboy/gin-jwt/v3"
	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type login struct {
  Username string `form:"username" json:"username" binding:"required"`
  Password string `form:"password" json:"password" binding:"required"`
}

var (
  identityKey = "id"
)

type AuthManager struct{
  Config *config.Config `container:"type"`
  UserRepo provider.UserRepository `container:"type"`
}

func (a *AuthManager) Init() (*gjwt.GinJWTMiddleware, error) {
  // the jwt middleware
  authMiddleware, err := gjwt.New(a.initParams())
  if err != nil {
    return nil, fmt.Errorf("JWT Error: %w", err)
  }

  errInit := authMiddleware.MiddlewareInit()
  if errInit != nil {
    return nil, fmt.Errorf("authMiddleware.MiddlewareInit() Error: %w", errInit)
  }

  return authMiddleware, nil
}

func (a *AuthManager) initParams() *gjwt.GinJWTMiddleware {
  return &gjwt.GinJWTMiddleware{
    Realm:       "admin zone",
    Key:         []byte(a.Config.AuthTokenSecret),
    Timeout:     time.Minute * 3,
    MaxRefresh:  time.Minute * 3,
    IdentityKey: identityKey,
    PayloadFunc: a.payloadFunc(),

    IdentityHandler: a.identityHandler(),
    Authenticator:   a.authenticator(),
    Authorizer:      a.authorizer(),
    Unauthorized:    a.unauthorized(),
    LogoutResponse:  a.logoutResponse(),
    TokenLookup:     "header: Authorization, query: token, cookie: jwt",
    // TokenLookup: "query:token",
    // TokenLookup: "cookie:token",
    TokenHeadName: "Bearer",
    TimeFunc:      time.Now,
  }
}

func (a *AuthManager) payloadFunc() func(data any) jwt.MapClaims {
  return func(data any) jwt.MapClaims {
    if v, ok := data.(*entity.User); ok {
      return jwt.MapClaims{
        identityKey: v.ID.String(),
      }
    }
    return jwt.MapClaims{}
  }
}

func (a *AuthManager) identityHandler() func(c *gin.Context) any {
  return func(c *gin.Context) any {
    claims := gjwt.ExtractClaims(c)
    user, err := a.UserRepo.FindByID(uuid.MustParse(claims[identityKey].(string)))
    if err != nil {
      return err
    }
    return user
  }
}

func (a *AuthManager) authenticator() func(c *gin.Context) (any, error) {
  return func(c *gin.Context) (any, error) {
    var loginVals login
    if err := c.ShouldBind(&loginVals); err != nil {
      return "", gjwt.ErrMissingLoginValues
    }

    user, err := a.UserRepo.FindByUsername(loginVals.Username)
    if err != nil {
      return nil, gjwt.ErrFailedAuthentication
    }
    if user == nil || !user.PasswordValidate(loginVals.Password) {
      return nil, gjwt.ErrFailedAuthentication
    }

    return user, nil
  }
}

func (a *AuthManager) authorizer() func(c *gin.Context, data any) bool {
  return func(c *gin.Context, data any) bool {    
    if v, ok := data.(*entity.User); ok && v.Role == entity.UserRoleAdmin {
      return true
    }
    return false
  }
}

func (a *AuthManager) unauthorized() func(c *gin.Context, code int, message string) {
  return func(c *gin.Context, code int, message string) {
    c.JSON(code, gin.H{
      "code":    code,
      "message": message,
    })
  }
}

func (a *AuthManager) logoutResponse() func(c *gin.Context) {
  return func(c *gin.Context) {
    // This demonstrates that claims are now accessible during logout
    claims := gjwt.ExtractClaims(c)
    user, exists := c.Get(identityKey)

    response := gin.H{
      "code":    http.StatusOK,
      "message": "Successfully logged out",
    }

    // Show that we can access user information during logout
    if len(claims) > 0 {
      response["logged_out_user"] = claims[identityKey]
    }
    if exists {
      response["user_info"] = user.(*entity.User).UserName
    }

    c.JSON(http.StatusOK, response)
  }
}
