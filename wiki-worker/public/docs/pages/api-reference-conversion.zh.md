# 內容轉換

把 HTML 轉成 Markdown 或節點樹、合併快照與去重的六個匯出函式。不需要瀏覽器，手上已有 HTML 即可使用。

## Markdown

### HTMLToMarkdown

```go
func HTMLToMarkdown(content, baseURL string, keepLinks bool) (string, error)
```

HTML 轉 Markdown。`baseURL` 用於把相對連結與圖片來源解析為絕對 URL。

| 元素 | 輸出 |
|---|---|
| `h1`–`h6` | `#` 到 `######` 標題 |
| `p`、`br` | 段落與換行 |
| `li` | `- ` 項目；有序列表不編號 |
| `strong` / `b`、`em` / `i` | `**粗體**`、`*斜體*` |
| `code`、`pre` | 行內反引號、不帶語言的程式碼區塊 |
| `blockquote` | `> ` 前綴 |
| `time` | 有屬性時輸出 ` [datetime] `，否則輸出內文 |
| `a` | `keepLinks` 時為 `[text](url)`，否則只留文字 |
| `img` | `keepLinks` 時為 `![alt](url)`，優先取 `data-src`；`data:` URI 丟棄 |
| `nav`、`header`、`footer`、`aside` | 僅 `keepLinks` 時保留 |
| `script`、`style`、`noscript`、`iframe`、`form`、`button`、`input`、`select`、`textarea`、`svg`、`canvas`、`video`、`audio` | 一律丟棄 |
| 表格與其他區塊 | 文字前後換行；不產生 Markdown 表格語法 |

最後每行去除前後空白、連續空行收斂為一行，只由 `-`、`#`、`|`、`_`、`*` 組成的行會被移除。

### DedupMarkdownParagraphs

```go
func DedupMarkdownParagraphs(md string) string
```

以空行切段，去除前後空白後與先前段落相同者丟棄，整理空行並修剪結果。

## 節點樹

### HTMLToNode

```go
func HTMLToNode(content, baseURL string, keepLinks bool) ([]*Node, error)
```

以相同的元素規則把 HTML 轉成 `Node` 樹，接著套用 `DedupTree`，並把恰有一個子節點的節點扁平化成該子節點（`link` 節點除外）。這些處理的結果：

- 只含文字的段落會變成 `text` 節點，外層型別消失
- `link` 即使只包一個子節點也保留 `link` 型別與 `Href`，只含文字的連結不會在扁平化時遺失
- `image` 節點與沒有子節點的 `time` 節點沒有 `Text` 也沒有子節點，會被 `DedupTree` 移除；即使 `keepLinks` 為真，回傳的樹也不含圖片

### DedupTree

```go
func DedupTree(nodes []*Node) []*Node
```

以串接文字的 FNV-1a 雜湊移除重複的 `paragraph`、`heading`、`list_item`、`blockquote`、`code_block` 節點，保留整棵樹中第一次出現者。也會移除除 `linebreak` 以外所有無文字且無子節點的節點。

## HTML 前處理

### Merge

```go
func Merge(snapshots []string) (string, error)
```

把後續每份快照的 `<body>` 子節點依序接到第一份之後。空切片回傳 `no snapshots`；單一元素原樣回傳。第一份沒有 `<body>` 時直接回傳它，後續解析失敗的快照略過。

### InlineTimeElements

```go
func InlineTimeElements(htmlSrc string) (string, error)
```

把每個 `<time>` 換成純文字：

| `<time>` 具有 | 替換為 |
|---|---|
| `datetime` 與文字 | ` [datetime] text ` |
| 只有 `datetime` | ` [datetime] ` |
| 只有文字 | ` text ` |
| 兩者皆無 | 移除 |

`<time>` 是連結的唯一子節點時連同連結一起替換。輸入不含 `<time>` 時不解析、原樣回傳。
