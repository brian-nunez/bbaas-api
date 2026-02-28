package v1

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	handlererrors "github.com/brian-nunez/bbaas-api/internal/handlers/errors"
	"github.com/labstack/echo/v4"
)

func APIKeyAuthMiddleware(applicationsService *applications.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawAPIKey := extractAPIToken(c)
			if rawAPIKey == "" {
				response := handlererrors.Unauthorized().
					WithMessage("Missing API key. Use Authorization: Bearer <key> or X-API-Key header").
					Build()
				return c.JSON(response.HTTPStatusCode, response)
			}

			authCtx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
			defer cancel()
			principal, err := applicationsService.AuthenticateAPIKey(authCtx, rawAPIKey)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					response := handlererrors.ServiceNotAvailable().
						WithMessage("API key authentication timed out").
						Build()
					return c.JSON(response.HTTPStatusCode, response)
				}
				if errors.Is(err, applications.ErrInvalidAPIKey) {
					response := handlererrors.Unauthorized().WithMessage("Invalid API key").Build()
					return c.JSON(response.HTTPStatusCode, response)
				}
				return err
			}

			setAPIKeyPrincipal(c, principal)
			return next(c)
		}
	}
}

func extractAPIToken(c echo.Context) string {
	authorizationHeader := strings.TrimSpace(c.Request().Header.Get(echo.HeaderAuthorization))
	if strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
		return strings.TrimSpace(authorizationHeader[7:])
	}

	return strings.TrimSpace(c.Request().Header.Get("X-API-Key"))
}
