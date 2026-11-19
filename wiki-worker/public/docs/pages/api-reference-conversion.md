# Content Conversion

The six exported functions that turn HTML into Markdown or a node tree, merge snapshots and remove duplicates. They need no browser and work on HTML you already have.

## Markdown

### HTMLToMarkdown

```go
func HTMLToMarkdown(content, baseURL string, keepLinks bool) (string, error)
```

Converts HTML to Markdown. `baseURL` resolves relative links and image sources to absolute URLs.

| Element | Output |
|---|---|
| `h1`–`h6` | `#` to `######` headings |
| `p`, `br` | Paragraph breaks and line breaks |
| `li` | `- ` items; ordered lists are not numbered |
| `strong` / `b`, `em` / `i` | `**bold**`, `*italic*` |
| `code`, `pre` | Inline backticks, fenced block without a language |
| `blockquote` | `> ` prefix |
| `time` | ` [datetime] ` when the attribute exists, otherwise the inner text |
| `a` | `[text](url)` with `keepLinks`, otherwise the text only |
| `img` | `![alt](url)` with `keepLinks`, preferring `data-src`; `data:` URIs are dropped |
| `nav`, `header`, `footer`, `aside` | Kept only with `keepLinks` |
| `script`, `style`, `noscript`, `iframe`, `form`, `button`, `input`, `select`, `textarea`, `svg`, `canvas`, `video`, `audio` | Always dropped |
| tables and other blocks | Line breaks around the text; no Markdown table syntax |

Afterwards every line is trimmed, runs of blank lines collapse to one, and lines made only of `-`, `#`, `|`, `_` or `*` are removed.

### DedupMarkdownParagraphs

```go
func DedupMarkdownParagraphs(md string) string
```

Splits on blank lines, drops paragraphs identical to an earlier one after trimming, normalizes blank lines and trims the result.

## Node tree

### HTMLToNode

```go
func HTMLToNode(content, baseURL string, keepLinks bool) ([]*Node, error)
```

Converts HTML to a `Node` tree with the same element rules, then applies `DedupTree` and flattens any node with exactly one child into that child, except `link` nodes. Consequences of those passes:

- A paragraph containing only text becomes a `text` node; the wrapper type is lost
- A `link` keeps its `link` type and `Href` even when it wraps a single child, so single-text links survive flattening
- `image` nodes and `time` nodes without children carry no `Text` and no children, so `DedupTree` removes them; the returned tree contains no images even with `keepLinks`

### DedupTree

```go
func DedupTree(nodes []*Node) []*Node
```

Removes repeated `paragraph`, `heading`, `list_item`, `blockquote` and `code_block` nodes by FNV-1a hash of their concatenated text, keeping the first occurrence anywhere in the tree. It also drops every node other than `linebreak` that has no text and no children.

## HTML preprocessing

### Merge

```go
func Merge(snapshots []string) (string, error)
```

Appends the `<body>` children of each later snapshot to the first, in order. An empty slice returns `no snapshots`; a single element is returned unchanged. When the first snapshot has no `<body>` it is returned as-is, and later snapshots that fail to parse are skipped.

### InlineTimeElements

```go
func InlineTimeElements(htmlSrc string) (string, error)
```

Replaces each `<time>` with plain text:

| `<time>` has | Replacement |
|---|---|
| `datetime` and text | ` [datetime] text ` |
| `datetime` only | ` [datetime] ` |
| text only | ` text ` |
| neither | removed |

When the `<time>` is the only child of a link, the link is replaced along with it. Input without `<time>` is returned unchanged without parsing.
