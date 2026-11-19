# How Agenvoy Uses go-browser

[Agenvoy](https://agenvoy.com/) is a local AI agent harness in Go that runs 24/7 on your own machine; every web page its agent reads goes through go-browser. This page maps each Agenvoy tool to the go-browser calls it makes, based on the Agenvoy source.

## Tool Map

Agenvoy depends on `github.com/pardnchiu/go-browser v0.3.5`. Three of its [web tools](https://agenvoy.com/built-in-tools-web) call [`Fetch`](/api-reference#fetch), always with a 30-second timeout:

| Agenvoy tool | Source | When it calls `Fetch` | `Option` it passes |
| --- | --- | --- | --- |
| `fetch_page` | `internal/tools/fetchPage/fetchPage.go` | Every page read that is not served from the 30-minute cache | `KeepLinks`, `SameSession`, `Visible`, `Type`, `ScrollCount`, `MaxLength` |
| `download_file` | `internal/tools/fetchPage/downloadFile.go` | `render=true`: the page is rendered and written to disk instead of returned | Same as `fetch_page` with `SameSession` on, headless first, no length cap |
| `search_web` | `internal/tools/searchWeb/fetch.go` | `cdp=true`, or after DuckDuckGo answers the plain HTTP request with 202 | `Type` = `TypeHTML`, `MaxLength` = 200 KiB |

## fetch_page

`fetch_page` is how the agent reads a link. Each tool parameter maps onto one [`Option`](/api-reference-types#option) field:

| Tool parameter | `Option` field | Behavior in go-browser |
| --- | --- | --- |
| `output_type` (`markdown` / `html` / `json`) | `Type` (`TypeMarkdown` / `TypeHTML` / `TypeJSON`) | See [Output Formats](/configuration-output-formats) |
| `keep_links` | `KeepLinks` | Same-domain links stay in the Markdown |
| `same_session` (default `true`) | `SameSession` | The user's Chrome cookies are decrypted and injected, so login-only pages load; see [Cookie Sessions](/core-concepts-cookie-session) |
| `headless` (default `true`) | `Visible` = `!headless` | Headless first, headed retry when blocked; see [Fetch Routing](/core-concepts-routing) |
| `scroll` (0-20) | `ScrollCount`, with 0 sent as -1 (no scrolling) | Lazy content loading; see [Page Loading](/core-concepts-snapshots) |

Agenvoy also sets one field itself:

- `MaxLength`: 1 MiB plus one byte when the page is returned to the agent, so an `html` page over 1 MiB can be detected and refused with a pointer to `download_file`; unlimited when the page is saved

What Agenvoy does with the [`Result`](/api-reference-types#result):

- `markdown`: `Title`, `Href`, `Author`, `PublishedAt` and `Excerpt` become YAML frontmatter above `Content`
- `json`: `Content` is decoded into its `Title` and node `Tree`
- A returned [`*Error`](/api-reference-types#error), a `Status` of 400 or above, or a title that [`IsErrorPageTitle`](/api-reference#iserrorpagetitle) matches marks the URL as failed in Agenvoy's ToriiDB tool cache, so the same URL is skipped (5xx for 12 hours, empty pages for 72 hours, 4xx for 90 days) until the agent passes `force`; see [ToriiDB in Agenvoy](https://toriidb.pardn.io/agenvoy)

After the fetch, Agenvoy's decision model filters the page against the current request; the measured effect is on the [fetch_page Filtering Benchmark](https://agenvoy.com/providers-typesafe-fetch-page).

## download_file

With `render=true`, `download_file` runs the same handler as `fetch_page` with `SameSession` on, headless first, and the default scroll count, then writes the Markdown, HTML, or bare JSON node tree to the output file. Without `render`, it is a plain HTTP download and go-browser is not used.

## search_web

`search_web` first posts the query to DuckDuckGo's HTML endpoint with a plain HTTP client. When that returns 202 (the bot check), Agenvoy switches to go-browser for the rest of the process: `Fetch` loads the same results page in Chrome with `TypeHTML` and a 200 KiB `MaxLength`, and Agenvoy parses the result list from the returned HTML. The agent can also force this path with `cdp=true`.

## What Agenvoy Gets from go-browser

Behavior described on this site is what an Agenvoy agent gets when it browses: headed retry on 403, 429 or 503, cookie reuse through `SameSession`, consent banner dismissal, readability extraction into Markdown, and one shared Chrome instance gated by the concurrency limit. See [Architecture](/architecture) for how these layers fit together.
