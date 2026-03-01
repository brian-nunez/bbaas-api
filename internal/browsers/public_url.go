package browsers

import (
	"net/url"
	"path"
	"strings"
)

// PublicCDPURLFromBrowser maps a browser CDP endpoint to a public base URL while preserving
// the original endpoint path and dynamic browser port.
// Example:
//
//	raw:  ws://127.0.0.1:50100/devtools/browser/abc
//	base: https://bbaas-manager.b8z.me
//	out:  wss://bbaas-manager.b8z.me/50100/devtools/browser/abc
func PublicCDPURLFromBrowser(browser Browser, publicBaseURL string) string {
	raw := strings.TrimSpace(browser.CDPURL)
	if raw == "" {
		raw = browser.CDPHTTPURL
	}
	return PublicCDPURLFromRaw(raw, publicBaseURL)
}

func PublicCDPURLFromRaw(rawURL string, publicBaseURL string) string {
	trimmedRaw := strings.TrimSpace(rawURL)
	normalizedBase := strings.TrimSpace(publicBaseURL)
	if normalizedBase == "" || trimmedRaw == "" {
		return trimmedRaw
	}

	base, err := url.Parse(normalizedBase)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return trimmedRaw
	}

	parsedRaw, err := url.Parse(trimmedRaw)
	if err != nil || parsedRaw.Host == "" {
		return trimmedRaw
	}

	port := parsedRaw.Port()
	if port == "" || base.Host == "" {
		return trimmedRaw
	}

	rewrittenPath := path.Join("/", strings.Trim(base.Path, "/"), port)
	rawPath := strings.Trim(parsedRaw.Path, "/")
	if rawPath != "" {
		rewrittenPath = path.Join(rewrittenPath, rawPath)
	}

	rewritten := url.URL{
		Scheme:   mapScheme(parsedRaw.Scheme, base.Scheme),
		Host:     base.Host,
		Path:     rewrittenPath,
		RawQuery: parsedRaw.RawQuery,
		Fragment: parsedRaw.Fragment,
	}

	return rewritten.String()
}

func RewriteBrowserForPublicGateway(browser Browser, publicBaseURL string) Browser {
	browser.CDPHTTPURL = PublicCDPURLFromRaw(browser.CDPHTTPURL, publicBaseURL)
	browser.CDPURL = PublicCDPURLFromRaw(browser.CDPURL, publicBaseURL)
	return browser
}

func mapScheme(rawScheme string, baseScheme string) string {
	switch strings.ToLower(rawScheme) {
	case "ws", "wss":
		switch strings.ToLower(baseScheme) {
		case "https", "wss":
			return "wss"
		default:
			return "ws"
		}
	case "http", "https":
		switch strings.ToLower(baseScheme) {
		case "https", "wss":
			return "https"
		default:
			return "http"
		}
	default:
		// Fall back to base scheme for unknown/rawless schemes.
		return strings.ToLower(baseScheme)
	}
}
