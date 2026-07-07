# go-browser

> [Info]
> Maintained as a standalone module since [`pardnchiu/go-pkg@v0.13.0`](https://github.com/pardnchiu/go-pkg).
>
> 中文文件：[doc/README.zh.md](doc/README.zh.md)

Chromium-based web fetching + interactive tab toolkit.

- **one-shot fetch**: `Fetch` → readability main content → markdown / HTML / JSON
- **interactive tabs**: `CreateTab` / `TabClick` / `TabType` / `TabScroll` / `TabNavigate` / `TabEval` / `TabSnapshot` / `CloseTab`
- **shared real Chrome login session** (macOS / Linux): cookie decryption + CDP injection, no need to change how Chrome is launched
- **headless-first + headed fallback**: tries headless fetch first by default, automatically retries with a headed browser + session on 403/429/503/anti-bot pages
- **anti-bot detection**: Cloudflare / Akamai challenge pages are detected and fail fast as 403
- **concurrency control**: `SetMaxConcurrency` caps concurrent fetches; idle browsers are auto-reclaimed

## Install

```bash
go get github.com/pardnchiu/go-browser
```

## Quick Start

```go
import goBrowser "github.com/pardnchiu/go-browser/core"

defer goBrowser.Close()

result, err := goBrowser.Fetch(ctx, "https://example.com/article", 30*time.Second, nil)
// result.Content / result.Title / result.Author / result.PublishedAt / result.FinalURL / result.Status
```

## Option

```go
type Option struct {
	IdleWait    time.Duration // DOM-stable wait time, default 2s
	MaxLength   int           // output content length cap (bytes), default 1MiB, 0 = unlimited
	UserAgent   string        // custom User-Agent
	KeepLinks   bool          // whether markdown output keeps links
	StealthJS   string        // custom anti-detection injection script, defaults to built-in stealth.js
	SettleJS    string        // custom settle-wait script, defaults to built-in listener.js
	Viewport    *Viewport     // window size, default 1280x960
	SameSession bool          // launch with a real Chrome profile (carries cookies/session)
	Headless    bool          // true = skip the headless attempt, always open a headed browser + session
	Profile     string        // Chrome profile name, default "Default"
	Type        int           // TypeMarkdown(0) / TypeHTML(1) / TypeJSON(2)
	ScrollCount int           // number of scrolls during fetch, default 3
}

type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}
```

## Fetch (one-shot)

```go
// default: headless fetch, auto-falls back to headed + session on anti-bot responses
result, err := goBrowser.Fetch(ctx, "https://example.com/article", 30*time.Second, nil)

// share a real Chrome login session (Darwin / Linux)
result, err = goBrowser.Fetch(ctx, "https://x.com/home", 30*time.Second, &goBrowser.Option{
	SameSession: true,
	Profile:     "Profile 1", // default "Default"
})

// always open a headed browser + session, never attempt headless
result, err = goBrowser.Fetch(ctx, "https://example.com/login-required", 30*time.Second, &goBrowser.Option{
	Headless: true,
})

// switch output format / customize scroll count
result, err = goBrowser.Fetch(ctx, "https://example.com", 30*time.Second, &goBrowser.Option{
	Type:        goBrowser.TypeJSON, // TypeMarkdown(0) / TypeHTML(1) / TypeJSON(2)
	ScrollCount: 5,
})

// custom viewport / user-agent / content length cap
result, err = goBrowser.Fetch(ctx, "https://example.com", 30*time.Second, &goBrowser.Option{
	Viewport:  &goBrowser.Viewport{Width: 1920, Height: 1080, DeviceScaleFactor: 2},
	UserAgent: "Mozilla/5.0 ...",
	MaxLength: 500 * 1024,
	KeepLinks: true,
})
```

### Result

```go
type Result struct {
	Href        string
	FinalURL    string
	Content     string
	ContentType string  // non-empty = raw JSON/XML response, not parsed by readability
	Title       string
	Author      string
	PublishedAt string  // RFC3339
	Excerpt     string
	Status      int
	Tree        []*Node // populated only when Type == TypeJSON
}
```

Responses with a `Content-Type` of `application/json` or `*/xml` are returned as-is (`ContentType` non-empty), skipping readability/markdown conversion.

### Error Handling

```go
var fe *goBrowser.Error
if errors.As(err, &fe) {
	_ = fe.Status // 404 / 403 / 429 / 503 / 204 ...
}
```

- `403`: HTTP 403, or an anti-bot challenge page (Cloudflare/Akamai, etc.) was detected
- `429` / `503`: triggers the headless → headed+session automatic retry (only when a display is available)
- `204`: the markdown extracted by readability was empty
- `404`: the URL path/query matched a 404 segment

## Interactive Tabs

Each function can be used independently as an LLM tool call.

```go
tabID, err := goBrowser.CreateTab(ctx, "https://example.com/login", &goBrowser.Option{
	SameSession: true,
	Profile:     "Profile 1",
})
defer goBrowser.CloseTab(tabID)

snap, err := goBrowser.TabSnapshot(tabID)          // current page content (same shape as Fetch's Result)
err = goBrowser.TabType(tabID, "#email", "x@y.com")
err = goBrowser.TabType(tabID, "#password", "...")
err = goBrowser.TabClick(tabID, "#submit")
err = goBrowser.TabScroll(tabID, 3)                // count = 0 uses Option.ScrollCount
snap, err = goBrowser.TabSnapshot(tabID)
err = goBrowser.TabNavigate(tabID, "https://example.com/dashboard")
title, err := goBrowser.TabEval(tabID, `document.title`)
```

All tabs share a single browser instance at any given time; closing the last tab automatically releases the browser and profile snapshot.

## Concurrency & Resource Management

```go
goBrowser.SetMaxConcurrency(4) // default 8, cap on concurrent fetches

defer goBrowser.Close()        // release the shared headless browser before the program exits
```

Shared browsers idle for more than 1 minute are automatically closed and reclaimed.

> [Info]
> This project was auto-generated by Claude Code after reading the source code.
