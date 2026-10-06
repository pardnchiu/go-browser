# go-browser - Documentation

Last updated: 2026-10-06

> Back to [README](../README.md)

## Prerequisites

- Go 1.25 or higher
- Google Chrome or Chromium (macOS or Linux)
- macOS: Chrome at `/Applications/Google Chrome.app/` or `/Applications/Chromium.app/`
- Linux: `google-chrome`, `google-chrome-stable`, `chromium`, or `chromium-browser` on `PATH`
- For `SameSession`:
  - `sqlite3` CLI (reads the Cookies database)
  - macOS: built-in `security` tool (reads the Chrome Safe Storage password)
  - Linux: `secret-tool` (`libsecret-tools`, reads the Chrome Safe Storage password)

## Installation

### Using go get

```bash
go get github.com/pardnchiu/go-browser/core
```

### From Source

```bash
git clone https://github.com/pardnchiu/go-browser.git
cd go-browser
go build ./...
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DISPLAY` | No | On Linux, marks a display as available and enables headed retry (X11) |
| `WAYLAND_DISPLAY` | No | On Linux, marks a display as available and enables headed retry (Wayland) |

macOS always counts as having a display. On Linux with neither variable set, the library skips headed retry and returns the headless result.

### Chrome Profile

`SameSession` reads the Chrome profile from:

| Platform | Path |
|----------|------|
| macOS | `~/Library/Application Support/Google/Chrome/<Profile>` |
| Linux | `~/.config/google-chrome/<Profile>` |

`<Profile>` defaults to `Default`; override it with `Option.Profile`. When the profile is missing, the library falls back to a regular cached browser instead of returning an error.

## Usage

### Basic: Fetch as Markdown

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	browser "github.com/pardnchiu/go-browser/core"
)

func main() {
	defer browser.Close()

	result, err := browser.Fetch(context.Background(), "https://example.com", 30*time.Second, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Title)
	fmt.Println(result.Content)
}
```

A `nil` `opt` applies every default: Markdown output, headless first, three scrolls.

### Handling HTTP Errors

```go
result, err := browser.Fetch(ctx, href, 30*time.Second, nil)
if err != nil {
	var httpErr *browser.Error
	if errors.As(err, &httpErr) {
		// 403: blocked or Cloudflare challenge; 204: no content extracted
		log.Printf("status %d: %s", httpErr.Status, httpErr.Href)
		return
	}
	log.Fatal(err)
}
fmt.Println(result.Status, result.FinalURL)
```

### Output Formats

```go
// Markdown (default), truncated by MaxLength
md, err := browser.Fetch(ctx, href, timeout, &browser.Option{Type: browser.TypeMarkdown})
if err != nil {
	log.Fatal(err)
}

// HTML: all scroll snapshots merged, <time> inlined
html, err := browser.Fetch(ctx, href, timeout, &browser.Option{Type: browser.TypeHTML})
if err != nil {
	log.Fatal(err)
}

// JSON: Content holds the serialized Result, including the Tree
tree, err := browser.Fetch(ctx, href, timeout, &browser.Option{Type: browser.TypeJSON})
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(md.Content), len(html.Content), len(tree.Content))
```

When the response Content-Type contains `json` or `xml`, the library returns the body as-is regardless of `Type` and fills `ContentType`.

### Advanced: Read Login-Required Pages with a Cookie Session

```go
result, err := browser.Fetch(ctx, "https://example.com/dashboard", 60*time.Second, &browser.Option{
	SameSession: true,
	Profile:     "Default",
	ScrollCount: 5,
	KeepLinks:   true,
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(result.Content)
```

`SameSession: true` copies the profile's Cookies file to a temp directory, decrypts it, and injects the cookies into a single-use browser. Multi-step interaction such as clicking or form filling is out of scope; use Playwright MCP instead.

### Advanced: Inspect Consent Banner Handling

```go
result, err := browser.Fetch(ctx, href, 30*time.Second, nil)
if err != nil {
	log.Fatal(err)
}
switch result.Consent {
case "", "none", "skipped":
	fmt.Println("no banner dismissed")
default:
	fmt.Println("applied strategy:", result.Consent)
}
```

### Concurrency and Shutdown

```go
browser.SetMaxConcurrency(4)
defer browser.Close()
```

## API Reference

### Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

Fetches the URL through Chrome. With `timeout <= 0`, only `ctx` bounds the call. `href` must have a scheme and a hostname containing `.`, otherwise it returns `invalid url`.

Routing order:

| Condition | Behavior |
|-----------|----------|
| `Option.Headless == true` | Headless only, no retry |
| Domain in the built-in social list (`facebook.com`, `x.com`, `linkedin.com`, ...) and a display exists | Headed directly |
| Otherwise | Headless first; retry headed on 403, 429, or 503 when a display exists |

### SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

Sets the maximum number of concurrent page loads (default 8). Ignores `n <= 0`.

### Close

```go
func Close()
```

Closes every cached browser instance. Cached instances idle for more than 5 minutes also close automatically.

### HTML Helpers

| Function | Signature | Description |
|----------|-----------|-------------|
| `Merge` | `func Merge(snapshots []string) (string, error)` | Merges multiple HTML snapshots |
| `InlineTimeElements` | `func InlineTimeElements(htmlSrc string) (string, error)` | Inlines `<time>` datetime values as text |
| `HTMLToMarkdown` | `func HTMLToMarkdown(content, baseURL string, keepLinks bool) (string, error)` | Converts HTML to Markdown |
| `HTMLToNode` | `func HTMLToNode(content, baseURL string, keepLinks bool) ([]*Node, error)` | Converts HTML to a node tree |
| `DedupMarkdownParagraphs` | `func DedupMarkdownParagraphs(md string) string` | Removes duplicate Markdown paragraphs |
| `DedupTree` | `func DedupTree(nodes []*Node) []*Node` | Removes duplicate nodes |

`Fetch` chains these internally; each one also works standalone.

### Option

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Type` | `int` | `TypeMarkdown` | `TypeMarkdown` / `TypeHTML` / `TypeJSON` |
| `Headless` | `bool` | `false` | `true` forces headless with no headed retry; `false` means headless first |
| `SameSession` | `bool` | `false` | Injects cookies from the local Chrome profile |
| `Profile` | `string` | `"Default"` | Chrome profile name used by `SameSession` |
| `ScrollCount` | `int` | `3` | Scroll steps; negative means 0; stops early when the page is not scrollable or the snapshot stops changing |
| `IdleWait` | `time.Duration` | `2s` | Upper bound for each DOM-settle wait |
| `MaxLength` | `int` | `1 << 20` | Markdown output limit in bytes |
| `KeepLinks` | `bool` | `false` | Keeps links and images |
| `UserAgent` | `string` | `DefaultUserAgent` | Custom User-Agent; forms the browser cache key together with headless |
| `StealthJS` | `string` | built-in | Script evaluated before every new document |
| `SettleJS` | `string` | built-in | Script evaluated after the DOM settles |
| `Viewport` | `*Viewport` | `1280x960` | Window size |

### Viewport

| Field | Type | Description |
|-------|------|-------------|
| `Width` | `int` | Width |
| `Height` | `int` | Height |
| `DeviceScaleFactor` | `float64` | Scale factor; `0` means `1` |

### Result

| Field | Type | Description |
|-------|------|-------------|
| `Href` | `string` | Original request URL |
| `FinalURL` | `string` | Final URL after redirects |
| `Content` | `string` | Markdown / HTML / JSON string, or raw JSON / XML |
| `ContentType` | `string` | Set only when JSON / XML passes through |
| `Title` | `string` | Article title (Markdown mode) |
| `Author` | `string` | Article byline (Markdown mode) |
| `PublishedAt` | `string` | Publish time in RFC3339 (Markdown mode) |
| `Excerpt` | `string` | Article excerpt (Markdown mode) |
| `Status` | `int` | HTTP status of the navigation response |
| `Consent` | `string` | Applied dismissal strategies (`selector` / `text` / `removed`), joined by `,` across passes; `none`, `skipped` (paywall or login wall detected), or empty means nothing was dismissed |
| `Tree` | `[]*Node` | Node tree (present only inside `TypeJSON` serialized content) |

### Node

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | Node type |
| `Text` | `string` | Text content |
| `Level` | `int` | Heading level |
| `Datetime` | `string` | `<time>` datetime |
| `Href` | `string` | Link (with `keepLinks`) |
| `Src` | `string` | Image source (with `keepLinks`) |
| `Alt` | `string` | Image alt text (with `keepLinks`) |
| `Children` | `[]*Node` | Child nodes |

### Error

```go
type Error struct {
	Status int
	Href   string
}
```

| Status | Trigger |
|--------|---------|
| `403` / `404` | Final URL path or query contains a `403` / `404` segment |
| `403` | Page title matches Cloudflare / access-denied phrases |
| `204` | Converted Markdown is empty |
| `>= 400` | Readability extracts nothing and the navigation status is 400 or above |

### Constants and Variables

| Name | Description |
|------|-------------|
| `TypeMarkdown` | Markdown output |
| `TypeHTML` | Merged HTML output |
| `TypeJSON` | Node-tree JSON output |
| `DefaultUserAgent` | Default Chrome 124 Linux User-Agent |
| `ErrProfileNotFound` | Chrome profile not found; `Fetch` falls back internally and never returns it |

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
