# 疑難排解

`Fetch` 回傳的每種錯誤代表什麼、從哪裡產生，以及是否會以有頭模式重試。

## 狀態碼錯誤

這類錯誤是 `*browser.Error`，用 `errors.As` 取出後讀 `Error.Status`。

| 錯誤 | 產生時機 | 有頭重試 |
|---|---|---|
| `http 403` | 導向後的 URL 路徑片段或 query 值為 `403`；第一份快照的標題命中驗證頁字樣（`just a moment`、`attention required`、`checking your browser`、`access denied`、`請稍候`）；或萃取不出文章且回應狀態為 403 | 預設分支且有 GUI 時會 |
| `http 429` / `http 503` | 萃取不出文章且回應狀態為 429 或 503 | 預設分支且有 GUI 時會 |
| `http 404` | 導向後的 URL 路徑片段或 query 值為 `404`，或萃取不出文章且狀態為 404 | 不會 |
| `http 204` | 萃取成功但去重後 Markdown 為空 | 不會 |
| 其他 `http 4xx/5xx` | 萃取不出文章且回應狀態 ≥ 400 | 不會 |

擷取前偵測到的真人驗證，在無頭模式會對所有輸出格式回傳 `403`。Readability 另有標題判斷、`204` 與萃取失敗的狀態碼錯誤，只適用 `TypeMarkdown` 和 `TypeJSON`。`Result.Status` 為 `403`、`429` 或 `503` 也會觸發預設路由的有頭重試（需有 GUI）。只有預設的無頭優先路由會重試。見 [Fetch 路由](/zh/core-concepts-routing)。

## 一般錯誤

有頭模式若回傳 `ErrChallengeRequired`，請在有 GUI 的環境設定正值 `ChallengeWait`，並在 Chrome 內手動完成驗證。若等待逾時則回傳 `ErrChallenge`；兩者均可用 `errors.Is` 判斷。

| 訊息 | 原因 |
|---|---|
| `url.Parse: ...` | URL 無法解析 |
| `[Invalid] <url>: ...` | URL 缺 scheme、hostname 不含 `.`、含帳密，或超過 2048 字元 |
| `readability: no article extracted from N snapshots` | 所有快照都解析不出文章，且狀態碼小於 400 |
| `acquireSem: context deadline exceeded` | 併發上限已滿直到 `timeout` 到期；見 [瀏覽器快取](/zh/core-concepts-browser-cache) |
| `launcher.Launch: ...` | Chrome 啟動失敗，多半是找不到執行檔或沙箱限制 |
| `extract cookies: security find-generic-password: ...` | macOS 鑰匙圈未解鎖或拒絕存取；僅 `SameSession` 會走到 |
| `[Denied] chrome safe storage: no v11 keyring password (gnome-keyring/kwallet): ...` | Linux 的 gnome-keyring 與 KWallet 都沒回傳 Chrome Safe Storage 密碼；僅 `SameSession` |
| `[Denied] chrome safe storage: N keyring password(s) failed to decrypt v11 cookies` | 找到 keyring 密碼但都解不開 `v11` Cookie（domain-hash 不符）；僅 `SameSession` |
| `[Unavailable] cookies db: ...` / `[Upstream] cookies db: ...` | 複製出的 Cookie 資料庫無法開啟或查詢；僅 `SameSession` |
| `inject cookies (0/N): ...` | 所有 Cookie 都注入失敗，逐筆重試也一樣 |

Chrome 設定檔不存在時內部會包裝 `browser.ErrProfileNotFound`，但 `Fetch` 不會回傳它：呼叫改以不帶 Cookie 的一般瀏覽器繼續。上表其他 `SameSession` 錯誤則直接回傳，不會降級。

## 無桌面伺服器

`DISPLAY` 與 `WAYLAND_DISPLAY` 皆未設定的 Linux 主機不會啟動有頭瀏覽器，被擋的回應會直接以錯誤回傳而不重試。見 [執行環境](/zh/configuration-environment)。
