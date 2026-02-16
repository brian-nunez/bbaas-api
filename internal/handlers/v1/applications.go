package v1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	handlererrors "github.com/brian-nunez/bbaas-api/internal/handlers/errors"
	"github.com/labstack/echo/v4"
)

type ApplicationsHandler struct {
	applicationService *applications.Service
}

func NewApplicationsHandler(applicationService *applications.Service) *ApplicationsHandler {
	return &ApplicationsHandler{
		applicationService: applicationService,
	}
}

type registerApplicationRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	GitHubProfileLink string `json:"githubProfileLink"`
}

func (h *ApplicationsHandler) RegisterApplication(c echo.Context) error {
	var request registerApplicationRequest
	if err := c.Bind(&request); err != nil {
		response := handlererrors.InvalidRequest().WithMessage("Invalid JSON body").Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	registeredApplication, err := h.applicationService.Register(c.Request().Context(), applications.RegisterInput{
		Name:              request.Name,
		Description:       request.Description,
		GitHubProfileLink: request.GitHubProfileLink,
	})
	if err != nil {
		if isApplicationValidationError(err) {
			response := handlererrors.InvalidRequest().WithMessage(err.Error()).Build()
			return c.JSON(response.HTTPStatusCode, response)
		}

		return fmt.Errorf("register application: %w", err)
	}

	return c.JSON(http.StatusCreated, registeredApplication)
}

func isApplicationValidationError(err error) bool {
	return errors.Is(err, applications.ErrNameRequired) ||
		errors.Is(err, applications.ErrNameTooLong) ||
		errors.Is(err, applications.ErrDescriptionTooLong) ||
		errors.Is(err, applications.ErrGitHubLinkRequired) ||
		errors.Is(err, applications.ErrInvalidGitHubLink)
}
