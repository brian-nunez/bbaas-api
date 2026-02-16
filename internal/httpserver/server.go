package httpserver

import (
	"context"
	"fmt"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/browsers"
	v1 "github.com/brian-nunez/bbaas-api/internal/handlers/v1"
	"github.com/labstack/echo/v4"
)

type Server interface {
	Start(addr string) error
	Shutdown(ctx context.Context) error
}

type BootstrapConfig struct {
	StaticDirectories map[string]string
	CDPManagerBaseURL string
}

func Bootstrap(config BootstrapConfig) (Server, error) {
	browserManagerClient, err := browsers.NewHTTPManagerClient(config.CDPManagerBaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create CDP manager client: %w", err)
	}

	applicationRepository := applications.NewInMemoryRepository()
	applicationService := applications.NewService(applicationRepository)

	browserOwnershipStore := browsers.NewInMemoryOwnershipStore()
	browserService := browsers.NewService(browserManagerClient, browserOwnershipStore)

	server := New().
		WithStaticAssets(config.StaticDirectories).
		WithDefaultMiddleware().
		WithErrorHandler().
		WithRoutes(func(e *echo.Echo) {
			v1.RegisterRoutes(e, v1.Dependencies{
				ApplicationService: applicationService,
				BrowserService:     browserService,
			})
		}).
		WithNotFound().
		Build()

	return server, nil
}
