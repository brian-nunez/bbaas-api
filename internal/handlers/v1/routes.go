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
	e.Use(uihandlers.SessionMiddleware(dependencies.UsersService))

	uiHandler := uihandlers.NewHandler(
		dependencies.UsersService,
		dependencies.ApplicationsService,
		dependencies.DashboardService,
	)

	browsersHandler := NewBrowsersHandler(dependencies.BrowserService)
	apiKeyMiddleware := APIKeyAuthMiddleware(dependencies.ApplicationsService)

	e.GET("/", uiHandler.Home)
	e.GET("/register", uiHandler.ShowRegister, uihandlers.RequireGuest)
	e.POST("/register", uiHandler.Register, uihandlers.RequireGuest)
	e.GET("/login", uiHandler.ShowLogin, uihandlers.RequireGuest)
	e.POST("/login", uiHandler.Login, uihandlers.RequireGuest)
	e.POST("/logout", uiHandler.Logout, uihandlers.RequireAuth)

	e.GET("/dashboard", uiHandler.Dashboard, uihandlers.RequireAuth)
	e.POST("/dashboard/applications", uiHandler.CreateApplication, uihandlers.RequireAuth)
	e.POST("/dashboard/applications/:applicationId/api-keys", uiHandler.CreateAPIKey, uihandlers.RequireAuth)
	e.POST("/dashboard/applications/:applicationId/api-keys/:keyId/revoke", uiHandler.RevokeAPIKey, uihandlers.RequireAuth)

	v1Group := e.Group("/api/v1")
	v1Group.GET("/health", HealthHandler)

	browsersGroup := v1Group.Group("/browsers", apiKeyMiddleware)
	browsersGroup.POST("", browsersHandler.SpawnBrowser)
	browsersGroup.GET("", browsersHandler.ListBrowsers)
	browsersGroup.GET("/:id", browsersHandler.GetBrowser)
	browsersGroup.POST("/:id/keepalive", browsersHandler.KeepAliveBrowser)
	browsersGroup.DELETE("/:id", browsersHandler.CloseBrowser)
}
