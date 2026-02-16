package uihandlers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/brian-nunez/bbaas-api/internal/applications"
	"github.com/brian-nunez/bbaas-api/internal/dashboard"
	"github.com/brian-nunez/bbaas-api/internal/users"
	"github.com/brian-nunez/bbaas-api/views/pages"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	usersService        *users.Service
	applicationsService *applications.Service
	dashboardService    *dashboard.Service
}

func NewHandler(usersService *users.Service, applicationsService *applications.Service, dashboardService *dashboard.Service) *Handler {
	return &Handler{
		usersService:        usersService,
		applicationsService: applicationsService,
		dashboardService:    dashboardService,
	}
}

func (h *Handler) Home(c echo.Context) error {
	if _, ok := getCurrentUser(c); ok {
		return c.Redirect(http.StatusSeeOther, "/dashboard")
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *Handler) ShowRegister(c echo.Context) error {
	return renderAuth(c, "Register", "Create your account", "Use email + password to get started.", "/register", "Create account", "Already have an account?", "/login", "", "")
}

func (h *Handler) Register(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	_, err := h.usersService.Register(c.Request().Context(), email, password)
	if err != nil {
		return renderAuth(c, "Register", "Create your account", "Use email + password to get started.", "/register", "Create account", "Already have an account?", "/login", err.Error(), email)
	}

	_, sessionToken, err := h.usersService.Login(c.Request().Context(), email, password)
	if err != nil {
		return renderAuth(c, "Register", "Create your account", "Use email + password to get started.", "/register", "Create account", "Already have an account?", "/login", err.Error(), email)
	}

	setSessionCookie(c, sessionToken)
	return c.Redirect(http.StatusSeeOther, "/dashboard?success=Account+created")
}

func (h *Handler) ShowLogin(c echo.Context) error {
	return renderAuth(c, "Login", "Welcome back", "Login with your account credentials.", "/login", "Login", "Need an account?", "/register", "", "")
}

func (h *Handler) Login(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	_, sessionToken, err := h.usersService.Login(c.Request().Context(), email, password)
	if err != nil {
		return renderAuth(c, "Login", "Welcome back", "Login with your account credentials.", "/login", "Login", "Need an account?", "/register", err.Error(), email)
	}

	setSessionCookie(c, sessionToken)
	return c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (h *Handler) Logout(c echo.Context) error {
	sessionCookie, err := c.Cookie(sessionCookieName)
	if err == nil && sessionCookie != nil {
		_ = h.usersService.Logout(c.Request().Context(), sessionCookie.Value)
	}
	clearSessionCookie(c)

	return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *Handler) Dashboard(c echo.Context) error {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	viewData, err := h.dashboardService.BuildViewData(c.Request().Context(), currentUser)
	if err != nil {
		return err
	}

	successMessage := strings.TrimSpace(c.QueryParam("success"))
	errorMessage := strings.TrimSpace(c.QueryParam("error"))
	newAPIKey := strings.TrimSpace(c.QueryParam("new_key"))

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	return pages.Dashboard(viewData, successMessage, errorMessage, newAPIKey).Render(context.Background(), c.Response().Writer)
}

func (h *Handler) CreateApplication(c echo.Context) error {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	_, err := h.applicationsService.RegisterApplication(c.Request().Context(), currentUser, applications.RegisterApplicationInput{
		Name:        c.FormValue("name"),
		Description: c.FormValue("description"),
		GitHubLink:  c.FormValue("githubLink"),
		Domain:      c.FormValue("domain"),
	})
	if err != nil {
		return redirectToDashboard(c, "", err.Error(), "")
	}

	return redirectToDashboard(c, "Application created", "", "")
}

func (h *Handler) CreateAPIKey(c echo.Context) error {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	applicationID := c.Param("applicationId")
	createdKey, err := h.applicationsService.CreateAPIKey(c.Request().Context(), currentUser, applicationID, applications.CreateAPIKeyInput{
		Name: c.FormValue("name"),
		Permissions: applications.APIKeyPermissions{
			CanRead:   c.FormValue("canRead") != "",
			CanWrite:  c.FormValue("canWrite") != "",
			CanDelete: c.FormValue("canDelete") != "",
		},
	})
	if err != nil {
		return redirectToDashboard(c, "", err.Error(), "")
	}

	return redirectToDashboard(c, fmt.Sprintf("API key %q created", createdKey.APIKey.Name), "", createdKey.Token)
}

func (h *Handler) RevokeAPIKey(c echo.Context) error {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	applicationID := c.Param("applicationId")
	keyID := c.Param("keyId")
	err := h.applicationsService.RevokeAPIKey(c.Request().Context(), currentUser, applicationID, keyID)
	if err != nil {
		return redirectToDashboard(c, "", err.Error(), "")
	}

	return redirectToDashboard(c, "API key revoked", "", "")
}

func renderAuth(c echo.Context, pageTitle string, heading string, subtitle string, action string, submitLabel string, secondaryLabel string, secondaryURL string, errorMessage string, email string) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	return pages.AuthPage(pageTitle, heading, subtitle, action, submitLabel, secondaryLabel, secondaryURL, errorMessage, email).Render(context.Background(), c.Response().Writer)
}

func redirectToDashboard(c echo.Context, successMessage string, errorMessage string, newAPIKey string) error {
	query := make(url.Values)
	if strings.TrimSpace(successMessage) != "" {
		query.Set("success", successMessage)
	}
	if strings.TrimSpace(errorMessage) != "" {
		query.Set("error", errorMessage)
	}
	if strings.TrimSpace(newAPIKey) != "" {
		query.Set("new_key", newAPIKey)
	}

	path := "/dashboard"
	encodedQuery := query.Encode()
	if encodedQuery != "" {
		path += "?" + encodedQuery
	}

	return c.Redirect(http.StatusSeeOther, path)
}
