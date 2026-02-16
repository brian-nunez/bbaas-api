package v1

import (
	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/browsers"
	uihandlers "github.com/brian-nunez/bbaas-api/internal/handlers/v1/ui"
	"github.com/labstack/echo/v4"
)

type Dependencies struct {
	ApplicationService *applications.Service
	BrowserService     *browsers.Service
}

func RegisterRoutes(e *echo.Echo, dependencies Dependencies) {
	applicationsHandler := NewApplicationsHandler(dependencies.ApplicationService)
	browsersHandler := NewBrowsersHandler(dependencies.BrowserService)
	apiTokenMiddleware := APIKeyAuthMiddleware(dependencies.ApplicationService)

	e.GET("/", uihandlers.HomeHandler)

	v1Group := e.Group("/api/v1")
	v1Group.GET("/health", HealthHandler)
	v1Group.POST("/applications", applicationsHandler.RegisterApplication)

	browsersGroup := v1Group.Group("/browsers", apiTokenMiddleware)
	browsersGroup.POST("", browsersHandler.SpawnBrowser)
	browsersGroup.GET("", browsersHandler.ListBrowsers)
	browsersGroup.GET("/:id", browsersHandler.GetBrowser)
	browsersGroup.POST("/:id/keepalive", browsersHandler.KeepAliveBrowser)
	browsersGroup.DELETE("/:id", browsersHandler.CloseBrowser)
}
