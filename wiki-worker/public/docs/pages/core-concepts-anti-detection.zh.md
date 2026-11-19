# 反偵測

go-browser 為了讓自動化的 Chrome 更接近一般瀏覽器做了哪些調整。

## 啟動參數

| 參數 | 效果 |
|---|---|
| `disable-blink-features=AutomationControlled` | Chrome 不透過 Blink 宣告自動化 |
| `user-agent` | `Option.UserAgent`，或 `DefaultUserAgent`（Linux x86_64 上的 Chrome 124） |
| `window-size=1280,960` | 桌面尺寸視窗，與預設 viewport 一致 |
| `window-position=-32000,-32000` | 僅有頭視窗：放在螢幕外 |

## 注入的 stealth 腳本

`Option.StealthJS` 以 `EvalOnNewDocument` 註冊，因此在分頁載入的每份文件中都早於頁面腳本執行。內嵌預設值做四件事：

| 屬性 | 頁面看到的值 |
|---|---|
| `navigator.webdriver` | `undefined` |
| `navigator.plugins` | 三筆：Chrome PDF Plugin、Chrome PDF Viewer、Native Client |
| `navigator.languages` | `["en-US", "en"]` |
| `window.chrome` | `{ runtime: {} }` |

設定 `StealthJS` 會完全取代預設值，而不是附加在後面。

## 行為

### 擬人化捲動

每輪捲動等待 150ms 加最多 300ms 隨機抖動，並在 300ms 內平滑捲動而非跳轉，見 [頁面載入](/zh/core-concepts-snapshots)。

### 有頭降級

若偵測到需要真人驗證，請依 [Fetch 路由](/zh/core-concepts-routing#真人驗證) 的說明，讓使用者在瀏覽器中完成。
