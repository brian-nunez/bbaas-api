package browsers

import "testing"

func TestPublicCDPURLFromRaw(t *testing.T) {
	got := PublicCDPURLFromRaw(
		"ws://127.0.0.1:50100/devtools/browser/abc",
		"https://bbaas-manager.b8z.me",
	)

	want := "wss://bbaas-manager.b8z.me/50100/devtools/browser/abc"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestPublicCDPURLFromRaw_RewritesHTTPURL(t *testing.T) {
	got := PublicCDPURLFromRaw(
		"http://127.0.0.1:40555",
		"https://bbaas-manager.b8z.me",
	)

	want := "https://bbaas-manager.b8z.me/40555"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestPublicCDPURLFromRaw_RespectsBasePath(t *testing.T) {
	got := PublicCDPURLFromRaw(
		"ws://127.0.0.1:50100/devtools/browser/abc",
		"https://bbaas-manager.b8z.me/browsers",
	)

	want := "wss://bbaas-manager.b8z.me/browsers/50100/devtools/browser/abc"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestRewriteBrowserForPublicGateway(t *testing.T) {
	browser := Browser{
		CDPHTTPURL: "http://127.0.0.1:50100",
		CDPURL:     "ws://127.0.0.1:50100/devtools/browser/abc",
	}

	rewritten := RewriteBrowserForPublicGateway(browser, "https://bbaas-manager.b8z.me")
	wantHTTP := "https://bbaas-manager.b8z.me/50100"
	wantWS := "wss://bbaas-manager.b8z.me/50100/devtools/browser/abc"

	if rewritten.CDPHTTPURL != wantHTTP {
		t.Fatalf("expected cdpHttpUrl %s, got %s", wantHTTP, rewritten.CDPHTTPURL)
	}
	if rewritten.CDPURL != wantWS {
		t.Fatalf("expected cdpUrl %s, got %s", wantWS, rewritten.CDPURL)
	}
}
