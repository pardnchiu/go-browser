package goBrowser

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func skipIfNoChrome(t *testing.T) {
	t.Helper()
	if chromePath() == "" {
		t.Skip("chrome/chromium not found")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	Close()
	os.Exit(code)
}

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"test","value":42,"items":["a","b","c"]}`))
	})

	mux.HandleFunc("/xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><root><item id="1">hello</item><item id="2">world</item></root>`))
	})

	mux.HandleFunc("/text-xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.Write([]byte(`<?xml version="1.0"?><feed><entry>test entry</entry></feed>`))
	})

	mux.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Test Page</title></head><body>
<article>
<h1>Test Article Title</h1>
<p>This is the first paragraph of the test article. It contains enough text for the readability algorithm to identify this as meaningful content worth extracting from the page.</p>
<p>This is the second paragraph providing additional content. The readability library needs multiple paragraphs with substantial text to properly identify the main content area of the page.</p>
<p>A third paragraph ensures we have sufficient density of text content within the article element for reliable extraction by the go-readability library.</p>
</article>
</body></html>`))
	})

	mux.HandleFunc("/cf-challenge", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Just a moment...</title></head><body>
<div id="challenge-running"><div class="ctp-checkbox-container"><div>Checking your browser before accessing the site.</div></div></div>
</body></html>`))
	})

	mux.HandleFunc("/rate-limit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Rate Limited</title></head><body><p>Too many requests</p></body></html>`))
	})

	mux.HandleFunc("/akamai-block", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Access Denied</title></head><body>
<h1>Access Denied</h1>
<p>You don't have permission to access this resource. Reference ID: abc123</p>
</body></html>`))
	})

	mux.HandleFunc("/empty-body", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Empty</title></head><body></body></html>`))
	})

	return httptest.NewServer(mux)
}

func fetchOpt() *Option {
	return &Option{
		ScrollCount: 1,
		IdleWait:    500 * time.Millisecond,
	}
}

func TestFetchContentType(t *testing.T) {
	skipIfNoChrome(t)
	srv := newTestServer()
	defer srv.Close()

	ctx := context.Background()
	timeout := 30 * time.Second

	t.Run("JSON", func(t *testing.T) {
		result, err := Fetch(ctx, srv.URL+"/json", timeout, fetchOpt())
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if !strings.Contains(result.ContentType, "json") {
			t.Errorf("ContentType = %q, want contains 'json'", result.ContentType)
		}
		var dic map[string]any
		if err := json.Unmarshal([]byte(result.Content), &dic); err != nil {
			t.Fatalf("Content is not valid JSON: %v\nContent: %s", err, result.Content)
		}
		if dic["name"] != "test" {
			t.Errorf("dic[name] = %v, want 'test'", dic["name"])
		}
		if dic["value"] != float64(42) {
			t.Errorf("dic[value] = %v, want 42", dic["value"])
		}
		if result.Title != "" {
			t.Errorf("Title = %q, want empty (no readability extraction)", result.Title)
		}
	})

	t.Run("XML", func(t *testing.T) {
		result, err := Fetch(ctx, srv.URL+"/xml", timeout, fetchOpt())
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if !strings.Contains(result.ContentType, "xml") {
			t.Errorf("ContentType = %q, want contains 'xml'", result.ContentType)
		}
		if !strings.Contains(result.Content, "<root>") {
			t.Errorf("Content missing <root>:\n%s", result.Content)
		}
		if !strings.Contains(result.Content, "hello") {
			t.Errorf("Content missing 'hello':\n%s", result.Content)
		}
		if result.Title != "" {
			t.Errorf("Title = %q, want empty (no readability extraction)", result.Title)
		}
	})

	t.Run("TextXML", func(t *testing.T) {
		result, err := Fetch(ctx, srv.URL+"/text-xml", timeout, fetchOpt())
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if !strings.Contains(result.ContentType, "xml") {
			t.Errorf("ContentType = %q, want contains 'xml'", result.ContentType)
		}
		if !strings.Contains(result.Content, "<feed>") {
			t.Errorf("Content missing <feed>:\n%s", result.Content)
		}
		if !strings.Contains(result.Content, "test entry") {
			t.Errorf("Content missing 'test entry':\n%s", result.Content)
		}
	})

	t.Run("HTML", func(t *testing.T) {
		result, err := Fetch(ctx, srv.URL+"/html", timeout, fetchOpt())
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if result.ContentType != "" {
			t.Errorf("ContentType = %q, want empty for HTML", result.ContentType)
		}
		if result.Content == "" {
			t.Fatal("Content is empty")
		}
		if strings.Contains(result.Content, "<html") || strings.Contains(result.Content, "<body") {
			t.Errorf("Content looks like raw HTML, expected markdown:\n%.200s", result.Content)
		}
	})
}

func TestFetchHeadlessOption(t *testing.T) {
	skipIfNoChrome(t)
	srv := newTestServer()
	defer srv.Close()

	ctx := context.Background()
	timeout := 30 * time.Second

	t.Run("HeadlessFalse", func(t *testing.T) {
		opt := fetchOpt()
		opt.Headless = false
		result, err := Fetch(ctx, srv.URL+"/html", timeout, opt)
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if result.Content == "" {
			t.Fatal("Content is empty")
		}
	})

	t.Run("HeadlessTrue", func(t *testing.T) {
		opt := fetchOpt()
		opt.Headless = true
		result, err := Fetch(ctx, srv.URL+"/html", timeout, opt)
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if result.Content == "" {
			t.Fatal("Content is empty")
		}
	})
}

func TestTabContentType(t *testing.T) {
	skipIfNoChrome(t)
	srv := newTestServer()
	defer srv.Close()

	ctx := context.Background()

	t.Run("JSON", func(t *testing.T) {
		tabID, err := CreateTab(ctx, srv.URL+"/json", fetchOpt())
		if err != nil {
			t.Fatalf("CreateTab: %v", err)
		}
		defer CloseTab(tabID)

		result, err := TabSnapshot(tabID)
		if err != nil {
			t.Fatalf("TabSnapshot: %v", err)
		}
		if !strings.Contains(result.ContentType, "json") {
			t.Errorf("ContentType = %q, want contains 'json'", result.ContentType)
		}
		var dic map[string]any
		if err := json.Unmarshal([]byte(result.Content), &dic); err != nil {
			t.Fatalf("Content is not valid JSON: %v\nContent: %s", err, result.Content)
		}
		if dic["name"] != "test" {
			t.Errorf("dic[name] = %v, want 'test'", dic["name"])
		}
	})

	t.Run("XML", func(t *testing.T) {
		tabID, err := CreateTab(ctx, srv.URL+"/xml", fetchOpt())
		if err != nil {
			t.Fatalf("CreateTab: %v", err)
		}
		defer CloseTab(tabID)

		result, err := TabSnapshot(tabID)
		if err != nil {
			t.Fatalf("TabSnapshot: %v", err)
		}
		if !strings.Contains(result.ContentType, "xml") {
			t.Errorf("ContentType = %q, want contains 'xml'", result.ContentType)
		}
		if !strings.Contains(result.Content, "<root>") {
			t.Errorf("Content missing <root>:\n%s", result.Content)
		}
		if !strings.Contains(result.Content, "hello") {
			t.Errorf("Content missing 'hello':\n%s", result.Content)
		}
	})

	t.Run("NavigateFromHTMLToJSON", func(t *testing.T) {
		tabID, err := CreateTab(ctx, srv.URL+"/html", fetchOpt())
		if err != nil {
			t.Fatalf("CreateTab: %v", err)
		}
		defer CloseTab(tabID)

		if err := TabNavigate(tabID, srv.URL+"/json"); err != nil {
			t.Fatalf("TabNavigate: %v", err)
		}

		result, err := TabSnapshot(tabID)
		if err != nil {
			t.Fatalf("TabSnapshot: %v", err)
		}
		if !strings.Contains(result.ContentType, "json") {
			t.Errorf("ContentType = %q after navigate to JSON, want contains 'json'", result.ContentType)
		}
		var dic map[string]any
		if err := json.Unmarshal([]byte(result.Content), &dic); err != nil {
			t.Fatalf("Content is not valid JSON after navigate: %v\nContent: %s", err, result.Content)
		}
	})
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"403", &Error{Status: 403, Href: "x"}, true},
		{"429", &Error{Status: 429, Href: "x"}, true},
		{"503", &Error{Status: 503, Href: "x"}, true},
		{"204", &Error{Status: 204, Href: "x"}, true},
		{"404", &Error{Status: 404, Href: "x"}, false},
		{"NoArticle", errors.New("readability: no article extracted from 3 snapshots"), true},
		{"OtherError", errors.New("page.Navigate: timeout"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRetry(tt.err); got != tt.want {
				t.Errorf("shouldRetry(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAntiBot(t *testing.T) {
	skipIfNoChrome(t)
	srv := newTestServer()
	defer srv.Close()

	ctx := context.Background()
	timeout := 30 * time.Second

	t.Run("CloudflareChallenge", func(t *testing.T) {
		_, err := Fetch(ctx, srv.URL+"/cf-challenge", timeout, fetchOpt())
		if err == nil {
			t.Fatal("expected error for Cloudflare challenge page")
		}
		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("expected *Error, got %T: %v", err, err)
		}
		if e.Status != 403 {
			t.Errorf("Status = %d, want 403", e.Status)
		}
	})

	t.Run("AkamaiBlock", func(t *testing.T) {
		_, err := Fetch(ctx, srv.URL+"/akamai-block", timeout, fetchOpt())
		if err == nil {
			t.Fatal("expected error for Akamai block page")
		}
		var e *Error
		if !errors.As(err, &e) {
			t.Fatalf("expected *Error, got %T: %v", err, err)
		}
		if e.Status != 403 {
			t.Errorf("Status = %d, want 403", e.Status)
		}
	})

	t.Run("RateLimit429", func(t *testing.T) {
		result, err := Fetch(ctx, srv.URL+"/rate-limit", timeout, fetchOpt())
		if err != nil {
			t.Logf("429 returned error (both attempts see same mock): %v", err)
			return
		}
		if result.Status != 429 {
			t.Errorf("Status = %d, want 429", result.Status)
		}
		t.Logf("status=%d content_len=%d", result.Status, len(result.Content))
	})

	t.Run("EmptyBody", func(t *testing.T) {
		result, err := Fetch(ctx, srv.URL+"/empty-body", timeout, fetchOpt())
		if err != nil {
			t.Logf("empty body returned error: %v", err)
			return
		}
		t.Logf("status=%d content_len=%d", result.Status, len(result.Content))
	})
}

func TestRealSites(t *testing.T) {
	skipIfNoChrome(t)
	if testing.Short() {
		t.Skip("skipping real site tests in short mode")
	}

	ctx := context.Background()
	timeout := 60 * time.Second
	opt := &Option{
		ScrollCount: 1,
		IdleWait:    2 * time.Second,
	}

	tests := []struct {
		name     string
		url      string
		wantLen  int
		mayBlock bool
	}{
		{"Wikipedia", "https://en.wikipedia.org/wiki/Go_(programming_language)", 500, false},
		{"GitHub", "https://github.com/go-rod/rod", 100, false},
		{"BBC", "https://www.bbc.com/news", 200, false},
		{"HackerNews", "https://news.ycombinator.com", 100, false},
		{"Cloudflare_Blog", "https://blog.cloudflare.com", 100, false},
		{"Medium", "https://medium.com/tag/golang", 100, true},
		{"Reddit", "https://www.reddit.com/r/golang/", 100, true},
		{"StackOverflow", "https://stackoverflow.com/questions/tagged/go", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Fetch(ctx, tt.url, timeout, opt)
			if err != nil {
				var e *Error
				if errors.As(err, &e) {
					t.Logf("BLOCKED status=%d href=%s", e.Status, e.Href)
				}
				if tt.mayBlock {
					t.Skipf("blocked (expected for strict anti-bot): %v", err)
				}
				t.Fatalf("Fetch(%s): %v", tt.url, err)
			}
			t.Logf("status=%d title=%q finalURL=%s content_len=%d",
				result.Status, result.Title, result.FinalURL, len(result.Content))
			if len(result.Content) < tt.wantLen {
				t.Errorf("content too short: got %d bytes, want >= %d\ncontent: %.300s",
					len(result.Content), tt.wantLen, result.Content)
			}
		})
	}
}
