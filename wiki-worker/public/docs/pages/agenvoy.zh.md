# Agenvoy 如何使用 go-browser

[Agenvoy](https://agenvoy.com/zh/) 是以 Go 打造、24 小時跑在你電腦上的本地 AI agent harness；它的 agent 讀取的每個網頁都經過 go-browser。本頁對照 Agenvoy 原始碼，列出每個 Agenvoy 工具對 go-browser 的呼叫方式。

## 工具對照

Agenvoy 依賴 `github.com/pardnchiu/go-browser v0.3.5`，其中三個[網頁工具](https://agenvoy.com/zh/built-in-tools-web)會呼叫 [`Fetch`](/zh/api-reference#fetch)，逾時一律 30 秒：

| Agenvoy 工具 | 原始碼 | 何時呼叫 `Fetch` | 傳入的 `Option` |
| --- | --- | --- | --- |
| `fetch_page` | `internal/tools/fetchPage/fetchPage.go` | 每次讀取頁面且未命中 30 分鐘快取時 | `KeepLinks`、`SameSession`、`Visible`、`Type`、`ScrollCount`、`MaxLength` |
| `download_file` | `internal/tools/fetchPage/downloadFile.go` | `render=true`：頁面渲染後寫入檔案，而非回傳給 agent | 與 `fetch_page` 相同，固定開啟 `SameSession`、先無頭、不限長度 |
| `search_web` | `internal/tools/searchWeb/fetch.go` | `cdp=true`，或 DuckDuckGo 對一般 HTTP 請求回 202 之後 | `Type` = `TypeHTML`、`MaxLength` = 200 KiB |

## fetch_page

`fetch_page` 是 agent 讀取連結的方式，每個工具參數各對應一個 [`Option`](/zh/api-reference-types#option) 欄位：

| 工具參數 | `Option` 欄位 | go-browser 中的行為 |
| --- | --- | --- |
| `output_type`（`markdown`／`html`／`json`） | `Type`（`TypeMarkdown`／`TypeHTML`／`TypeJSON`） | 見 [輸出格式](/zh/configuration-output-formats) |
| `keep_links` | `KeepLinks` | Markdown 保留同網域連結 |
| `same_session`（預設 `true`） | `SameSession` | 解密並注入使用者的 Chrome Cookie，需登入的頁面也能載入；見 [登入 Cookie](/zh/core-concepts-cookie-session) |
| `headless`（預設 `true`） | `Visible` = `!headless` | 先無頭，被擋時以有頭模式重試；見 [Fetch 路由](/zh/core-concepts-routing) |
| `scroll`（0-20） | `ScrollCount`，0 會轉成 -1（不捲動） | 載入延遲內容；見 [頁面載入](/zh/core-concepts-snapshots) |

Agenvoy 另外自行設定一個欄位：

- `MaxLength`：回傳給 agent 時設為 1 MiB 加 1 byte，藉此偵測超過 1 MiB 的 `html` 頁面並拒絕、改指向 `download_file`；存檔時不限長度

Agenvoy 對 [`Result`](/zh/api-reference-types#result) 的處理：

- `markdown`：`Title`、`Href`、`Author`、`PublishedAt`、`Excerpt` 寫成 `Content` 上方的 YAML frontmatter
- `json`：`Content` 解碼成 `Title` 與節點 `Tree`
- 回傳 [`*Error`](/zh/api-reference-types#error)、`Status` 為 400 以上，或標題命中 [`IsErrorPageTitle`](/zh/api-reference#iserrorpagetitle)時，該網址會記入 Agenvoy 的 ToriiDB 工具快取並被略過（5xx 12 小時、空白頁 72 小時、4xx 90 天），直到 agent 傳入 `force`；見 [ToriiDB 在 Agenvoy 中的用法](https://toriidb.pardn.io/zh/agenvoy)

讀取後，Agenvoy 的決策模型會依當前請求過濾頁面內容，實測效果見 [fetch_page 過濾基準測試](https://agenvoy.com/zh/providers-typesafe-fetch-page)。

## download_file

`render=true` 時，`download_file` 執行與 `fetch_page` 相同的處理流程，固定開啟 `SameSession`、先無頭、使用預設捲動次數，再把 Markdown、HTML 或純 JSON 節點樹寫入輸出檔。不帶 `render` 時是一般 HTTP 下載，不經過 go-browser。

## search_web

`search_web` 先以一般 HTTP 用戶端把查詢 POST 到 DuckDuckGo 的 HTML 端點。回應 202（機器人檢查）後，Agenvoy 在整個程序存續期間改用 go-browser：`Fetch` 以 `TypeHTML`、`MaxLength` 200 KiB 在 Chrome 中載入同一個結果頁，Agenvoy 再從回傳的 HTML 解析結果清單。agent 也能以 `cdp=true` 直接走這條路徑。

## Agenvoy 從 go-browser 得到什麼

本站描述的行為，就是 Agenvoy agent 瀏覽網頁時實際得到的行為：403、429、503 時以有頭模式重試、透過 `SameSession` 沿用 Cookie、自動關閉同意橫幅、以 readability 擷取成 Markdown，以及受並行上限控管的共用 Chrome 實例。各層如何組合見 [架構](/zh/architecture)。
