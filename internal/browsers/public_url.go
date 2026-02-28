package browsers

import (
	"net"
	"net/url"
	"path"
	"strings"
)

// PublicCDPURLFromBrowser maps a browser's internal CDP endpoint to a public gateway URL.
// Example:
//
//	raw:  http://127.0.0.1:50100
//	base: https://bbaas-manager.b8z.me/browsers
//	out:  https://bbaas-manager.b8z.me/browsers/50100
func PublicCDPURLFromBrowser(browser Browser, publicBaseURL string) string {
	return PublicCDPURLFromRaw(browser.CDPHTTPURL, browser.CDPURL, publicBaseURL)
}

func PublicCDPURLFromRaw(rawHTTPURL string, rawWSURL string, publicBaseURL string) string {
	normalizedBase := strings.TrimSpace(publicBaseURL)
	if normalizedBase == "" {
		return strings.TrimSpace(rawHTTPURL)
	}

	base, err := url.Parse(normalizedBase)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return strings.TrimSpace(rawHTTPURL)
	}

	port := extractPort(rawHTTPURL)
	if port == "" {
		port = extractPort(rawWSURL)
	}
	if port == "" {
		return strings.TrimSpace(rawHTTPURL)
	}

	base.Path = path.Join("/", strings.TrimSpace(base.Path), port)
	base.RawPath = ""
	return base.String()
}

func RewriteBrowserForPublicGateway(browser Browser, publicBaseURL string) Browser {
	publicURL := PublicCDPURLFromBrowser(browser, publicBaseURL)
	if publicURL == "" {
		return browser
	}

	browser.CDPHTTPURL = publicURL
	browser.CDPURL = publicURL
	return browser
}

func extractPort(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Port() != "" {
		return parsed.Port()
	}

	if !strings.Contains(trimmed, "://") {
		parsedWithScheme, parseErr := url.Parse("http://" + trimmed)
		if parseErr == nil && parsedWithScheme.Port() != "" {
			return parsedWithScheme.Port()
		}
	}

	if host, _, splitErr := net.SplitHostPort(trimmed); splitErr == nil && host != "" {
		_, port, _ := net.SplitHostPort(trimmed)
		return port
	}

	return ""
}
