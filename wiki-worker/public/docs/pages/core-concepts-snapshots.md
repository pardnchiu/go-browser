# Page Loading

When a page counts as loaded, how the scroll loop captures dynamically loaded content, and when it stops.

## Load event

After navigation the page waits for the `load` event for at most 10 seconds. Pages whose subresources never finish (stalled trackers, long-polling scripts) would otherwise block the fetch indefinitely:

| Outcome after 10 seconds | Behavior |
|---|---|
| `load` fired | Continue to settling |
| The `Fetch` context ended first | The context error is returned |
| `load` still pending, `document.readyState` is `interactive` or `complete` | Continue to settling; the DOM is already parsed |
| `load` still pending, `document.readyState` is `loading` | Keep waiting for `load` with no 10-second cap, bounded only by the attempt context (`ctx` plus `timeout`) |

A failure here ends the attempt with `page.WaitLoad: ...`.

## Settling

Before anything is extracted, the page settles in two steps:

1. `WaitDOMStable` with a window of `min(400ms, IdleWait)` and a 0.01 change threshold, capped at `IdleWait` overall
2. `SettleJS`, capped at `IdleWait`; the default `listener.js` resolves after one `requestIdleCallback` (250ms timeout) or after 1 second, whichever comes first

Only step 1 runs again: after each consent pass that dismisses something, and after each scroll. `SettleJS` runs once per load. Timeouts here are not errors: extraction continues with whatever the DOM holds.

## Scroll loop

After consent handling, the first snapshot is the full page HTML. Then, up to `ScrollCount` times:

| Step | Detail |
|---|---|
| Check | Stop when the document is no longer taller than the viewport plus 4px |
| Wait | 150ms plus 0–300ms of random jitter; stops if the context ends |
| Scroll | Smooth scroll to the bottom over 300ms with `requestAnimationFrame` |
| Settle | DOM stability wait only (step 1); `SettleJS` is not re-run |
| Snapshot | Stop when the new HTML is byte-identical to the previous snapshot |

`ScrollCount` defaults to 3; a negative value disables scrolling entirely, leaving only the first snapshot.

## Why multiple snapshots

Infinite-scroll feeds and lazy-loaded articles replace or append content as you scroll. Each snapshot captures one state, and the output stage combines them: `TypeHTML` merges their `<body>` children, while `TypeMarkdown` and `TypeJSON` run readability on each snapshot and join the article contents. Repeated paragraphs introduced by overlapping snapshots are removed afterwards. See [Output Formats](/configuration-output-formats).

## Viewport

The page uses `Option.Viewport`, 1280 × 960 by default, with `DeviceScaleFactor` 0 treated as 1. A taller viewport loads more content per snapshot.
