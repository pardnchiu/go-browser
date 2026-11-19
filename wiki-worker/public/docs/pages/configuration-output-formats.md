# Output Formats

What `Result` contains for each `Option.Type`, and the separate path for JSON and XML documents.

## Choosing a format

```go
opt := &browser.Option{Type: browser.TypeMarkdown} // default
opt = &browser.Option{Type: browser.TypeHTML}      // merged HTML
opt = &browser.Option{Type: browser.TypeJSON}      // Result with a node tree, serialized into Content
```

| Constant | Value | `Result.Content` | Other fields set |
|---|---|---|---|
| `TypeMarkdown` | 0 | Readability content of every snapshot, converted to Markdown and paragraph-deduplicated, cut to `MaxLength` bytes | `Href`, `FinalURL`, `Title`, `Author`, `PublishedAt`, `Excerpt`, `Status`, `Consent` |
| `TypeHTML` | 1 | All snapshots merged with `Merge`, then `InlineTimeElements` | `Href`, `FinalURL`, `Status`, `Consent` |
| `TypeJSON` | 2 | A JSON string of the full `Result`, with `Tree` set and its own `Content` empty | `Href`, `FinalURL`, `Status`, `Consent` |

## Format details

### TypeMarkdown

Readability runs on each snapshot after `InlineTimeElements`. Title, author, excerpt and publish time come from the **first** snapshot that parses; `PublishedAt` is RFC3339 when readability finds a date. When every snapshot parses but yields empty content, the merged raw HTML is converted instead. An empty final Markdown returns `http 204`. Conversion rules are on [Content Conversion](/api-reference-conversion).

### TypeHTML

No readability runs, so `Title` and the other metadata stay empty. The output keeps the full page structure, including navigation and scripts, with every later snapshot's `<body>` children appended after the first.

### TypeJSON

The same readability content as Markdown is converted with `HTMLToNode` into a `Node` tree. The returned `Result` carries the metadata only inside the JSON string:

```go
var full browser.Result
if err := json.Unmarshal([]byte(result.Content), &full); err != nil {
	return err
}
fmt.Println(full.Title, len(full.Tree))
```

Because the Markdown step still runs first, an empty Markdown result returns `http 204` here too. The node schema is on [Types](/api-reference-types).

## JSON and XML documents

When `document.contentType` contains `json` or `xml`, settling, consent, scrolling and conversion are all skipped:

| Content type | `Result.Content` |
|---|---|
| JSON | `document.body.innerText`, the raw response text |
| XML | The source from Chrome's XML viewer, or a serialized document |

`ContentType` is set, and `Type` is ignored. If the extracted text is empty, the normal HTML path runs instead.
