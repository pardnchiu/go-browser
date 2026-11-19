# 已移除 API

舊版曾存在、但已不在相同 import path 下的每個匯出符號，附移除版本與替代做法。

## v0.3.2：互動分頁 API 與套件更名

v0.3.2 把函式庫收斂為單次 `Fetch` 萃取。Go package 名稱同時由 `goBrowser` 改為 `browser`；import path `github.com/pardnchiu/go-browser/core` 不變，只需更新識別字與 import 別名。

| 符號 | v0.3.1 簽名 | 狀態 |
|---|---|---|
| `CreateTab` | `func CreateTab(ctx context.Context, href string, opt *Option) (string, error)` | 移除於 v0.3.2；無公開替代，單次讀取改用 `Fetch` |
| `CloseTab` | `func CloseTab(tabID string) error` | 移除於 v0.3.2；無公開替代 |
| `TabSnapshot` | `func TabSnapshot(tabID string) (*Result, error)` | 移除於 v0.3.2；`Fetch` 直接回傳萃取內容 |
| `TabClick` | `func TabClick(tabID, selector string) error` | 移除於 v0.3.2；無公開替代，請改用完整 CDP 用戶端 |
| `TabType` | `func TabType(tabID, selector, text string) error` | 移除於 v0.3.2；無公開替代，請改用完整 CDP 用戶端 |
| `TabScroll` | `func TabScroll(tabID string, count int) error` | 移除於 v0.3.2；`Fetch` 內的捲動由 `Option.ScrollCount` 控制 |
| `TabNavigate` | `func TabNavigate(tabID, href string) error` | 移除於 v0.3.2；以新 URL 呼叫 `Fetch` |
| `TabEval` | `func TabEval(tabID, js string) (string, error)` | 移除於 v0.3.2；`Option.SettleJS` 每次呼叫執行一段腳本，但不回傳結果 |

`Close` 不再關閉共用的互動工作階段，該工作階段已隨分頁 API 一起移除。

## v0.2.0：套件搬到 core

v0.2.0 把整個套件從 module 根目錄 `github.com/pardnchiu/go-browser` 搬到 `github.com/pardnchiu/go-browser/core`。下列符號都自根路徑移除，並以相同名稱延續於 `core`，另有註明者除外。

| 符號 | 種類 | 狀態 |
|---|---|---|
| `Fetch`、`Close`、`SetMaxConcurrency` | func | 移除於 v0.2.0（根路徑）；`core` 下同名 |
| `HTMLToMarkdown`、`HTMLToNode`、`Merge`、`DedupTree`、`DedupMarkdownParagraphs`、`InlineTimeElements` | func | 移除於 v0.2.0（根路徑）；`core` 下同名 |
| `Option`、`Result`、`Error`、`Node`、`Viewport`、`Browser` | type | 移除於 v0.2.0（根路徑）；`core` 下同名 |
| `Error.Error` | method | 移除於 v0.2.0（根路徑）；`core` 下同名方法 |
| `DefaultUserAgent` | const | 移除於 v0.2.0（根路徑）；`core` 下同名 |
| `TypeMarkdown`、`TypeHTML`、`TypeJSON` | const | 移除於 v0.2.0（根路徑）；`core` 下同名 |
| `CreateTab`、`CloseTab`、`TabSnapshot`、`TabClick`、`TabType`、`TabScroll`、`TabNavigate`、`TabEval` | func | 移除於 v0.2.0（根路徑）；搬到 `core` 後再移除於 v0.3.2 |

## 遷移

```go
// v0.1.x
import goBrowser "github.com/pardnchiu/go-browser"

// v0.2.0 – v0.3.1
import goBrowser "github.com/pardnchiu/go-browser/core"

// v0.3.2 起
import browser "github.com/pardnchiu/go-browser/core"

result, err := browser.Fetch(ctx, href, 30*time.Second, &browser.Option{SameSession: true})
```

各版本的發行說明見 [版本紀錄](/released/)。
