# API 參考

`package browser`（import path `github.com/pardnchiu/go-browser/core`）的擷取與生命週期函式，以及所有匯出符號的索引。

## Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

以 Chrome 載入 `href`，並依 `opt.Type` 選擇的格式回傳內容。

| 參數 | 說明 |
|---|---|
| `ctx` | 取消呼叫中的每個等待點，含 semaphore 與捲動延遲 |
| `href` | 必須有 scheme、hostname 含 `.`、不含帳密且不超過 2048 字元，否則回 `[Invalid] <url>: <原因>` |
| `timeout` | 大於 0 時每次嘗試的頁面載入各包一層 `ctx`（不含 `SameSession` 的 Cookie 擷取與瀏覽器啟動）；0 或以下表示不另設期限 |
| `opt` | 可為 `nil`；零值欄位採預設，見 [Option 欄位](/zh/configuration) |

依 `opt` 與執行環境選擇 headless 或有頭，僅在回應為 403／429／503 且主機有 GUI 時以有頭重試一次（[Fetch 路由](/zh/core-concepts-routing)）。錯誤為帶 HTTP 類狀態碼的 `*Error`，或包裝後的一般錯誤（[疑難排解](/zh/getting-started-troubleshooting)）。可在多個 goroutine 同時呼叫，併發由 `SetMaxConcurrency` 限制。

## Close

```go
func Close()
```

關閉全部快取的瀏覽器實例並清空快取。可重複呼叫；之後的 `Fetch` 會啟動新實例。

## SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

設定所有瀏覽器同時開啟的頁面上限，預設 8。`n <= 0` 時不做任何事。見 [瀏覽器快取](/zh/core-concepts-browser-cache)。

## ValidURL

```go
func ValidURL(href string) (*url.URL, error)
```

以 `Fetch` 相同的規則解析 `href`，但不檢查 scheme：不超過 2048 字元、不含帳密、hostname 含 `.`。錯誤格式為 `[Invalid] <url>: <原因>`；帳密會遮蓋，過長的網址只保留前 64 字元。

## IsErrorPageTitle

```go
func IsErrorPageTitle(title string) bool
```

`title` 含找不到頁面或拒絕存取字樣（如 `404`、`page not found`、`找不到頁面`）時回傳 `true`，不分大小寫。字樣清單在 `core/embed/error_page_titles.json`，涵蓋 16 種語言。

## 符號索引

| 符號 | 種類 | 頁面 |
|---|---|---|
| `Fetch`、`Close`、`SetMaxConcurrency`、`ValidURL`、`IsErrorPageTitle` | func | 本頁 |
| `HTMLToMarkdown`、`HTMLToNode`、`Merge`、`DedupTree`、`DedupMarkdownParagraphs`、`InlineTimeElements` | func | [內容轉換](/zh/api-reference-conversion) |
| `Option`、`Viewport`、`Result`、`Error`、`Node`、`Browser` | type | [型別](/zh/api-reference-types) |
| `TypeMarkdown`、`TypeHTML`、`TypeJSON`、`DefaultUserAgent`、`ErrProfileNotFound`、`ErrChallengeRequired`、`ErrChallenge` | const / var | [型別](/zh/api-reference-types) |

舊版曾存在的符號列於 [已移除 API](/zh/removed-api)。
