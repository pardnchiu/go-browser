# go-browser

> [Info]
> 自 [`pardnchiu/go-pkg@v0.13.0`](https://github.com/pardnchiu/go-pkg) 起獨立出來維護。

Chromium 抓取網頁 + 互動式 tab 工具。

- **one-shot fetch**：`Fetch` → readability 主文 → markdown / HTML / JSON
- **interactive tabs**：`CreateTab` / `TabClick` / `TabType` / `TabScroll` / `TabNavigate` / `TabEval` / `TabSnapshot` / `CloseTab`
- **共用真人 Chrome 登入態**（macOS / Linux）：cookie 自解 + CDP 注入，不需改 Chrome 啟動方式

## 安裝

```bash
go get github.com/pardnchiu/go-browser
```

## 範例

```go
import goBrowser "github.com/pardnchiu/go-browser/core"

defer goBrowser.Close()

// 一次性抓取
result, err := goBrowser.Fetch(ctx, "https://example.com/article", 30*time.Second, nil)
// result.Content / result.Title / result.Author / result.PublishedAt / result.FinalURL / result.Status

// 共用真人 Chrome 登入 session（Darwin / Linux）
result, err = goBrowser.Fetch(ctx, "https://x.com/home", 30*time.Second, &goBrowser.Option{
	SameSession: true,
	Profile:     "Profile 1", // 預設 "Default"
})

// 切換輸出格式 / 自訂 scroll 次數
result, err = goBrowser.Fetch(ctx, "https://example.com", 30*time.Second, &goBrowser.Option{
	Type:        goBrowser.TypeJSON, // TypeMarkdown(0) / TypeHTML(1) / TypeJSON(2)
	ScrollCount: 5,
})

// 互動式 tab（每個函式可獨立 LLM tool-call）
tabID, _ := goBrowser.CreateTab(ctx, "https://example.com/login", &goBrowser.Option{
	SameSession: true,
	Profile:     "Profile 1",
})
defer goBrowser.CloseTab(tabID)

snap, _ := goBrowser.TabSnapshot(tabID)
goBrowser.TabType(tabID, "#email", "x@y.com")
goBrowser.TabType(tabID, "#password", "...")
goBrowser.TabClick(tabID, "#submit")
goBrowser.TabScroll(tabID, 3)
snap, _ = goBrowser.TabSnapshot(tabID)
goBrowser.TabNavigate(tabID, "https://example.com/dashboard")
title, _ := goBrowser.TabEval(tabID, `document.title`)

// HTTP 錯誤分流
var fe *goBrowser.Error
if errors.As(err, &fe) {
	_ = fe.Status // 404 / 503 / 204 / 403
}
```
