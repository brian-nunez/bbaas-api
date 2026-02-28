package httpserver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/authorization"
	"github.com/brian-nunez/bbaas-api/internal/browsers"
	"github.com/brian-nunez/bbaas-api/internal/dashboard"
	"github.com/brian-nunez/bbaas-api/internal/data"
	v1 "github.com/brian-nunez/bbaas-api/internal/handlers/v1"
	"github.com/brian-nunez/bbaas-api/internal/users"
	"github.com/labstack/echo/v4"
)

type Server interface {
	Start(addr string) error
	Shutdown(ctx context.Context) error
}

type BootstrapConfig struct {
	StaticDirectories map[string]string
	CDPManagerBaseURL string
	CDPPublicBaseURL  string
	DBDriver          string
	DBDSN             string
}

type appServer struct {
	echo *echo.Echo
	db   *sql.DB
}

func (s *appServer) Start(addr string) error {
	return s.echo.Start(addr)
}

func (s *appServer) Shutdown(ctx context.Context) error {
	echoShutdownErr := s.echo.Shutdown(ctx)
	dbCloseErr := s.db.Close()
	if echoShutdownErr != nil {
		return echoShutdownErr
	}
	if dbCloseErr != nil {
		return dbCloseErr
	}

	return nil
}

func Bootstrap(config BootstrapConfig) (Server, error) {
	db, _, err := data.Open(data.Config{
		Driver:       config.DBDriver,
		DSN:          config.DBDSN,
		MaxOpenConns: 10,
		MaxIdleConns: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := data.RunMigrations(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("run database migrations: %w", err)
	}

	store := data.NewStore(db)
	usersService := users.NewService(store)
	webAuthorizer := authorization.NewWebAuthorizer()
	applicationsService := applications.NewService(store, webAuthorizer)
	browserManagerClient, err := browsers.NewHTTPManagerClient(config.CDPManagerBaseURL, nil)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create CDP manager client: %w", err)
	}
	dashboardService := dashboard.NewService(store, usersService, applicationsService, browserManagerClient, config.CDPPublicBaseURL)

	apiAuthorizer := authorization.NewAPIAuthorizer()
	browserService := browsers.NewService(browserManagerClient, store, apiAuthorizer, config.CDPPublicBaseURL)

	echoServer := New().
		WithStaticAssets(config.StaticDirectories).
		WithDefaultMiddleware().
		WithErrorHandler().
		WithRoutes(func(e *echo.Echo) {
			v1.RegisterRoutes(e, v1.Dependencies{
				UsersService:        usersService,
				ApplicationsService: applicationsService,
				BrowserService:      browserService,
				DashboardService:    dashboardService,
			})
		}).
		WithNotFound().
		Build()

	return &appServer{
		echo: echoServer,
		db:   db,
	}, nil
}
