package goBrowser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	goRod "github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

type Browser struct {
	browser  *goRod.Browser
	lastUsed time.Time
}

const (
	browserIdleTTL   = 5 * time.Minute
	browserIdleCheck = 1 * time.Minute
)

var (
	mu        sync.Mutex
	browser   *Browser
	fetchSem  = make(chan struct{}, 8)
	evictOnce sync.Once
)

func startEvictor() {
	evictOnce.Do(func() {
		go func() {
			t := time.NewTicker(browserIdleCheck)
			defer t.Stop()
			for range t.C {
				mu.Lock()
				if browser != nil && time.Since(browser.lastUsed) > browserIdleTTL {
					_ = browser.browser.Close()
					browser = nil
				}
				mu.Unlock()
			}
		}()
	})
}

func SetMaxConcurrency(n int) {
	if n <= 0 {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	fetchSem = make(chan struct{}, n)
}

func acquireSem(ctx context.Context) (func(), error) {
	mu.Lock()
	sem := fetchSem
	mu.Unlock()

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func ensureBrowser(userAgent string, headless bool) (*goRod.Browser, error) {
	mu.Lock()
	defer mu.Unlock()
	startEvictor()
	if browser != nil {
		browser.lastUsed = time.Now()
		return browser.browser, nil
	}

	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	l := launcher.New().
		Headless(headless).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox", "").
		Set("disable-dev-shm-usage", "").
		Set("window-size", "1280,960").
		Set("user-agent", userAgent)

	if !headless {
		l = l.Set("window-position", "-32000,-32000")
	}

	if bin := chromePath(); bin != "" {
		l = l.Bin(bin)
	}

	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launcher.Launch: %w", err)
	}

	b := goRod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("browser.Connect: %w", err)
	}
	browser = &Browser{browser: b, lastUsed: time.Now()}
	return b, nil
}

func launchWithSnapshot(ctx context.Context, profileName, userAgent string, headless bool) (*goRod.Browser, func(), error) {
	profileRoot := chromeProfileRoot()
	if profileRoot == "" {
		return nil, nil, fmt.Errorf("cannot resolve chrome profile path on %s", runtime.GOOS)
	}
	srcProfileDir := filepath.Join(profileRoot, profileName)
	if _, err := os.Stat(srcProfileDir); err != nil {
		return nil, nil, fmt.Errorf("chrome profile %q not found at %s: %w", profileName, srcProfileDir, err)
	}

	tmpDir, err := os.MkdirTemp("", "rod-snapshot-*")
	if err != nil {
		return nil, nil, fmt.Errorf("mktemp: %w", err)
	}
	removeTmp := func() { _ = os.RemoveAll(tmpDir) }

	cookiesSnap := filepath.Join(tmpDir, "_cookies")
	if err := os.MkdirAll(cookiesSnap, 0700); err != nil {
		removeTmp()
		return nil, nil, fmt.Errorf("mkdir cookies snap: %w", err)
	}
	for _, name := range []string{"Cookies", "Cookies-wal", "Cookies-shm"} {
		if err := copyFileIfExists(filepath.Join(srcProfileDir, name), filepath.Join(cookiesSnap, name)); err != nil {
			removeTmp()
			return nil, nil, fmt.Errorf("copy %s: %w", name, err)
		}
	}

	cookies, err := extractChromeCookies(ctx, filepath.Join(cookiesSnap, "Cookies"))
	if err != nil {
		removeTmp()
		return nil, nil, fmt.Errorf("extract cookies: %w", err)
	}

	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	l := launcher.New().
		Headless(headless).
		UserDataDir(tmpDir).
		Delete("use-mock-keychain").
		Delete("password-store").
		Set("password-store", "keychain").
		Set("disable-blink-features", "AutomationControlled").
		Set("no-first-run", "").
		Set("no-default-browser-check", "").
		Set("disable-sync", "").
		Set("disable-background-networking", "").
		Set("no-sandbox", "").
		Set("disable-dev-shm-usage", "").
		Set("window-size", "1280,960").
		Set("user-agent", userAgent)

	if !headless {
		l = l.Set("window-position", "-32000,-32000")
	}

	if bin := chromePath(); bin != "" {
		l = l.Bin(bin)
	}

	controlURL, err := l.Launch()
	if err != nil {
		removeTmp()
		return nil, nil, fmt.Errorf("launcher.Launch: %w", err)
	}

	b := goRod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		removeTmp()
		return nil, nil, fmt.Errorf("browser.Connect: %w", err)
	}

	if len(cookies) > 0 {
		sanitized := make([]*proto.NetworkCookieParam, 0, len(cookies))
		for _, c := range cookies {
			if c.Domain == "" || c.Name == "" {
				continue
			}
			if c.Path == "" {
				c.Path = "/"
			}
			sanitized = append(sanitized, c)
		}
		if err := b.SetCookies(sanitized); err != nil {
			success := 0
			var lastErr error
			for _, c := range sanitized {
				if e := b.SetCookies([]*proto.NetworkCookieParam{c}); e == nil {
					success++
				} else {
					lastErr = e
				}
			}
			if success == 0 {
				_ = b.Close()
				removeTmp()
				return nil, nil, fmt.Errorf("inject cookies (0/%d): %w", len(sanitized), lastErr)
			}
		}
	}

	cleanup := func() {
		_ = b.Close()
		removeTmp()
	}
	return b, cleanup, nil
}

func chromeProfileRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	case "linux":
		return filepath.Join(home, ".config", "google-chrome")
	}
	return ""
}

func copyFileIfExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file, got dir: %s", src)
	}
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = io.Copy(d, s)
	return err
}

func Close() {
	interactiveMu.Lock()
	for id, t := range tabs {
		_ = t.page.Close()
		t.release()
		delete(tabs, id)
	}
	if interactiveCleanup != nil {
		interactiveCleanup()
		interactiveCleanup = nil
	}
	interactiveBrowser = nil
	interactiveMu.Unlock()

	mu.Lock()
	defer mu.Unlock()
	if browser != nil {
		_ = browser.browser.Close()
		browser = nil
	}
}

func hasDisplay() bool {
	if runtime.GOOS == "darwin" {
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func chromePath() string {
	switch runtime.GOOS {
	case "darwin":
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "linux":
		for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
			if p, err := exec.LookPath(name); err == nil {
				return p
			}
		}
	}
	return ""
}
