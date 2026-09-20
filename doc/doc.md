# go-browser - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.25 or higher
- Google Chrome or Chromium (macOS or Linux)
- macOS: Chrome at `/Applications/Google Chrome.app/`
- Linux: `google-chrome`, `google-chrome-stable`, `chromium`, or `chromium-browser` on `PATH`
- `sqlite3` CLI (cookie extraction)
- macOS: built-in `security` tool (Chrome Safe Storage password)
- Linux: `secret-tool` (`libsecret-tools`) for Chrome Safe Storage password

## Installation

### From Source

```bash
git clone https://github.com/pardnchiu/go-browser.git
cd go-browser
go build ./...
```

### Using go get

```bash
go get github.com/pardnchiu/go-browser
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DISPLAY` | No | On Linux, enables headed mode via X11 |
| `WAYLAND_DISPLAY` | No | On Linux, enables headed mode via Wayland |

### Chrome Profile

The library auto-detects the Chrome profile path:

| Platform | Path |
|----------|------|
| macOS | `~/Library/Application Support/Google/Chrome` |
| Linux | `~/.config/google-chrome` |

Defaults to the profile named `Default`. Override with `Option.Profile`.

## Usage

### Basic: Fetch as Markdown

```go
package main

import (
	"context"
	"fmt"
	"time"

	browser "github.com/pardnchiu/go-browser"
)

func main() {
	ctx := context.Background()
	result, err := browser.Fetch(ctx, "https://example.com", 30*time.Second, &browser.Option{
		Type:        browser.TypeMarkdown,
		Headless:    true,
		ScrollCount: 3,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Title)
	fmt.Println(result.Content)
	// Consent may be none / skipped / strategy names; it only records an attempt
	fmt.Println(result.Consent)
}
```

### Advanced: Login-Required Pages via Cookie Session

```go
result, err := browser.Fetch(ctx, "https://login-required-site.com", 60*time.Second, &browser.Option{
	Type:        browser.TypeMarkdown,
	SameSession: true,
	Profile:     "Default",
	ScrollCount: 5,
	KeepLinks:   true,
})
if err != nil {
	panic(err)
}
```

`SameSession: true` copies and decrypts cookies from the local Chrome profile into a temporary browser before navigation. Full CDP interaction (click, type, multi-step flows) is out of scope — use Playwright MCP for that.

### Output Formats

```go
// Markdown (default)
result, err := browser.Fetch(ctx, url, timeout, &browser.Option{Type: browser.TypeMarkdown})
if err != nil {
	panic(err)
}

// HTML (merged scroll snapshots)
result, err = browser.Fetch(ctx, url, timeout, &browser.Option{Type: browser.TypeHTML})
if err != nil {
	panic(err)
}

// JSON structure tree
result, err = browser.Fetch(ctx, url, timeout, &browser.Option{Type: browser.TypeJSON})
if err != nil {
	panic(err)
}
```

### Concurrency and Shutdown

```go
browser.SetMaxConcurrency(4) // default 8
defer browser.Close()        // close all cached browser instances
```

## API Reference

### Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

Fetches the URL through Chrome. Tries headless first; on 403/429/503 with a display available, retries headed. Session-required hosts prefer the cookie path.

### SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

Sets the maximum concurrent fetches (default 8). Values `n <= 0` are ignored.

### Close

```go
func Close()
```

Closes all cached browser instances and releases resources.

### Merge / Dedup / HTML conversion

```go
func Merge(snapshots []string) (string, error)
func DedupTree(nodes []*Node)
func DedupMarkdownParagraphs(md string) string
func InlineTimeElements(htmlSrc string) (string, error)
func HTMLToNode(content, baseURL string, keepLinks bool) ([]*Node, error)
func HTMLToMarkdown(content, baseURL string, keepLinks bool) (string, error)
```

Multi-snapshot merge, paragraph/node dedup, and HTML → Markdown/tree conversion. Used inside `Fetch`; also callable directly.

### Option

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `IdleWait` | `time.Duration` | `2s` | Wait for DOM stability |
| `MaxLength` | `int` | `1MB` | Max Markdown output length in bytes |
| `UserAgent` | `string` | Chrome 124 | Custom User-Agent; also part of the browser cache key |
| `KeepLinks` | `bool` | `false` | Keep links and images |
| `StealthJS` | `string` | Built-in | Custom stealth script |
| `SettleJS` | `string` | Built-in | JS run after page load |
| `Viewport` | `*Viewport` | `1280x960` | Viewport size and device scale |
| `SameSession` | `bool` | `false` | Use local Chrome profile cookies |
| `Headless` | `bool` | `false` | Force headless (no headed fallback when true) |
| `Profile` | `string` | `"Default"` | Chrome profile name |
| `Type` | `int` | `TypeMarkdown` | `TypeMarkdown` / `TypeHTML` / `TypeJSON` |
| `ScrollCount` | `int` | `3` | Scroll simulation count; negative becomes 0 |

### Result

| Field | Type | Description |
|-------|------|-------------|
| `Href` | `string` | Original request URL |
| `FinalURL` | `string` | Final URL after redirects |
| `Content` | `string` | Extracted content (Markdown / HTML / JSON string) |
| `ContentType` | `string` | Page Content-Type (when JSON/XML is returned raw) |
| `Title` | `string` | Page title |
| `Author` | `string` | Article author |
| `PublishedAt` | `string` | Publication time (RFC3339) |
| `Excerpt` | `string` | Article excerpt |
| `Status` | `int` | HTTP status code |
| `Consent` | `string` | Cookie consent banner attempt result (optional) |
| `Tree` | `[]*Node` | Structure-tree nodes in JSON mode |

### Constants

| Name | Description |
|------|-------------|
| `TypeMarkdown` | Markdown output |
| `TypeHTML` | Merged HTML output |
| `TypeJSON` | Structure-tree JSON output |
| `DefaultUserAgent` | Default Chrome 124 User-Agent |
| `ErrProfileNotFound` | Named Chrome profile was not found |

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
