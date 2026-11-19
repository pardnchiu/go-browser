# 核心概念

一次 `Fetch` 依序執行的步驟，從 URL 驗證到回傳 `Result`，並連到說明各步驟的頁面。

## 擷取流程

| # | 步驟 | 內容 | 細節 |
|---|---|---|---|
| 1 | 驗證 | 補齊 `Option` 預設值；URL 必須有 scheme 且 hostname 含 `.` | [Option 欄位](/zh/configuration) |
| 2 | 路由 | 依 `Option.Headless`、`Option.Visible`、session 網域清單與 GUI 有無決定 headless 或有頭 | [Fetch 路由](/zh/core-concepts-routing) |
| 3 | 取得瀏覽器 | 以 `(headless, userAgent)` 為鍵的快取實例；設 `SameSession` 時改用帶 Cookie 的一次性實例 | [瀏覽器快取](/zh/core-concepts-browser-cache)、[Cookie 工作階段](/zh/core-concepts-cookie-session) |
| 4 | 取得名額 | 頁面等待全域併發 semaphore | [瀏覽器快取](/zh/core-concepts-browser-cache) |
| 5 | 開啟頁面 | 空白分頁套用 viewport 與 stealth 腳本，接著導航並等待 `load`（上限 10 秒，逾時改看 `readyState`） | [反偵測](/zh/core-concepts-anti-detection)、[頁面載入](/zh/core-concepts-snapshots#load-事件) |
| 6 | 檢查 URL | 導向後 URL 的路徑片段或 query 值為 `404`／`403` 時以 `*Error` 結束 | [疑難排解](/zh/getting-started-troubleshooting) |
| 7 | 資料短路 | JSON 與 XML 文件直接回傳原始文字並填 `ContentType` | [輸出格式](/zh/configuration-output-formats) |
| 8 | 等待穩定 | DOM 穩定等待，接著執行 `SettleJS` | [頁面載入](/zh/core-concepts-snapshots) |
| 9 | 同意橫幅 | 最多兩輪同意橫幅處理 | [同意橫幅](/zh/core-concepts-consent) |
| 10 | 捲動 | 最多 `ScrollCount` 輪，每輪一份快照 | [頁面載入](/zh/core-concepts-snapshots) |
| 11 | 轉換 | 快照轉成 HTML、Markdown 或 JSON 樹 | [輸出格式](/zh/configuration-output-formats) |

## Context 與逾時

傳給 `Fetch` 的 `ctx` 會傳到流程中每個等待點：semaphore、settle、consent 執行與捲動延遲。`timeout` 大於 0 時每次嘗試的頁面載入各包一層子 context，因此有頭重試會重新取得一份完整的逾時。`SameSession` 的 Cookie 擷取與瀏覽器啟動發生在子 context 建立之前，只受 `ctx` 限制。settle 與 consent 另以 `IdleWait` 限制每次等待。

## 一次呼叫不做的事

- 不保留分頁：每次呼叫建立頁面並在回傳前關閉
- 除了 `StealthJS` 與 `SettleJS` 之外不點擊、不輸入、不執行自訂腳本；互動式分頁 API 已於 v0.3.2 移除，見 [已移除 API](/zh/removed-api)
