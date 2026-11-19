# Types

The exported types, constants and variables of `package browser`, with field-level notes.

## Call types

### Option

The per-call settings. The full field table with defaults is on [Option Fields](/configuration).

### Viewport

```go
type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}
```

Page size in CSS pixels. `DeviceScaleFactor` 0 is treated as 1.

### Result

| Field | Type | Notes |
|---|---|---|
| `Href` | `string` | The requested URL |
| `FinalURL` | `string` | The URL after redirects |
| `Content` | `string` | Markdown, HTML, a JSON string or raw JSON/XML, depending on `Type` and content type |
| `ContentType` | `string` | Set only for JSON and XML documents; omitted from JSON when empty |
| `Title` | `string` | Readability title, `TypeMarkdown` only |
| `Author` | `string` | Readability byline, `TypeMarkdown` only |
| `PublishedAt` | `string` | RFC3339, when readability finds a date |
| `Excerpt` | `string` | Readability excerpt, `TypeMarkdown` only |
| `Status` | `int` | `responseStatus` from Navigation Timing; 0 when unavailable |
| `Consent` | `string` | Consent outcome, see [Consent Banners](/core-concepts-consent); omitted from JSON when empty |
| `Tree` | `[]*Node` | Set inside the `TypeJSON` string only; omitted from JSON when empty |

The struct has no JSON tags apart from `omitempty`, so field names serialize as written (`Href`, `FinalURL`, ...).

### Error

```go
type Error struct {
	Status int
	Href   string
}

func (e *Error) Error() string
```

`Error()` renders as `http {status}: {href}`. Returned for blocked responses and challenge pages (403), missing pages (404), empty extraction (204) and other statuses of 400 or higher when nothing could be extracted. Retrieve it with `errors.As`:

```go
var e *browser.Error
if errors.As(err, &e) && e.Status == 403 {
	// blocked even after the headed retry, or no GUI available
}
```

## Node

| Field | JSON key | Notes |
|---|---|---|
| `Type` | `type` | See the table below |
| `Text` | `text` | Content of `text` nodes |
| `Level` | `level` | 1–6 for `heading` |
| `Datetime` | `datetime` | The `datetime` attribute of `time` |
| `Href` | `href` | Absolute URL of `link` |
| `Src` / `Alt` | `src` / `alt` | Attributes of `image` |
| `Children` | `children` | Child nodes |

| `Type` | Source element |
|---|---|
| `heading` | `h1`–`h6` |
| `paragraph` | `p` |
| `list` / `ordered_list` / `list_item` | `ul` / `ol` / `li` |
| `link` | `a`, with `keepLinks` |
| `image` | `img`, with `keepLinks` (removed later, see [Content Conversion](/api-reference-conversion)) |
| `bold` / `italic` | `strong`, `b` / `em`, `i` |
| `code` / `code_block` | `code` / `pre` |
| `blockquote` | `blockquote` |
| `time` | `time` |
| `linebreak` | `br` |
| `container` | `div`, `section`, `article`, `main`, `table`, `tr`, `td`, `figure` and other block elements |
| `nav` / `header` / `footer` / `aside` | Same elements, with `keepLinks` |
| `text` | Non-empty text |

## Other exports

### Browser

```go
type Browser struct {
	// unexported fields
}
```

The cache entry wrapping a go-rod browser and its last-use time. It is exported but has no exported fields or methods, and no function returns it; callers have no use for it.

### Constants and variables

| Name | Value | Notes |
|---|---|---|
| `TypeMarkdown` | `0` | Default output format |
| `TypeHTML` | `1` | Merged HTML |
| `TypeJSON` | `2` | Serialized `Result` with `Tree` |
| `DefaultUserAgent` | Chrome 124 on Linux x86_64 | Used when `Option.UserAgent` is empty |
| `ErrProfileNotFound` | `chrome profile not found` | Sentinel for a missing Chrome profile, matched with `errors.Is`; `Fetch` handles it internally by falling back |
| `ErrChallengeRequired` | `[Denied] challenge: manual verification required` | Headed mode detected a challenge with waiting disabled; match with `errors.Is` |
| `ErrChallenge` | `[Timeout] challenge: not completed by the user` | The manual verification wait expired; match with `errors.Is` |
