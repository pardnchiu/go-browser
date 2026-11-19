# 快速開始

從 `go get` 到第一次取得 Markdown：前置需求、安裝、一次 `Fetch` 呼叫，以及關閉瀏覽器。

## 前置需求

| 項目 | 需求 | 為何 |
|---|---|---|
| Go | 1.25.1 以上 | `go.mod` 宣告的版本 |
| 作業系統 | macOS 或 Linux | Chrome 路徑搜尋與 Cookie 解密只實作這兩個平台；其他平台仍可以 headless 執行 `Fetch`，由 go-rod 自行尋找瀏覽器，但沒有 `SameSession` 與有頭重試 |
| Chrome / Chromium | 已安裝 | macOS 找 `/Applications/Google Chrome.app` 或 `Chromium.app`；Linux 依序找 `google-chrome`、`google-chrome-stable`、`chromium`、`chromium-browser` |
| `security`（macOS） | 系統內建 | 取得 Chrome Safe Storage 密碼，僅 `SameSession` 使用 |
| `secret-tool` 與／或 `kwallet-query`（Linux） | `libsecret-tools` / `kwallet` | 從 gnome-keyring 或 KWallet 取得 Chrome Safe Storage 密碼，僅 `SameSession` 使用 |

找不到 Chrome 可執行檔時，go-rod 會改用自行下載的瀏覽器；`SameSession` 仍需要本機 Chrome 設定檔才有 Cookie 可讀。

## 安裝

```bash
go get github.com/pardnchiu/go-browser/core
```

套件位於 `core` 子目錄，package 名稱為 `browser`：

```go
import browser "github.com/pardnchiu/go-browser/core"
```

## 第一次呼叫

```go
package main

import (
	"context"
	"fmt"
	"time"

	browser "github.com/pardnchiu/go-browser/core"
)

func main() {
	defer browser.Close()

	result, err := browser.Fetch(context.Background(), "https://example.com", 30*time.Second, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Title)
	fmt.Println(result.Content)
}
```

`opt` 傳 `nil` 即採全部預設值：headless 優先、最多捲動 3 次、Markdown 輸出、2 秒 idle 等待。URL 必須有 scheme 且 hostname 含 `.`，因此 `http://localhost:8080` 會回傳 `invalid url`。

### 結束前呼叫 Close

瀏覽器實例會被快取重用，並由背景 evictor 在閒置 5 分鐘後回收。程式結束前呼叫 `browser.Close()` 會立即關閉全部快取實例；不呼叫則 Chrome 行程會留到 evictor 回收或程序結束為止。

## 下一步

- [疑難排解](/zh/getting-started-troubleshooting)：每種錯誤代表什麼
- [輸出格式](/zh/configuration-output-formats)：Markdown、HTML 或 JSON 樹
- [Cookie 工作階段](/zh/core-concepts-cookie-session)：讀取需登入的頁面
- [Option 欄位](/zh/configuration)：`Option` 全部欄位與預設值
