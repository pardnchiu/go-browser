# go-browser - 技術文件

> 返回 [README](./README.zh.md)

## 前置需求

- Go 1.25 或更高版本
- Google Chrome 或 Chromium 瀏覽器（macOS 或 Linux）
- macOS：Chrome 安裝於 `/Applications/Google Chrome.app/`
- Linux：`google-chrome`、`google-chrome-stable`、`chromium` 或 `chromium-browser` 可在 `PATH` 中找到
- `sqlite3` 命令列工具（用於 Cookie 提取）
- macOS：`security` 命令列工具（系統內建，用於 Chrome Safe Storage 密碼）
- Linux：`secret-tool`（用於 Chrome Safe Storage 密碼，需安裝 `libsecret-tools`）

## 安裝

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

## 設定

### 環境變數

| 變數 | 必要 | 說明 |
|------|------|------|
| `DISPLAY` | 否 | Linux 上若需使用非 headless 模式，需設定此變數指向 X11 顯示 |
| `WAYLAND_DISPLAY` | 否 | Linux 上若需使用非 headless 模式，可設定此變數指向 Wayland 顯示 |

### Chrome 設定檔

本函式庫會自動偵測 Chrome 設定檔路徑：

| 平台 | 路徑 |
|------|------|
| macOS | `~/Library/Application Support/Google/Chrome` |
| Linux | `~/.config/google-chrome` |

預設使用名為 `Default` 的設定檔。若需使用其他設定檔，透過 `Option.Profile` 指定。

## 使用方式

### 基礎：擷取頁面為 Markdown

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
        Type:      goBrowser.TypeMarkdown,
        Headless:   true,
        ScrollCount: 3,
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Title)
    fmt.Println(result.Content)
}
```

### 進階：使用 Cookie 會話存取登入頁面

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

### 互動式分頁操作

```go
// 建立分頁並導航
tabID, err := goBrowser.CreateTab(ctx, "https://example.com", &goBrowser.Option{
    SameSession: true,
    Headless:    false,
})
if err != nil {
    panic(err)
}
defer goBrowser.CloseTab(tabID)

// 點擊元素
if err := goBrowser.TabClick(tabID, "#login-button"); err != nil {
    panic(err)
}

// 輸入文字
if err := goBrowser.TabType(tabID, "#username", "myuser"); err != nil {
    panic(err)
}

// 滾動頁面
if err := goBrowser.TabScroll(tabID, 3); err != nil {
    panic(err)
}

// 擷取快照
result, err := goBrowser.TabSnapshot(tabID)
if err != nil {
    panic(err)
}
fmt.Println(result.Content)

// 執行自訂 JavaScript
output, err := goBrowser.TabEval(tabID, "document.title")
if err != nil {
    panic(err)
}
fmt.Println(output)
```

### 輸出格式

```go
// Markdown（預設）
result, _ := goBrowser.Fetch(ctx, url, timeout, &goBrowser.Option{Type: goBrowser.TypeMarkdown})

// HTML
result, _ := goBrowser.Fetch(ctx, url, timeout, &goBrowser.Option{Type: goBrowser.TypeHTML})

// JSON 樹狀結構
result, _ := goBrowser.Fetch(ctx, url, timeout, &goBrowser.Option{Type: goBrowser.TypeJSON})
```

## API 參考

### Fetch

```go
func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error)
```

擷取指定 URL 的內容。自動判斷是否需要 headless 模式或 Cookie 會話，並在遇到 403/429 時自動重試。

### CreateTab

```go
func CreateTab(ctx context.Context, href string, opt *Option) (string, error)
```

建立互動式分頁，回傳分頁 ID 供後續操作使用。

### TabClick

```go
func TabClick(tabID, selector string) error
```

在指定分頁中點擊符合 CSS 選擇器的元素。

### TabType

```go
func TabType(tabID, selector, text string) error
```

在指定分頁中向符合 CSS 選擇器的輸入框填入文字。

### TabScroll

```go
func TabScroll(tabID string, count int) error
```

在指定分頁中滾動頁面指定次數。

### TabNavigate

```go
func TabNavigate(tabID, href string) error
```

在現有分頁中導航至新 URL。

### TabEval

```go
func TabEval(tabID, js string) (string, error)
```

在指定分頁中執行 JavaScript 並回傳結果。

### TabSnapshot

```go
func TabSnapshot(tabID string) (*Result, error)
```

擷取指定分頁的當前頁面內容，依 `Option.Type` 輸出 Markdown、HTML 或 JSON。

### CloseTab

```go
func CloseTab(tabID string) error
```

關閉指定分頁並釋放資源。當所有分頁關閉時，互動式瀏覽器實例也會自動關閉。

### Close

```go
func Close()
```

關閉所有分頁與瀏覽器實例，釋放所有資源。

### SetMaxConcurrency

```go
func SetMaxConcurrency(n int)
```

設定最大並行擷取數量（預設為 8）。

### Option

| 欄位 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `IdleWait` | `time.Duration` | `2s` | 等待 DOM 穩定的時間 |
| `MaxLength` | `int` | `1MB` | 輸出內容最大長度（位元組） |
| `UserAgent` | `string` | Chrome 124 | 自訂 User-Agent |
| `KeepLinks` | `bool` | `false` | 是否保留連結與圖片 |
| `StealthJS` | `string` | 內建 | 自訂 stealth JavaScript |
| `SettleJS` | `string` | 內建 | 頁面載入後執行的 settle JavaScript |
| `Viewport` | `*Viewport` | `1280×960` | 視窗大小與裝置縮放比 |
| `SameSession` | `bool` | `false` | 使用 Chrome 設定檔的 Cookie 會話 |
| `Headless` | `bool` | `false` | 強制使用 headless 模式 |
| `Profile` | `string` | `"Default"` | Chrome 設定檔名稱 |
| `Type` | `int` | `TypeMarkdown` | 輸出格式：`TypeMarkdown`、`TypeHTML`、`TypeJSON` |
| `ScrollCount` | `int` | `3` | 模擬滾動次數 |

### Result

| 欄位 | 類型 | 說明 |
|------|------|------|
| `Href` | `string` | 原始請求 URL |
| `FinalURL` | `string` | 最終 URL（經過重導向） |
| `Content` | `string` | 提取的內容（Markdown / HTML / JSON） |
| `ContentType` | `string` | 頁面的 Content-Type |
| `Title` | `string` | 頁面標題 |
| `Author` | `string` | 文章作者 |
| `PublishedAt` | `string` | 發布時間（RFC3339 格式） |
| `Excerpt` | `string` | 文章摘要 |
| `Status` | `int` | HTTP 狀態碼 |
| `Tree` | `[]*Node` | JSON 模式下的結構化樹狀節點 |

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)