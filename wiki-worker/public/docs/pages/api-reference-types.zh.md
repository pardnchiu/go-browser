# 型別

`package browser` 匯出的型別、常數與變數，以及各欄位說明。

## 呼叫相關型別

### Option

單次呼叫的設定。含預設值的完整欄位表見 [Option 欄位](/zh/configuration)。

### Viewport

```go
type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}
```

以 CSS 像素表示的頁面尺寸。`DeviceScaleFactor` 為 0 時視為 1。

### Result

| 欄位 | 型別 | 說明 |
|---|---|---|
| `Href` | `string` | 請求的 URL |
| `FinalURL` | `string` | 導向後的 URL |
| `Content` | `string` | 依 `Type` 與內容型別為 Markdown、HTML、JSON 字串或原始 JSON／XML |
| `ContentType` | `string` | 僅 JSON 與 XML 文件填入；為空時不出現在 JSON |
| `Title` | `string` | readability 標題，僅 `TypeMarkdown` |
| `Author` | `string` | readability byline，僅 `TypeMarkdown` |
| `PublishedAt` | `string` | readability 找得到日期時為 RFC3339 |
| `Excerpt` | `string` | readability 摘要，僅 `TypeMarkdown` |
| `Status` | `int` | 取自 Navigation Timing 的 `responseStatus`；取不到時為 0 |
| `Consent` | `string` | 同意橫幅結果，見 [同意橫幅](/zh/core-concepts-consent)；為空時不出現在 JSON |
| `Tree` | `[]*Node` | 只在 `TypeJSON` 字串內填入；為空時不出現在 JSON |

除 `omitempty` 外沒有 JSON tag，欄位名稱照原樣序列化（`Href`、`FinalURL`……）。

### Error

```go
type Error struct {
	Status int
	Href   string
}

func (e *Error) Error() string
```

`Error()` 輸出 `http {status}: {href}`。用於被擋的回應與驗證頁（403）、頁面不存在（404）、萃取為空（204），以及萃取不出內容時其他 ≥ 400 的狀態。以 `errors.As` 取出：

```go
var e *browser.Error
if errors.As(err, &e) && e.Status == 403 {
	// 有頭重試後仍被擋，或沒有 GUI
}
```

## Node

| 欄位 | JSON key | 說明 |
|---|---|---|
| `Type` | `type` | 見下表 |
| `Text` | `text` | `text` 節點內容 |
| `Level` | `level` | `heading` 的 1–6 |
| `Datetime` | `datetime` | `time` 的 `datetime` 屬性 |
| `Href` | `href` | `link` 的絕對 URL |
| `Src` / `Alt` | `src` / `alt` | `image` 的屬性 |
| `Children` | `children` | 子節點 |

| `Type` | 來源元素 |
|---|---|
| `heading` | `h1`–`h6` |
| `paragraph` | `p` |
| `list` / `ordered_list` / `list_item` | `ul` / `ol` / `li` |
| `link` | `a`，需 `keepLinks` |
| `image` | `img`，需 `keepLinks`（之後會被移除，見 [內容轉換](/zh/api-reference-conversion)） |
| `bold` / `italic` | `strong`、`b` / `em`、`i` |
| `code` / `code_block` | `code` / `pre` |
| `blockquote` | `blockquote` |
| `time` | `time` |
| `linebreak` | `br` |
| `container` | `div`、`section`、`article`、`main`、`table`、`tr`、`td`、`figure` 等區塊元素 |
| `nav` / `header` / `footer` / `aside` | 同名元素，需 `keepLinks` |
| `text` | 非空文字 |

## 其他匯出

### Browser

```go
type Browser struct {
	// unexported fields
}
```

包裝 go-rod 瀏覽器與最後使用時間的快取項目。雖然匯出，但沒有匯出欄位或方法，也沒有函式回傳它；呼叫端用不到。

### 常數與變數

| 名稱 | 值 | 說明 |
|---|---|---|
| `TypeMarkdown` | `0` | 預設輸出格式 |
| `TypeHTML` | `1` | 合併後的 HTML |
| `TypeJSON` | `2` | 含 `Tree` 的序列化 `Result` |
| `DefaultUserAgent` | Linux x86_64 上的 Chrome 124 | `Option.UserAgent` 為空時使用 |
| `ErrProfileNotFound` | `chrome profile not found` | Chrome 設定檔不存在的哨兵錯誤，以 `errors.Is` 判斷；`Fetch` 內部會自行降級處理 |
| `ErrChallengeRequired` | `[Denied] challenge: manual verification required` | 有頭模式偵測到真人驗證，但未啟用等待；以 `errors.Is` 判斷 |
| `ErrChallenge` | `[Timeout] challenge: not completed by the user` | 真人驗證等待逾時；以 `errors.Is` 判斷 |
