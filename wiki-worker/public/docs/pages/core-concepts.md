# Core Concepts

The ordered steps one `Fetch` call runs, from URL validation to the returned `Result`, with a link to the page that explains each step.

## The fetch pipeline

| # | Step | What happens | Details |
|---|---|---|---|
| 1 | Validate | `Option` defaults are filled in; the URL must have a scheme and a dotted hostname | [Option Fields](/configuration) |
| 2 | Route | Headless or headed is chosen from `Option.Headless`, `Option.Visible`, the session domain list and GUI availability | [Fetch Routing](/core-concepts-routing) |
| 3 | Get a browser | A cached instance keyed by `(headless, userAgent)`, or a throwaway one with cookies when `SameSession` is set | [Browser Cache](/core-concepts-browser-cache), [Cookie Sessions](/core-concepts-cookie-session) |
| 4 | Acquire a slot | The page waits on the global concurrency semaphore | [Browser Cache](/core-concepts-browser-cache) |
| 5 | Open the page | A blank tab gets the viewport and the stealth script, then navigates and waits for `load` (10-second cap with a `readyState` fallback) | [Anti-Detection](/core-concepts-anti-detection), [Page Loading](/core-concepts-snapshots#load-event) |
| 6 | Check the URL | A `404` or `403` path segment or query value in the post-redirect URL ends the call with `*Error` | [Troubleshooting](/getting-started-troubleshooting) |
| 7 | Short-circuit data | JSON and XML documents are returned as raw text with `ContentType` set | [Output Formats](/configuration-output-formats) |
| 8 | Settle | DOM stability wait, then `SettleJS` | [Page Loading](/core-concepts-snapshots) |
| 9 | Consent | Up to two passes of consent banner dismissal | [Consent Banners](/core-concepts-consent) |
| 10 | Scroll | Up to `ScrollCount` passes, one snapshot per pass | [Page Loading](/core-concepts-snapshots) |
| 11 | Convert | Snapshots become HTML, Markdown or a JSON tree | [Output Formats](/configuration-output-formats) |

## Context and timeout

The `ctx` passed to `Fetch` reaches every wait in the pipeline: the semaphore, settling, consent evaluation and scroll delays. A `timeout` above 0 wraps it in a child context for each attempt's page load, so a headed retry gets a fresh timeout of its own. `SameSession` cookie extraction and browser launch run before that child context exists and are bounded only by `ctx`. Settle and consent steps additionally cap each wait at `IdleWait`.

## What one call does not do

- It does not keep the tab open: every call creates a page and closes it before returning
- It does not click, type or run custom scripts beyond `StealthJS` and `SettleJS`; the interactive tab API was removed in v0.3.2, see [Removed API](/removed-api)
