# 執行環境

go-browser 從主機讀取的環境變數、Chrome 執行檔位置與設定檔路徑。

## 環境變數

| 變數 | 影響 |
|---|---|
| `DISPLAY` | Linux 上非空即視為具備 GUI，啟用有頭重試、`Option.Visible` 與 session 網域的有頭模式 |
| `WAYLAND_DISPLAY` | 同上 |

macOS 無條件視為具備 GUI。兩者皆未設定的 Linux 主機只會使用 headless，不會因為嘗試啟動有頭瀏覽器而失敗。go-browser 本身不讀取其他變數；`HOME` 與 `TMPDIR` 會在取得設定檔根目錄與建立暫存設定檔時被間接使用，go-rod 另會讀取自身的 `-rod` 旗標。

## Chrome 位置

### 執行檔搜尋

| 平台 | 順序 |
|---|---|
| macOS | `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome` → `/Applications/Chromium.app/Contents/MacOS/Chromium` |
| Linux | `google-chrome` → `google-chrome-stable` → `chromium` → `chromium-browser`（`PATH` 內） |

都找不到時交由 go-rod 決定瀏覽器來源，必要時自行下載。

### 設定檔根目錄

| 平台 | 路徑 |
|---|---|
| macOS | `~/Library/Application Support/Google/Chrome` |
| Linux | `~/.config/google-chrome` |

`SameSession` 使用以 `Option.Profile` 命名的子目錄。其他平台或目錄不存在時，呼叫改以不帶 Cookie 繼續。

## 外部指令

只有 `SameSession` 會執行外部指令：

| 指令 | 平台 | 用途 |
|---|---|---|
| `security` | macOS | 從鑰匙圈讀取 Chrome Safe Storage 密碼 |
| `secret-tool` | Linux | 從 gnome-keyring 的 Secret Service 讀取密碼 |
| `kwallet-query` | Linux | 從 KWallet（KDE）讀取密碼 |

## 容器

在 Docker 或 CI 中，把 Chrome 或 Chromium 裝在 `PATH` 上，不設 `DISPLAY` 以維持 headless，被擋的站點會直接回傳錯誤而不做有頭重試。`no-sandbox` 與 `disable-dev-shm-usage` 一律開啟，`/dev/shm` 過小不成問題。
