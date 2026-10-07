# go-browser - 技術文件

最後更新：2026-10-06

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.25 或更高版本
- Google Chrome 或 Chromium（macOS 或 Linux）
- macOS：Chrome 位於 `/Applications/Google Chrome.app/` 或 `/Applications/Chromium.app/`
- Linux：`PATH` 中可找到 `google-chrome`、`google-chrome-stable`、`chromium` 或 `chromium-browser`
- 使用 `SameSession` 時：
  - `sqlite3` 命令列工具（讀取 Cookies 資料庫）
  - macOS：內建 `security` 工具（讀取 Chrome Safe Storage 密碼）
  - Linux：`secret-tool`（`libsecret-tools`，讀取 Chrome Safe Storage 密碼）

## 安裝

### 使用 go get

```bash
go get github.com/pardnchiu/go-browser/core
```

### 從原始碼

```bash
git clone https://github.com/pardnchiu/go-browser.git
cd go-browser
go build ./...
```

## 設定

### 環境變數

| 變數 | 必要 | 說明 |
|------|------|------|
| `DISPLAY` | 否 | Linux 上存在時視為有顯示器，允許 headed 重試（X11） |
| `WAYLAND_DISPLAY` | 否 | Linux 上存在時視為有顯示器，允許 headed 重試（Wayland） |

macOS 一律視為有顯示器。Linux 兩者皆未設定時，不做 headed 重試，直接回傳 headless 結果。

### Chrome Profile

`SameSession` 從下列路徑讀取 Chrome profile：

| 平台 | 路徑 |
|------|------|
| macOS | `~/Library/Application Support/Google/Chrome/<Profile>` |
| Linux | `~/.config/google-chrome/<Profile>` |

`<Profile>` 預設為 `Default`，可由 `Option.Profile` 指定。找不到 profile 時自動退回一般快取瀏覽器，不回傳錯誤。

## 使用方式

### 基礎：擷取為 Markdown

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

`opt` 為 `nil` 時套用全部預設值：Markdown 輸出、headless 優先、捲動 3 次。

### 處理 HTTP 錯誤

```go
result, err := browser.Fetch(ctx, href, 30*time.Second, nil)
if err != nil {
	var httpErr *browser.Error
	if errors.As(err, &httpErr) {
		// 403：被擋或 Cloudflare 驗證頁；204：未萃取到內容
		log.Printf("status %d: %s", httpErr.Status, httpErr.Href)
		return
	}
	log.Fatal(err)
}
fmt.Println(result.Status, result.FinalURL)
```

### 輸出格式

```go
// Markdown（預設），受 MaxLength 截斷
md, err := browser.Fetch(ctx, href, timeout, &browser.Option{Type: browser.TypeMarkdown})
if err != nil {
	log.Fatal(err)
}

// HTML：合併所有捲動快照並內嵌 <time>
html, err := browser.Fetch(ctx, href, timeout, &browser.Option{Type: browser.TypeHTML})
if err != nil {
	log.Fatal(err)
}

// JSON：Content 為序列化後的 Result，含 Tree 結構樹
tree, err := browser.Fetch(ctx, href, timeout, &browser.Option{Type: browser.TypeJSON})
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(md.Content), len(html.Content), len(tree.Content))
```

回應 Content-Type 含 `json` 或 `xml` 時，不論 `Type` 皆原樣回傳內容，並填入 `ContentType`。

### 進階：以 Cookie Session 讀取需登入頁面

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

`SameSession: true` 會把 profile 的 Cookies 檔複製到暫存目錄、解密後注入一個用完即關的瀏覽器。點擊、填表等多步互動不在本函式庫範圍內，請改用 Playwright MCP。

### 進階：檢視 Cookie 橫幅處理結果

```go
result, err := browser.Fetch(ctx, href, 30*time.Second, nil)
if err != nil {
	log.Fatal(err)
}
switch result.Consent {
case "", "none", "skipped":
	fmt.Println("未關閉任何橫幅")
default:
	fmt.Println("已套用策略：", result.Consent)
}
```

### 併發與關閉

```go
browser.SetMaxConcurrency(4)
defer browser.Close()
```

## API 參考

### Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

以 Chrome 擷取指定 URL。`timeout <= 0` 時僅受 `ctx` 控制。`href` 需含 scheme 且主機名含 `.`，否則回傳 `invalid url`。

路由順序：

| 條件 | 行為 |
|------|------|
| `Option.Headless == true` | 僅 headless，不重試 |
| `Option.Visible == true` 或網域屬於內建社群清單（`facebook.com`、`x.com`、`linkedin.com` 等），且有顯示器 | 直接 headed |
| 其他 | 先 headless；回傳 403／429／503 且有顯示器時改 headed 重試 |

### SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

設定同時進行的頁面擷取上限（預設 8）。`n <= 0` 時忽略。

### Close

```go
func Close()
```

關閉所有快取中的瀏覽器實例。快取實例閒置超過 5 分鐘也會自動關閉。

### HTML 處理函式

| 函式 | 簽章 | 說明 |
|------|------|------|
| `Merge` | `func Merge(snapshots []string) (string, error)` | 合併多份 HTML 快照 |
| `InlineTimeElements` | `func InlineTimeElements(htmlSrc string) (string, error)` | 將 `<time>` 的 datetime 內嵌為文字 |
| `HTMLToMarkdown` | `func HTMLToMarkdown(content, baseURL string, keepLinks bool) (string, error)` | HTML 轉 Markdown |
| `HTMLToNode` | `func HTMLToNode(content, baseURL string, keepLinks bool) ([]*Node, error)` | HTML 轉結構樹 |
| `DedupMarkdownParagraphs` | `func DedupMarkdownParagraphs(md string) string` | 移除重複的 Markdown 段落 |
| `DedupTree` | `func DedupTree(nodes []*Node) []*Node` | 移除重複的節點 |

`Fetch` 內部已串接這些函式，亦可單獨使用。

### Option

| 欄位 | 型別 | 預設 | 說明 |
|------|------|------|------|
| `Type` | `int` | `TypeMarkdown` | `TypeMarkdown`／`TypeHTML`／`TypeJSON` |
| `Headless` | `bool` | `false` | `true` 強制 headless 且不做 headed 重試；`false` 為 headless 優先 |
| `Visible` | `bool` | `false` | `true` 在有顯示器時直接 headed；無顯示器時退回 headless 優先。`Headless` 為 `true` 時忽略 |
| `SameSession` | `bool` | `false` | 注入本機 Chrome profile 的 Cookie |
| `Profile` | `string` | `"Default"` | `SameSession` 使用的 Chrome profile 名稱 |
| `ScrollCount` | `int` | `3` | 捲動次數；負值視為 0；頁面不可捲動或快照未變時提前結束 |
| `IdleWait` | `time.Duration` | `2s` | 每階段等待 DOM 穩定的上限 |
| `MaxLength` | `int` | `1 << 20` | Markdown 輸出長度上限（位元組） |
| `KeepLinks` | `bool` | `false` | 保留連結與圖片 |
| `UserAgent` | `string` | `DefaultUserAgent` | 自訂 User-Agent；與 headless 共同作為瀏覽器快取鍵 |
| `StealthJS` | `string` | 內建 | 每個新文件載入前執行的腳本 |
| `SettleJS` | `string` | 內建 | DOM 穩定後執行的腳本 |
| `Viewport` | `*Viewport` | `1280x960` | 視窗大小 |

### Viewport

| 欄位 | 型別 | 說明 |
|------|------|------|
| `Width` | `int` | 寬度 |
| `Height` | `int` | 高度 |
| `DeviceScaleFactor` | `float64` | 縮放比例，`0` 視為 `1` |

### Result

| 欄位 | 型別 | 說明 |
|------|------|------|
| `Href` | `string` | 原始請求 URL |
| `FinalURL` | `string` | 導向後的最終 URL |
| `Content` | `string` | Markdown／HTML／JSON 字串，或原樣 JSON／XML |
| `ContentType` | `string` | 僅在 JSON／XML 原樣直出時填入 |
| `Title` | `string` | 文章標題（Markdown 模式） |
| `Author` | `string` | 文章作者（Markdown 模式） |
| `PublishedAt` | `string` | 發布時間，RFC3339（Markdown 模式） |
| `Excerpt` | `string` | 文章摘要（Markdown 模式） |
| `Status` | `int` | 導覽回應的 HTTP 狀態碼 |
| `Consent` | `string` | 已套用的橫幅關閉策略（`selector`／`text`／`removed`），多輪以 `,` 串接；`none`、`skipped`（偵測到付費牆或登入牆而略過）或空字串表示未關閉 |
| `Tree` | `[]*Node` | 結構樹（僅存在於 `TypeJSON` 序列化內容中） |

### Node

| 欄位 | 型別 | 說明 |
|------|------|------|
| `Type` | `string` | 節點類型 |
| `Text` | `string` | 文字內容 |
| `Level` | `int` | 標題層級 |
| `Datetime` | `string` | `<time>` 的 datetime |
| `Href` | `string` | 連結（`keepLinks` 時） |
| `Src` | `string` | 圖片來源（`keepLinks` 時） |
| `Alt` | `string` | 圖片替代文字（`keepLinks` 時） |
| `Children` | `[]*Node` | 子節點 |

### Error

```go
type Error struct {
	Status int
	Href   string
}
```

| Status | 觸發情境 |
|--------|----------|
| `403`／`404` | 最終 URL 路徑或 query 含 `403`／`404` 片段 |
| `403` | 頁面標題命中 Cloudflare／存取拒絕類字樣 |
| `204` | 轉換後 Markdown 為空 |
| `>= 400` | Readability 無法萃取且導覽狀態碼 ≥ 400 |

### 常數與變數

| 名稱 | 說明 |
|------|------|
| `TypeMarkdown` | Markdown 輸出 |
| `TypeHTML` | 合併後 HTML 輸出 |
| `TypeJSON` | 結構樹 JSON 輸出 |
| `DefaultUserAgent` | 預設 Chrome 124 Linux User-Agent |
| `ErrProfileNotFound` | 找不到 Chrome profile；`Fetch` 內部會自動退回，不對外回傳 |

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
