# API Reference

The fetch and lifecycle functions of `package browser`, import path `github.com/pardnchiu/go-browser/core`, with an index of every exported symbol.

## Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

Loads `href` in Chrome and returns its content in the format chosen by `opt.Type`.

| Parameter | Notes |
|---|---|
| `ctx` | Cancels every wait in the call, including the semaphore and scroll delays |
| `href` | Must have a scheme, a hostname containing `.`, no credentials and at most 2048 characters, otherwise `[Invalid] <url>: <reason>` |
| `timeout` | Above 0, wraps `ctx` for each attempt's page load (not `SameSession` cookie extraction or browser launch); 0 or below means no extra deadline |
| `opt` | May be `nil`; zero fields take defaults, see [Option Fields](/configuration) |

Picks headless or headed from `opt` and the runtime environment, and retries once headed only when the response is 403, 429 or 503 and the host has a GUI ([Fetch Routing](/core-concepts-routing)). Errors are either `*Error` with an HTTP-like status or a wrapped plain error ([Troubleshooting](/getting-started-troubleshooting)). Safe to call from multiple goroutines; concurrency is capped by `SetMaxConcurrency`.

## Close

```go
func Close()
```

Closes every cached browser instance and clears the cache. Safe to call more than once; later `Fetch` calls launch new instances.

## SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

Sets the cap on concurrent pages across all browsers, default 8. Does nothing when `n <= 0`. See [Browser Cache](/core-concepts-browser-cache).

## ValidURL

```go
func ValidURL(href string) (*url.URL, error)
```

Parses `href` with the checks `Fetch` applies, except the scheme: at most 2048 characters, no credentials, a hostname containing `.`. Errors read `[Invalid] <url>: <reason>`; credentials are masked and an overlong URL is cut to its first 64 characters.

## IsErrorPageTitle

```go
func IsErrorPageTitle(title string) bool
```

Reports whether `title` contains a not-found or access-denied phrase such as `404`, `page not found` or `找不到頁面`, case-insensitive. The phrases live in `core/embed/error_page_titles.json` and cover 16 languages.

## Symbol index

| Symbol | Kind | Page |
|---|---|---|
| `Fetch`, `Close`, `SetMaxConcurrency`, `ValidURL`, `IsErrorPageTitle` | func | this page |
| `HTMLToMarkdown`, `HTMLToNode`, `Merge`, `DedupTree`, `DedupMarkdownParagraphs`, `InlineTimeElements` | func | [Content Conversion](/api-reference-conversion) |
| `Option`, `Viewport`, `Result`, `Error`, `Node`, `Browser` | type | [Types](/api-reference-types) |
| `TypeMarkdown`, `TypeHTML`, `TypeJSON`, `DefaultUserAgent`, `ErrProfileNotFound`, `ErrChallengeRequired`, `ErrChallenge` | const / var | [Types](/api-reference-types) |

Symbols that existed in earlier releases are listed on [Removed API](/removed-api).
