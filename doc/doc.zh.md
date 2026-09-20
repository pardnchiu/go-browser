# go-browser - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.25 或更高版本
- Google Chrome 或 Chromium（macOS 或 Linux）
- macOS：Chrome 安裝於 `/Applications/Google Chrome.app/`
- Linux：`PATH` 中可找到 `google-chrome`、`google-chrome-stable`、`chromium` 或 `chromium-browser`
- `sqlite3` 命令列工具（Cookie 擷取用）
- macOS：內建 `security` 工具（讀取 Chrome Safe Storage 密碼）
- Linux：`secret-tool`（需 `libsecret-tools`，讀取 Chrome Safe Storage 密碼）

## 安裝

### 從原始碼

```bash
git clone https://github.com/pardnchiu/go-browser.git
cd go-browser
go build ./...
```

### 使用 go get

```bash
go get github.com/pardnchiu/go-browser
```

## 設定

### 環境變數

| 變數 | 必要 | 說明 |
|------|------|------|
| `DISPLAY` | 否 | Linux 上設定後可走 X11 headed 模式 |
| `WAYLAND_DISPLAY` | 否 | Linux 上設定後可走 Wayland headed 模式 |

### Chrome Profile

函式庫會自動偵測 Chrome profile 路徑：

| 平台 | 路徑 |
|------|------|
| macOS | `~/Library/Application Support/Google/Chrome` |
| Linux | `~/.config/google-chrome` |

預設 profile 名稱為 `Default`。若要改用其他 profile，透過 `Option.Profile` 指定。

## 使用方式

### 基礎：擷取為 Markdown

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
	// Consent 可能為 none / skipped / 策略名稱；僅表示有嘗試關閉橫幅
	fmt.Println(result.Consent)
}
```

### 進階：以 Cookie Session 讀取需登入頁面

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

`SameSession: true` 會從本機 Chrome profile 複製並解密 Cookies，注入臨時瀏覽器後再導覽。完整 CDP 互動（點擊、填表、多步工作流）不在本函式庫範圍內，請改用 Playwright MCP。

### 輸出格式

```go
// Markdown（預設）
result, err := browser.Fetch(ctx, url, timeout, &browser.Option{Type: browser.TypeMarkdown})
if err != nil {
	panic(err)
}

// HTML（合併捲動快照）
result, err = browser.Fetch(ctx, url, timeout, &browser.Option{Type: browser.TypeHTML})
if err != nil {
	panic(err)
}

// JSON 結構樹
result, err = browser.Fetch(ctx, url, timeout, &browser.Option{Type: browser.TypeJSON})
if err != nil {
	panic(err)
}
```

### 併發與關閉

```go
browser.SetMaxConcurrency(4) // 預設 8
defer browser.Close()        // 關閉所有快取中的瀏覽器實例
```

## API 參考

### Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

以 Chrome 擷取指定 URL 內容。預設先嘗試 headless；若回傳 403/429/503 且環境有顯示器，才改 headed 重試。特定需 session 的網域會優先走 Cookie 路徑。

### SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

設定同時進行的最大擷取數（預設 8）。`n <= 0` 時忽略。

### Close

```go
func Close()
```

關閉所有快取的瀏覽器實例並釋放資源。

### Merge / Dedup / HTML 轉換

```go
func Merge(snapshots []string) (string, error)
func DedupTree(nodes []*Node)
func DedupMarkdownParagraphs(md string) string
func InlineTimeElements(htmlSrc string) (string, error)
func HTMLToNode(content, baseURL string, keepLinks bool) ([]*Node, error)
func HTMLToMarkdown(content, baseURL string, keepLinks bool) (string, error)
```

多快照合併、段落／節點去重，以及 HTML → Markdown／結構樹轉換。`Fetch` 內部已使用這些函式；亦可單獨呼叫。

### Option

| 欄位 | 型別 | 預設 | 說明 |
|------|------|------|------|
| `IdleWait` | `time.Duration` | `2s` | 等待 DOM 穩定的時間 |
| `MaxLength` | `int` | `1MB` | Markdown 輸出長度上限（位元組） |
| `UserAgent` | `string` | Chrome 124 | 自訂 User-Agent；亦為瀏覽器快取鍵之一 |
| `KeepLinks` | `bool` | `false` | 是否保留連結與圖片 |
| `StealthJS` | `string` | 內建 | 自訂 stealth 腳本 |
| `SettleJS` | `string` | 內建 | 頁面載入後執行的 JS |
| `Viewport` | `*Viewport` | `1280x960` | 視窗大小與 device scale |
| `SameSession` | `bool` | `false` | 使用本機 Chrome profile Cookie |
| `Headless` | `bool` | `false` | 強制 headless（為 true 時不做 headed fallback） |
| `Profile` | `string` | `"Default"` | Chrome profile 名稱 |
| `Type` | `int` | `TypeMarkdown` | `TypeMarkdown` / `TypeHTML` / `TypeJSON` |
| `ScrollCount` | `int` | `3` | 捲動模擬次數；負值視為 0 |

### Result

| 欄位 | 型別 | 說明 |
|------|------|------|
| `Href` | `string` | 原始請求 URL |
| `FinalURL` | `string` | 導向後最終 URL |
| `Content` | `string` | 擷取內容（Markdown / HTML / JSON 字串） |
| `ContentType` | `string` | 頁面 Content-Type（JSON/XML 直出時） |
| `Title` | `string` | 頁面標題 |
| `Author` | `string` | 文章作者 |
| `PublishedAt` | `string` | 發布時間（RFC3339） |
| `Excerpt` | `string` | 文章摘要 |
| `Status` | `int` | HTTP 狀態碼 |
| `Consent` | `string` | Cookie 同意橫幅嘗試結果（可選） |
| `Tree` | `[]*Node` | JSON 模式的結構樹節點 |

### 常數

| 常數 | 說明 |
|------|------|
| `TypeMarkdown` | Markdown 輸出 |
| `TypeHTML` | 合併後 HTML 輸出 |
| `TypeJSON` | 結構樹 JSON 輸出 |
| `DefaultUserAgent` | 預設 Chrome 124 User-Agent |
| `ErrProfileNotFound` | 找不到指定 Chrome profile |

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
