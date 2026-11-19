# 瀏覽器快取

Chrome 實例如何快取、回收與共用，以及全域併發上限如何限制同時開啟的頁面。

## 快取鍵

實例以 `(headless, userAgent)` 為鍵快取；`UserAgent` 為空時先解析成 `DefaultUserAgent` 再查詢。模式或 User-Agent 不同就是不同的 Chrome 行程。這是有頭重試能生效的前提：若只有單一共用實例，重試會拿回同一個 headless 瀏覽器，等於用相同條件重跑一次。

| 參數 | 值 |
|---|---|
| 快取鍵 | `(headless, userAgent)` |
| 閒置回收門檻 | 最後使用後 5 分鐘 |
| 回收檢查週期 | 1 分鐘 |
| Evictor 啟動 | 第一次快取啟動時，每個程序一次 |

每次交出快取實例都會更新 `lastUsed`。

### 啟動參數

快取實例一律帶 `disable-blink-features=AutomationControlled`、`no-sandbox`、`disable-dev-shm-usage`、`window-size=1280,960` 與解析後的 `user-agent`。有頭實例另加 `window-position=-32000,-32000`。

## 併發上限

單一 semaphore 限制所有實例同時開啟的頁面數，不分 headless 與有頭，預設 8。

```go
browser.SetMaxConcurrency(4)
```

`SetMaxConcurrency(n)` 替換 semaphore，`n <= 0` 時忽略。已持有名額的頁面會把名額還給舊的 semaphore，新上限只套用在呼叫之後開始的頁面。等待名額的頁面在 context 結束時放棄，回傳 `acquireSem: context deadline exceeded` 或 `context canceled`。

## Close

`browser.Close()` 立即關閉全部快取實例並清空快取。可重複呼叫，之後的 `Fetch` 會重新啟動實例。evictor goroutine 不會停止。

## SameSession 不走快取

`SameSession` 的瀏覽器不進快取：每次呼叫以暫存設定檔啟動新的 Chrome，呼叫回傳時關閉並刪除目錄。它仍計入併發上限。見 [Cookie 工作階段](/zh/core-concepts-cookie-session)。
