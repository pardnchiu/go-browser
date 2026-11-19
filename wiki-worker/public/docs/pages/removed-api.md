# Removed API

Every exported symbol that existed in an earlier release and no longer exists under the same import path, with the version that removed it and what to use instead.

## v0.3.2: interactive tab API and package rename

v0.3.2 narrowed the library to one-shot `Fetch` extraction. The Go package name also changed from `goBrowser` to `browser`; the import path `github.com/pardnchiu/go-browser/core` is unchanged, so only the identifier and any import alias need updating.

| Symbol | Signature in v0.3.1 | Status |
|---|---|---|
| `CreateTab` | `func CreateTab(ctx context.Context, href string, opt *Option) (string, error)` | Removed in v0.3.2; no public replacement, use `Fetch` for one-shot reads |
| `CloseTab` | `func CloseTab(tabID string) error` | Removed in v0.3.2; no public replacement |
| `TabSnapshot` | `func TabSnapshot(tabID string) (*Result, error)` | Removed in v0.3.2; `Fetch` returns the extracted content directly |
| `TabClick` | `func TabClick(tabID, selector string) error` | Removed in v0.3.2; no public replacement, use a full CDP client |
| `TabType` | `func TabType(tabID, selector, text string) error` | Removed in v0.3.2; no public replacement, use a full CDP client |
| `TabScroll` | `func TabScroll(tabID string, count int) error` | Removed in v0.3.2; `Option.ScrollCount` controls scrolling inside `Fetch` |
| `TabNavigate` | `func TabNavigate(tabID, href string) error` | Removed in v0.3.2; call `Fetch` with the new URL |
| `TabEval` | `func TabEval(tabID, js string) (string, error)` | Removed in v0.3.2; `Option.SettleJS` runs one script per call, results are not returned |

`Close` no longer tears down a shared interactive session, because that session was removed with the tab API.

## v0.2.0: package moved to core

v0.2.0 moved the whole package from the module root `github.com/pardnchiu/go-browser` into `github.com/pardnchiu/go-browser/core`. Every symbol below was removed from the root path and continues under `core` with the same name, unless noted.

| Symbol | Kind | Status |
|---|---|---|
| `Fetch`, `Close`, `SetMaxConcurrency` | func | Removed in v0.2.0 from the root path; same name under `core` |
| `HTMLToMarkdown`, `HTMLToNode`, `Merge`, `DedupTree`, `DedupMarkdownParagraphs`, `InlineTimeElements` | func | Removed in v0.2.0 from the root path; same name under `core` |
| `Option`, `Result`, `Error`, `Node`, `Viewport`, `Browser` | type | Removed in v0.2.0 from the root path; same name under `core` |
| `Error.Error` | method | Removed in v0.2.0 from the root path; same method under `core` |
| `DefaultUserAgent` | const | Removed in v0.2.0 from the root path; same name under `core` |
| `TypeMarkdown`, `TypeHTML`, `TypeJSON` | const | Removed in v0.2.0 from the root path; same name under `core` |
| `CreateTab`, `CloseTab`, `TabSnapshot`, `TabClick`, `TabType`, `TabScroll`, `TabNavigate`, `TabEval` | func | Removed in v0.2.0 from the root path; moved to `core`, then Removed in v0.3.2 |

## Migration

```go
// v0.1.x
import goBrowser "github.com/pardnchiu/go-browser"

// v0.2.0 – v0.3.1
import goBrowser "github.com/pardnchiu/go-browser/core"

// v0.3.2 and later
import browser "github.com/pardnchiu/go-browser/core"

result, err := browser.Fetch(ctx, href, 30*time.Second, &browser.Option{SameSession: true})
```

Release notes for each version are under [Released](/released/).
