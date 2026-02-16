package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/brian-nunez/bbaas-api/internal/browsers"
	handlererrors "github.com/brian-nunez/bbaas-api/internal/handlers/errors"
	"github.com/labstack/echo/v4"
)

type BrowsersHandler struct {
	browserService *browsers.Service
}

func NewBrowsersHandler(browserService *browsers.Service) *BrowsersHandler {
	return &BrowsersHandler{
		browserService: browserService,
	}
}

func (h *BrowsersHandler) SpawnBrowser(c echo.Context) error {
	application, ok := getAuthenticatedApplication(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing authenticated application")
	}

	request, err := decodeSpawnRequest(c)
	if err != nil {
		response := handlererrors.InvalidRequest().WithMessage("Invalid JSON body").Build()
		return c.JSON(response.HTTPStatusCode, response)
	}

	spawnedBrowser, err := h.browserService.SpawnForApplication(c.Request().Context(), application.ID, request)
	if err != nil {
		return mapBrowserServiceError(err)
	}

	return c.JSON(http.StatusCreated, spawnedBrowser)
}

func (h *BrowsersHandler) ListBrowsers(c echo.Context) error {
	application, ok := getAuthenticatedApplication(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing authenticated application")
	}

	availableBrowsers, err := h.browserService.ListForApplication(c.Request().Context(), application.ID)
	if err != nil {
		return mapBrowserServiceError(err)
	}

	return c.JSON(http.StatusOK, map[string][]browsers.Browser{
		"browsers": availableBrowsers,
	})
}

func (h *BrowsersHandler) GetBrowser(c echo.Context) error {
	application, ok := getAuthenticatedApplication(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing authenticated application")
	}

	browser, err := h.browserService.GetForApplication(c.Request().Context(), application.ID, strings.TrimSpace(c.Param("id")))
	if err != nil {
		return mapBrowserServiceError(err)
	}

	return c.JSON(http.StatusOK, map[string]browsers.Browser{
		"browser": browser,
	})
}

func (h *BrowsersHandler) KeepAliveBrowser(c echo.Context) error {
	application, ok := getAuthenticatedApplication(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing authenticated application")
	}

	browser, err := h.browserService.KeepAliveForApplication(c.Request().Context(), application.ID, strings.TrimSpace(c.Param("id")))
	if err != nil {
		return mapBrowserServiceError(err)
	}

	return c.JSON(http.StatusOK, map[string]browsers.Browser{
		"browser": browser,
	})
}

func (h *BrowsersHandler) CloseBrowser(c echo.Context) error {
	application, ok := getAuthenticatedApplication(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing authenticated application")
	}

	err := h.browserService.CloseForApplication(c.Request().Context(), application.ID, strings.TrimSpace(c.Param("id")))
	if err != nil {
		return mapBrowserServiceError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func decodeSpawnRequest(c echo.Context) (browsers.SpawnRequest, error) {
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 1<<20))
	if err != nil {
		return browsers.SpawnRequest{}, fmt.Errorf("read request body: %w", err)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return browsers.SpawnRequest{}, nil
	}

	var request browsers.SpawnRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return browsers.SpawnRequest{}, fmt.Errorf("decode request body: %w", err)
	}

	return request, nil
}

func mapBrowserServiceError(err error) error {
	if errors.Is(err, browsers.ErrBrowserNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	var upstreamError *browsers.UpstreamError
	if errors.As(err, &upstreamError) {
		return echo.NewHTTPError(upstreamError.StatusCode, upstreamError.Message)
	}

	return err
}
