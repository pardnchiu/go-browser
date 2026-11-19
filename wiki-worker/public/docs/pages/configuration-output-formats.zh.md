# 輸出格式

各種 `Option.Type` 下 `Result` 的內容，以及 JSON 與 XML 文件的獨立路徑。

## 選擇格式

```go
opt := &browser.Option{Type: browser.TypeMarkdown} // 預設
opt = &browser.Option{Type: browser.TypeHTML}      // 合併後的 HTML
opt = &browser.Option{Type: browser.TypeJSON}      // 含節點樹的 Result，序列化進 Content
```

| 常數 | 值 | `Result.Content` | 其他填入欄位 |
|---|---|---|---|
| `TypeMarkdown` | 0 | 每份快照的 readability 內容轉成 Markdown 並段落去重，截斷至 `MaxLength` bytes | `Href`、`FinalURL`、`Title`、`Author`、`PublishedAt`、`Excerpt`、`Status`、`Consent` |
| `TypeHTML` | 1 | 全部快照以 `Merge` 合併後再 `InlineTimeElements` | `Href`、`FinalURL`、`Status`、`Consent` |
| `TypeJSON` | 2 | 完整 `Result` 的 JSON 字串，內含 `Tree`，其自身 `Content` 為空 | `Href`、`FinalURL`、`Status`、`Consent` |

## 各格式細節

### TypeMarkdown

每份快照先經 `InlineTimeElements` 再跑 readability。標題、作者、摘要與發布時間取自**第一份**解析成功的快照；readability 找得到日期時 `PublishedAt` 為 RFC3339。所有快照都能解析但內容皆為空時，改轉換合併後的原始 HTML。最終 Markdown 為空時回傳 `http 204`。轉換規則見 [內容轉換](/zh/api-reference-conversion)。

### TypeHTML

不跑 readability，因此 `Title` 等中繼資料為空。輸出保留完整頁面結構（含導覽與 script），後續快照的 `<body>` 子節點依序接在第一份之後。

### TypeJSON

與 Markdown 相同的 readability 內容以 `HTMLToNode` 轉成 `Node` 樹。回傳的 `Result` 只在 JSON 字串內帶有中繼資料：

```go
var full browser.Result
if err := json.Unmarshal([]byte(result.Content), &full); err != nil {
	return err
}
fmt.Println(full.Title, len(full.Tree))
```

由於仍會先跑 Markdown 步驟，Markdown 為空時這裡同樣回傳 `http 204`。節點結構見 [型別](/zh/api-reference-types)。

## JSON 與 XML 文件

`document.contentType` 含 `json` 或 `xml` 時，等待穩定、同意橫幅、捲動與轉換全部略過：

| 內容型別 | `Result.Content` |
|---|---|
| JSON | `document.body.innerText`，即原始回應文字 |
| XML | Chrome XML 檢視器中的原始碼，或序列化後的文件 |

會填入 `ContentType`，並忽略 `Type`。取出的文字為空時改走一般 HTML 路徑。
