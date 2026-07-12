# go-browser - Documentation

> Back to [README](../README.md)

## Prerequisites

- Go 1.25 or higher
- Google Chrome or Chromium browser (macOS or Linux)
- macOS: Chrome installed at `/Applications/Google Chrome.app/`
- Linux: `google-chrome`, `google-chrome-stable`, `chromium`, or `chromium-browser` available in `PATH`
- `sqlite3` command-line tool (for cookie extraction)
- macOS: `security` command-line tool (built-in, for Chrome Safe Storage password)
- Linux: `secret-tool` (for Chrome Safe Storage password, requires `libsecret-tools` package)

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
| `DISPLAY` | No | On Linux, set this to use non-headless mode via X11 |
| `WAYLAND_DISPLAY` | No | On Linux, set this to use non-headless mode via Wayland |

### Chrome Profile

The library auto-detects the Chrome profile path:

| Platform | Path |
|----------|------|
| macOS | `~/Library/Application Support/Google/Chrome` |
| Linux | `~/.config/google-chrome` |

Defaults to the profile named `Default`. To use a different profile, specify it via `Option.Profile`.

## Usage

### Basic: Fetch Page as Markdown

```go
package main

import (
    "context"
    "fmt"
    "time"

    goBrowser "github.com/pardnchiu/go-browser"
)

func main() {
    ctx := context.Background()
    result, err := goBrowser.Fetch(ctx, "https://example.com", 30*time.Second, &goBrowser.Option{
        Type:        goBrowser.TypeMarkdown,
        Headless:    true,
        ScrollCount: 3,
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Title)
    fmt.Println(result.Content)
}
```

### Advanced: Access Login-Required Pages with Cookie Session

```go
result, err := goBrowser.Fetch(ctx, "https://login-required-site.com", 60*time.Second, &goBrowser.Option{
    Type:        goBrowser.TypeMarkdown,
    SameSession: true,
    Profile:     "Default",
    ScrollCount: 5,
    KeepLinks:   true,
})
if err != nil {
    panic(err)
}
```

### Interactive Tab Operations

```go
// Create a tab and navigate
tabID, err := goBrowser.CreateTab(ctx, "https://example.com", &goBrowser.Option{
    SameSession: true,
    Headless:    false,
})
if err != nil {
    panic(err)
}
defer goBrowser.CloseTab(tabID)

// Click an element
if err := goBrowser.TabClick(tabID, "#login-button"); err != nil {
    panic(err)
}

// Type text into a field
if err := goBrowser.TabType(tabID, "#username", "myuser"); err != nil {
    panic(err)
}

// Scroll the page
if err := goBrowser.TabScroll(tabID, 3); err != nil {
    panic(err)
}

// Take a snapshot
result, err := goBrowser.TabSnapshot(tabID)
if err != nil {
    panic(err)
}
fmt.Println(result.Content)

// Execute custom JavaScript
output, err := goBrowser.TabEval(tabID, "document.title")
if err != nil {
    panic(err)
}
fmt.Println(output)
```

### Output Formats

```go
// Markdown (default)
result, _ := goBrowser.Fetch(ctx, url, timeout, &goBrowser.Option{Type: goBrowser.TypeMarkdown})

// HTML
result, _ := goBrowser.Fetch(ctx, url, timeout, &goBrowser.Option{Type: goBrowser.TypeHTML})

// JSON tree
result, _ := goBrowser.Fetch(ctx, url, timeout, &goBrowser.Option{Type: goBrowser.TypeJSON})
```

## API Reference

### Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

Fetches the content of the specified URL. Automatically determines whether headless mode or cookie session is needed, and retries on 403/429 responses.

### CreateTab

```go
func CreateTab(ctx context.Context, href string, opt *Option) (string, error)
```

Creates an interactive tab and returns a tab ID for subsequent operations.

### TabClick

```go
func TabClick(tabID, selector string) error
```

Clicks the element matching the given CSS selector in the specified tab.

### TabType

```go
func TabType(tabID, selector, text string) error
```

Types the given text into the input element matching the CSS selector in the specified tab.

### TabScroll

```go
func TabScroll(tabID string, count int) error
```

Scrolls the page in the specified tab the given number of times.

### TabNavigate

```go
func TabNavigate(tabID, href string) error
```

Navigates an existing tab to a new URL.

### TabEval

```go
func TabEval(tabID, js string) (string, error)
```

Executes JavaScript in the specified tab and returns the result as a string.

### TabSnapshot

```go
func TabSnapshot(tabID string) (*Result, error)
```

Captures the current page content of the specified tab, outputting Markdown, HTML, or JSON depending on `Option.Type`.

### CloseTab

```go
func CloseTab(tabID string) error
```

Closes the specified tab and releases resources. When all tabs are closed, the interactive browser instance is automatically shut down.

### Close

```go
func Close()
```

Closes all tabs and browser instances, releasing all resources.

### SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

Sets the maximum number of concurrent fetches (default: 8).

### Option

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `IdleWait` | `time.Duration` | `2s` | Time to wait for DOM stability |
| `MaxLength` | `int` | `1MB` | Maximum output content length in bytes |
| `UserAgent` | `string` | Chrome 124 | Custom User-Agent string |
| `KeepLinks` | `bool` | `false` | Whether to preserve links and images |
| `StealthJS` | `string` | Built-in | Custom stealth JavaScript |
| `SettleJS` | `string` | Built-in | JavaScript to run after page load |
| `Viewport` | `*Viewport` | `1280x960` | Viewport size and device scale factor |
| `SameSession` | `bool` | `false` | Use Chrome profile cookie session |
| `Headless` | `bool` | `false` | Force headless mode |
| `Profile` | `string` | `"Default"` | Chrome profile name |
| `Type` | `int` | `TypeMarkdown` | Output format: `TypeMarkdown`, `TypeHTML`, `TypeJSON` |
| `ScrollCount` | `int` | `3` | Number of scroll simulations |

### Result

| Field | Type | Description |
|-------|------|-------------|
| `Href` | `string` | Original request URL |
| `FinalURL` | `string` | Final URL after redirects |
| `Content` | `string` | Extracted content (Markdown / HTML / JSON) |
| `ContentType` | `string` | Page Content-Type |
| `Title` | `string` | Page title |
| `Author` | `string` | Article author |
| `PublishedAt` | `string` | Publication time (RFC3339 format) |
| `Excerpt` | `string` | Article excerpt |
| `Status` | `int` | HTTP status code |
| `Tree` | `[]*Node` | Structured tree nodes in JSON mode |

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)