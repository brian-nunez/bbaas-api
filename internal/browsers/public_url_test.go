package browsers

import "testing"

func TestPublicCDPURLFromRaw(t *testing.T) {
	got := PublicCDPURLFromRaw(
		"http://127.0.0.1:50100",
		"ws://127.0.0.1:50100/devtools/browser/abc",
		"https://bbaas-manager.b8z.me/browsers",
	)

	want := "https://bbaas-manager.b8z.me/browsers/50100"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestPublicCDPURLFromRaw_UsesWSPortFallback(t *testing.T) {
	got := PublicCDPURLFromRaw(
		"",
		"ws://127.0.0.1:40555/devtools/browser/abc",
		"https://bbaas-manager.b8z.me/browsers",
	)

	want := "https://bbaas-manager.b8z.me/browsers/40555"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestRewriteBrowserForPublicGateway(t *testing.T) {
	browser := Browser{
		CDPHTTPURL: "http://127.0.0.1:50100",
		CDPURL:     "ws://127.0.0.1:50100/devtools/browser/abc",
	}

	rewritten := RewriteBrowserForPublicGateway(browser, "https://bbaas-manager.b8z.me/browsers")
	want := "https://bbaas-manager.b8z.me/browsers/50100"

	if rewritten.CDPHTTPURL != want {
		t.Fatalf("expected cdpHttpUrl %s, got %s", want, rewritten.CDPHTTPURL)
	}
	if rewritten.CDPURL != want {
		t.Fatalf("expected cdpUrl %s, got %s", want, rewritten.CDPURL)
	}
}
