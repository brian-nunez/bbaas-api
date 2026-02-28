package v1

import (
	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/browsers"
	"github.com/brian-nunez/bbaas-api/internal/dashboard"
	uihandlers "github.com/brian-nunez/bbaas-api/internal/handlers/v1/ui"
	"github.com/brian-nunez/bbaas-api/internal/users"
	"github.com/labstack/echo/v4"
)

type Dependencies struct {
	UsersService        *users.Service
	ApplicationsService *applications.Service
	BrowserService      *browsers.Service
	DashboardService    *dashboard.Service
}

func RegisterRoutes(e *echo.Echo, dependencies Dependencies) {
	uiHandler := uihandlers.NewHandler(
		dependencies.UsersService,
		dependencies.ApplicationsService,
		dependencies.DashboardService,
	)

	browsersHandler := NewBrowsersHandler(dependencies.BrowserService)
	apiKeyMiddleware := APIKeyAuthMiddleware(dependencies.ApplicationsService)

	uiGroup := e.Group("", uihandlers.SessionMiddleware(dependencies.UsersService))
	uiGroup.GET("/", uiHandler.Home)
	uiGroup.GET("/register", uiHandler.ShowRegister, uihandlers.RequireGuest)
	uiGroup.POST("/register", uiHandler.Register, uihandlers.RequireGuest)
	uiGroup.GET("/login", uiHandler.ShowLogin, uihandlers.RequireGuest)
	uiGroup.POST("/login", uiHandler.Login, uihandlers.RequireGuest)
	uiGroup.POST("/logout", uiHandler.Logout, uihandlers.RequireAuth)

	uiGroup.GET("/dashboard", uiHandler.Dashboard, uihandlers.RequireAuth)
	uiGroup.POST("/dashboard/applications", uiHandler.CreateApplication, uihandlers.RequireAuth)
	uiGroup.POST("/dashboard/applications/:applicationId/api-keys", uiHandler.CreateAPIKey, uihandlers.RequireAuth)
	uiGroup.POST("/dashboard/applications/:applicationId/api-keys/:keyId/revoke", uiHandler.RevokeAPIKey, uihandlers.RequireAuth)

	v1Group := e.Group("/api/v1")
	v1Group.GET("/health", HealthHandler)

	browsersGroup := v1Group.Group("/browsers", apiKeyMiddleware)
	browsersGroup.POST("", browsersHandler.SpawnBrowser)
	browsersGroup.GET("", browsersHandler.ListBrowsers)
	browsersGroup.GET("/:id", browsersHandler.GetBrowser)
	browsersGroup.POST("/:id/keepalive", browsersHandler.KeepAliveBrowser)
	browsersGroup.DELETE("/:id", browsersHandler.CloseBrowser)
}
