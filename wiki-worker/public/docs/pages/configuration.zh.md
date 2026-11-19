# Option 欄位

`browser.Option` 的每個欄位、維持零值時採用的值，以及欄位之間的交互作用。

## 欄位

| 欄位 | 型別 | 未設定時 | 說明 |
|---|---|---|---|
| `IdleWait` | `time.Duration` | `2s` | 每次 settle、`SettleJS` 與 consent 執行的上限；DOM 穩定視窗為 `min(400ms, IdleWait)` |
| `MaxLength` | `int` | `1048576`（`1 << 20`） | Markdown 輸出截斷到此 **byte** 數，可能切開多位元組字元；負值停用截斷；`TypeHTML` 與 `TypeJSON` 不套用 |
| `UserAgent` | `string` | `DefaultUserAgent` | 同時是瀏覽器快取鍵的一部分 |
| `KeepLinks` | `bool` | `false` | 轉換輸出保留連結、圖片與 `nav` / `header` / `footer` / `aside` |
| `StealthJS` | `string` | 內嵌 `stealth.js` | 以 `EvalOnNewDocument` 注入；會取代預設值，空字串恢復預設，見 [反偵測](/zh/core-concepts-anti-detection) |
| `SettleJS` | `string` | 內嵌 `listener.js` | DOM 穩定後執行一次；空字串恢復預設 |
| `Viewport` | `*Viewport` | `1280 × 960` | `DeviceScaleFactor` 為 0 時視為 1 |
| `SameSession` | `bool` | `false` | 以帶本機 Chrome Cookie 的暫存設定檔啟動，見 [Cookie 工作階段](/zh/core-concepts-cookie-session) |
| `Headless` | `bool` | `false` | 強制 headless，不做有頭重試 |
| `Visible` | `bool` | `false` | 有 GUI 時直接有頭、不重試；沒有 GUI 時退回 headless 優先。`Headless` 為 true 時忽略，見 [Fetch 路由](/zh/core-concepts-routing) |
| `ChallengeWait` | `time.Duration` | `0` | 偵測到真人驗證時，有頭模式可等待使用者完成的最長時間；小於或等於零時回傳 `ErrChallengeRequired`。見 [真人驗證](/zh/core-concepts-routing#真人驗證) |
| `Profile` | `string` | 最近使用的設定檔 | Chrome 設定檔目錄名，僅 `SameSession` 使用；空值時讀 Chrome `Local State` 的 `profile.last_used`，讀不到時為 `Default` |
| `Type` | `int` | `TypeMarkdown` | `TypeMarkdown`、`TypeHTML` 或 `TypeJSON`，見 [輸出格式](/zh/configuration-output-formats) |
| `ScrollCount` | `int` | `3` | 傳負值表示不捲動 |

`Fetch` 會先複製 `Option` 再補預設值，傳入的值不會被修改，可在多次呼叫間重用。

## Viewport

```go
type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}
```

## 欄位交互作用

| 組合 | 效果 |
|---|---|
| `Headless: true` + session 網域 | 以 headless 為準，忽略 session 網域清單 |
| `Headless: true` + `Visible: true` | 以 headless 為準，忽略 `Visible` |
| `Visible: true` 但沒有 GUI | 等同未設定：headless 優先，無法有頭重試 |
| `Headless: true` + `SameSession: true` | 在暫存 Cookie 設定檔上跑 headless 瀏覽器 |
| `SameSession: false` + `Profile` | `Profile` 不生效 |
| `ScrollCount: 0` | 視為未設定，捲動 3 次；要停用請傳 `-1` |
| `IdleWait` 小於 400ms | DOM 穩定視窗縮短為 `IdleWait` |

## 範例

```go
result, err := browser.Fetch(ctx, href, 45*time.Second, &browser.Option{
	Type:        browser.TypeMarkdown,
	KeepLinks:   true,
	ScrollCount: 5,
	IdleWait:    3 * time.Second,
	Viewport:    &browser.Viewport{Width: 1440, Height: 2000},
})
```
