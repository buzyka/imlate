package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	gjwt "github.com/appleboy/gin-jwt/v3"
	_ "github.com/buzyka/imlate/docs"
	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/http/controller/adminapi"
	"github.com/buzyka/imlate/internal/infrastructure/cron"
	"github.com/buzyka/imlate/internal/infrastructure/gocontainer"
	httpauth "github.com/buzyka/imlate/internal/infrastructure/http/auth"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/buzyka/imlate/internal/isb/search"
	"github.com/buzyka/imlate/internal/isb/tracker"
	themeview "github.com/buzyka/imlate/internal/usecase/theme"
	"github.com/buzyka/imlate/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
	"github.com/subosito/gotenv"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           ImLate Admin API
// @version         2.0
// @description     This is the administration API for the ImLate tracking system.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @BasePath  /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description JWT token in format: "Bearer <JWT>".
// @description Token lookup order: Authorization header, query parameter `token`, cookie `jwt`.
// @description Protected admin routes require admin role.
func main() {
	envFiles := []string{".env"}
	rootPath := "."
	if rp, err := util.GetRootPath(); err == nil {
		rootPath = rp
		if util.FileExists(rootPath + "/.env") {
			envFiles = append(envFiles, rootPath+"/.env")
		}
	}
	_ = gotenv.Load(envFiles...)

	cfg, err := config.NewFromEnv()
	if err != nil {
		panic(fmt.Sprintf("Error loading config from env: %v\n", err))
	}

	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("Error validating config: %v", err))
	}

	gocontainer.Build(&cfg)

	// Start cron jobs
	stopCron, err := cron.RunCron(&cfg)
	if err != nil {
		panic(fmt.Sprintf("Error starting cron jobs: %v\n", err))
	}
	defer stopCron()

	// Start Gin server
	r := gin.Default()
	r.SetHTMLTemplate(template.Must(template.ParseFiles(filepath.Join(rootPath, "website", "reader.html"))))

	r.Static("/assets", filepath.Join(rootPath, "website", "assets"))
	r.Static("/storage", filepath.Join(rootPath, "storage"))

	// Define gita simple GET route
	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	themeService := &themeview.Service{}
	container.MustFill(container.Global, themeService)
	r.GET("/", func(ctx *gin.Context) {
		data, err := themeService.GetReaderPageData()
		if err != nil {
			ctx.HTML(http.StatusOK, "reader.html", themeview.DefaultReaderPageData())
			return
		}
		ctx.HTML(http.StatusOK, "reader.html", data)
	})

	// Add Routes for swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// administration panel routes
	adminHandler := func(ctx *gin.Context) {
		ctx.File(filepath.Join(rootPath, "website", "admin", "index.html"))
	}
	r.GET("/admin", adminHandler)
	r.GET("/admin/users", adminHandler)
	r.GET("/admin/admin-users", adminHandler)
	r.GET("/admin/login", adminHandler)
	r.GET("/admin/settings", adminHandler)
	r.GET("/admin/settings/", adminHandler)

	r.Static("/admin/assets", filepath.Join(rootPath, "website", "admin", "assets"))

	searchController := &search.SearchController{}
	container.MustFill(container.Global, searchController)
	r.GET("/search/:id", searchController.SearchHandler())

	var terminalAuth httpauth.TerminalAuthMiddleware
	container.MustFill(container.Global, &terminalAuth)

	trackerController := &tracker.TrackerController{}
	container.MustFill(container.Global, trackerController)
	r.POST("/change-time", trackerController.ChangeTimeHandler())
	r.POST("/track", terminalAuth.MiddlewareFunc(), trackerController.TrackHandler())
	r.POST("/find-and-track", terminalAuth.MiddlewareFunc(), trackerController.FindAndTrackHandler())

	adminController := &adminapi.AdminAPIController{}
	container.MustFill(container.Global, adminController)
	r.POST("/register-terminal", adminController.RegisterTerminalHandler())

	registerAdminRoutes(r)

	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}
	// Start the server on the configured port
	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}

func registerAdminRoutes(r *gin.Engine) {
	var authManager httpauth.AuthManager
	container.MustFill(container.Global, &authManager)
	authMiddleware, err := authManager.Init()
	if err != nil {
		panic(fmt.Sprintf("Error initializing auth middleware: %v\n", err))
	}

	// Public routes
	r.POST("/login", adminLoginHandler(authMiddleware))
	r.POST("/refresh", adminRefreshHandler(authMiddleware)) // RFC 6749 compliant refresh endpoint

	adminGroup := r.Group("/admin-api", authMiddleware.MiddlewareFunc())
	adminGroup.GET("/dashboard", adminDashboardHandler)
	adminGroup.GET("/version", adminVersionHandler)

	adminController := &adminapi.AdminAPIController{}
	container.MustFill(container.Global, adminController)
	adminGroup.GET("/theme", adminController.GetThemeHandler())
	adminGroup.POST("/theme/assets/:slot", adminController.UploadThemeAssetHandler())
	adminGroup.DELETE("/theme/assets/:slot", adminController.ResetThemeAssetHandler())
	adminGroup.PUT("/theme/settings", adminController.UpdateThemeSettingsHandler())
	adminGroup.GET("/current-user", adminController.CurrentUserHandler())
	adminGroup.GET("/users", adminController.ListUsersHandler())
	adminGroup.GET("/users/:id", adminController.GetUserHandler())
	adminGroup.POST("/users", adminController.CreateUserHandler())
	adminGroup.PUT("/users/:id", adminController.UpdateUserHandler())
	adminGroup.PUT("/users/:id/password", adminController.UpdatePasswordHandler())
	adminGroup.DELETE("/users/:id", adminController.DeleteUserHandler())

	adminGroup.GET("/visitors", adminController.ListVisitorsHandler())
	adminGroup.GET("/visitors/:id", adminController.GetVisitorHandler())
	adminGroup.POST("/visitors", adminController.CreateVisitorHandler())
	adminGroup.PUT("/visitors/:id", adminController.UpdateVisitorHandler())
	adminGroup.POST("/visitors/:id/image", adminController.UploadVisitorImageHandler())
	adminGroup.POST("/visitors/:id/key", adminController.AddVisitorKeyHandler())
	adminGroup.DELETE("/visitors/:id/key/:key", adminController.RemoveVisitorKeyHandler())
	adminGroup.DELETE("/visitors/:id", adminController.DeleteVisitorHandler())

	adminGroup.GET("/reports/visits", adminController.VisitsReportsHandler())
	adminGroup.POST("/track/visit", adminController.ManualTrackHandler())
}

// adminLoginHandler godoc
// @Summary      Login for admin API
// @Description  Authenticates a user with username/password and returns a JWT token.
// @Description  Credentials can be sent as JSON or form fields (`username`, `password`).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      adminapi.LoginRequest      true  "Login credentials"
// @Success      200      {object}  adminapi.AuthTokenResponse
// @Failure      400      {object}  adminapi.AuthErrorResponse
// @Failure      401      {object}  adminapi.AuthErrorResponse
// @Router       /login [post]
func adminLoginHandler(authMiddleware *gjwt.GinJWTMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		authMiddleware.LoginHandler(c)
	}
}

// adminRefreshHandler godoc
// @Summary      Refresh admin JWT token
// @Description  Refreshes a valid JWT token and returns a new token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      adminapi.RefreshRequest  true  "Refresh token payload"
// @Success      200  {object}  adminapi.AuthTokenResponse
// @Failure      400  {object}  adminapi.AuthErrorResponse
// @Failure      401  {object}  adminapi.AuthErrorResponse
// @Router       /refresh [post]
func adminRefreshHandler(authMiddleware *gjwt.GinJWTMiddleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		authMiddleware.RefreshHandler(c)
	}
}

// adminDashboardHandler godoc
// @Summary      Get admin dashboard greeting
// @Description  Returns a simple dashboard welcome message for authenticated admin users.
// @Tags         admin-dashboard
// @Produce      json
// @Success      200  {object}  adminapi.DashboardResponse
// @Failure      401  {object}  adminapi.AuthErrorResponse
// @Failure      403  {object}  adminapi.AuthErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/dashboard [get]
func adminDashboardHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to the admin dashboard!",
	})
}

// adminVersionHandler godoc
// @Summary      Get application version
// @Description  Returns the current application version. In development returns "2.0.x-dev", in production the actual release version.
// @Tags         admin-system
// @Produce      json
// @Success      200  {object}  adminapi.VersionResponse
// @Failure      401  {object}  adminapi.AuthErrorResponse
// @Failure      403  {object}  adminapi.AuthErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/version [get]
func adminVersionHandler(c *gin.Context) {
	resp := adminapi.VersionResponse{
		Version: version.Version,
	}
	c.JSON(http.StatusOK, resp)
}
