package v1

import (
	"errors"
	"strings"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	handlererrors "github.com/brian-nunez/bbaas-api/internal/handlers/errors"
	"github.com/labstack/echo/v4"
)

func APIKeyAuthMiddleware(applicationService *applications.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiToken := extractAPIToken(c)
			if apiToken == "" {
				response := handlererrors.Unauthorized().
					WithMessage("Missing API token. Use Authorization: Bearer <token> or X-API-Key header").
					Build()
				return c.JSON(response.HTTPStatusCode, response)
			}

			application, err := applicationService.AuthenticateToken(c.Request().Context(), apiToken)
			if err != nil {
				if errors.Is(err, applications.ErrInvalidAPIToken) {
					response := handlererrors.Unauthorized().
						WithMessage("Invalid API token").
						Build()
					return c.JSON(response.HTTPStatusCode, response)
				}

				return err
			}

			setAuthenticatedApplication(c, application)
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
