# 頁面載入

頁面何時算載入完成、捲動迴圈如何擷取動態載入的內容，以及何時停止。

## load 事件

導航後頁面最多等待 `load` 事件 10 秒；子資源一直載不完（卡住的追蹤器、long-polling 腳本）的頁面否則會讓擷取無限期停住：

| 10 秒後的狀態 | 行為 |
|---|---|
| `load` 已觸發 | 進入等待穩定 |
| `Fetch` 的 context 先結束 | 回傳 context 錯誤 |
| `load` 未觸發、`document.readyState` 為 `interactive` 或 `complete` | 進入等待穩定；DOM 已解析完成 |
| `load` 未觸發、`document.readyState` 仍為 `loading` | 繼續等待 `load`，不再有 10 秒上限，只受該次嘗試的 context（`ctx` 加上 `timeout`）限制 |

此處失敗會以 `page.WaitLoad: ...` 結束該次嘗試。

## 等待穩定

萃取前頁面分兩段等待穩定：

1. `WaitDOMStable`，視窗為 `min(400ms, IdleWait)`、變動比例門檻 0.01，整體上限 `IdleWait`
2. `SettleJS`，上限 `IdleWait`；預設的 `listener.js` 在一次 `requestIdleCallback`（逾時 250ms）或 1 秒後結束，以先到者為準

之後只重跑第 1 段：每輪有關閉橫幅的同意處理後、每次捲動後各一次；`SettleJS` 每次載入只執行一次。這裡的逾時不算錯誤：萃取以當下 DOM 的內容繼續。

## 捲動迴圈

同意橫幅處理完後，第一份快照是整頁 HTML。接著最多執行 `ScrollCount` 次：

| 步驟 | 細節 |
|---|---|
| 檢查 | 文件高度不再超過視窗高度加 4px 時停止 |
| 等待 | 150ms 加 0–300ms 隨機抖動；context 結束時停止 |
| 捲動 | 以 `requestAnimationFrame` 在 300ms 內平滑捲到底 |
| 等待穩定 | 只等 DOM 穩定（第 1 段），不重跑 `SettleJS` |
| 快照 | 新 HTML 與前一份完全相同時停止 |

`ScrollCount` 預設 3；傳負值完全不捲動，只留第一份快照。

## 為何要多份快照

無限捲動的動態牆與延遲載入的文章會在捲動時替換或追加內容。每份快照記錄一個狀態，輸出階段再合併：`TypeHTML` 合併各份的 `<body>` 子節點，`TypeMarkdown` 與 `TypeJSON` 對每份快照各跑一次 readability 再串接內容。快照重疊造成的重複段落之後會被移除。見 [輸出格式](/zh/configuration-output-formats)。

## Viewport

頁面使用 `Option.Viewport`，預設 1280 × 960，`DeviceScaleFactor` 為 0 時視為 1。viewport 越高，每份快照載入的內容越多。
