# go-browser

> [Info]
> 自 [`pardnchiu/go-pkg@v0.13.0`](https://github.com/pardnchiu/go-pkg) 起獨立出來維護。
>
> English: [README.md](../README.md)

Chromium 抓取網頁 + 互動式 tab 工具。

- **one-shot fetch**：`Fetch` → readability 主文 → markdown / HTML / JSON
- **interactive tabs**：`CreateTab` / `TabClick` / `TabType` / `TabScroll` / `TabNavigate` / `TabEval` / `TabSnapshot` / `CloseTab`
- **共用真人 Chrome 登入態**（macOS / Linux）：cookie 自解 + CDP 注入，不需改 Chrome 啟動方式
- **headless-first + headed fallback**：預設先嘗試無頭抓取，遇 403/429/503/反爬頁再自動切換有頭 + session 重試
- **反爬蟲偵測**：Cloudflare / Akamai challenge 頁面自動判定為 403 並提前失敗
- **併發控制**：`SetMaxConcurrency` 限制同時抓取數；閒置瀏覽器自動回收

## 安裝

```bash
go get github.com/pardnchiu/go-browser
```

## 快速開始

```go
import goBrowser "github.com/pardnchiu/go-browser/core"

defer goBrowser.Close()

result, err := goBrowser.Fetch(ctx, "https://example.com/article", 30*time.Second, nil)
// result.Content / result.Title / result.Author / result.PublishedAt / result.FinalURL / result.Status
```

## Option

```go
type Option struct {
	IdleWait    time.Duration // DOM 穩定等待時間，預設 2s
	MaxLength   int           // 輸出內容長度上限（bytes），預設 1MiB，0 = 不限制
	UserAgent   string        // 自訂 User-Agent
	KeepLinks   bool          // markdown 輸出是否保留連結
	StealthJS   string        // 自訂反偵測注入腳本，預設內建 stealth.js
	SettleJS    string        // 自訂等待腳本，預設內建 listener.js
	Viewport    *Viewport     // 視窗尺寸，預設 1280x960
	SameSession bool          // 用真人 Chrome profile 啟動（帶 cookie/session）
	Headless    bool          // true = 略過 headless 嘗試，直接以有頭瀏覽器 + session 開啟
	Profile     string        // Chrome profile 名稱，預設 "Default"
	Type        int           // TypeMarkdown(0) / TypeHTML(1) / TypeJSON(2)
	ScrollCount int           // 抓取時捲動次數，預設 3
}

type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}
```

## Fetch（one-shot）

```go
// 預設：headless 抓取，遇反爬蟲自動 fallback 到有頭 + session
result, err := goBrowser.Fetch(ctx, "https://example.com/article", 30*time.Second, nil)

// 共用真人 Chrome 登入 session（Darwin / Linux）
result, err = goBrowser.Fetch(ctx, "https://x.com/home", 30*time.Second, &goBrowser.Option{
	SameSession: true,
	Profile:     "Profile 1", // 預設 "Default"
})

// 強制一律以有頭瀏覽器 + session 開啟，不嘗試 headless
result, err = goBrowser.Fetch(ctx, "https://example.com/login-required", 30*time.Second, &goBrowser.Option{
	Headless: true,
})

// 切換輸出格式 / 自訂 scroll 次數
result, err = goBrowser.Fetch(ctx, "https://example.com", 30*time.Second, &goBrowser.Option{
	Type:        goBrowser.TypeJSON, // TypeMarkdown(0) / TypeHTML(1) / TypeJSON(2)
	ScrollCount: 5,
})

// 自訂 viewport / user-agent / 內容長度上限
result, err = goBrowser.Fetch(ctx, "https://example.com", 30*time.Second, &goBrowser.Option{
	Viewport:  &goBrowser.Viewport{Width: 1920, Height: 1080, DeviceScaleFactor: 2},
	UserAgent: "Mozilla/5.0 ...",
	MaxLength: 500 * 1024,
	KeepLinks: true,
})
```

### Result

```go
type Result struct {
	Href        string
	FinalURL    string
	Content     string
	ContentType string  // 非空 = JSON/XML 原始回應，未經 readability 解析
	Title       string
	Author      string
	PublishedAt string  // RFC3339
	Excerpt     string
	Status      int
	Tree        []*Node // 僅 Type == TypeJSON 時填入
}
```

`Content-Type` 為 `application/json` 或 `*/xml` 的回應會直接回傳原始內容（`ContentType` 非空），不經 readability／markdown 轉換。

### 錯誤處理

```go
var fe *goBrowser.Error
if errors.As(err, &fe) {
	_ = fe.Status // 404 / 403 / 429 / 503 / 204 ...
}
```

- `403`：HTTP 403，或偵測到 Cloudflare/Akamai 等反爬蟲 challenge 頁面
- `429` / `503`：會觸發 headless → headed+session 自動重試（僅當有 display 時）
- `204`：readability 解析出的 markdown 為空
- `404`：URL path/query 命中 404 片段

## 互動式 Tab

每個函式可獨立作為 LLM tool-call。

```go
tabID, err := goBrowser.CreateTab(ctx, "https://example.com/login", &goBrowser.Option{
	SameSession: true,
	Profile:     "Profile 1",
})
defer goBrowser.CloseTab(tabID)

snap, err := goBrowser.TabSnapshot(tabID)          // 目前頁面內容（同 Fetch 的 Result）
err = goBrowser.TabType(tabID, "#email", "x@y.com")
err = goBrowser.TabType(tabID, "#password", "...")
err = goBrowser.TabClick(tabID, "#submit")
err = goBrowser.TabScroll(tabID, 3)                // count = 0 時用 Option.ScrollCount
snap, err = goBrowser.TabSnapshot(tabID)
err = goBrowser.TabNavigate(tabID, "https://example.com/dashboard")
title, err := goBrowser.TabEval(tabID, `document.title`)
```

同一時間所有 tab 共用一個瀏覽器實例；最後一個 tab 關閉後自動釋放瀏覽器與 profile snapshot。

## 併發與資源管理

```go
goBrowser.SetMaxConcurrency(4) // 預設 8，同時抓取數上限

defer goBrowser.Close()        // 程式結束前釋放共用 headless 瀏覽器
```

閒置超過 1 分鐘的共用瀏覽器會自動關閉並回收。

> [Info]
> 這個專案是閱讀代碼後由 Claude Code 自動生成
