# Anti-Detection

What go-browser changes so automated Chrome looks closer to a regular browser.

## Launch flags

| Flag | Effect |
|---|---|
| `disable-blink-features=AutomationControlled` | Chrome does not advertise automation through Blink |
| `user-agent` | `Option.UserAgent`, or `DefaultUserAgent` (Chrome 124 on Linux x86_64) |
| `window-size=1280,960` | A desktop-sized window, matching the default viewport |
| `window-position=-32000,-32000` | Headed windows only: placed off screen |

## Injected stealth script

`Option.StealthJS` is registered with `EvalOnNewDocument`, so it runs before any page script in every document the tab loads. The embedded default does four things:

| Property | Value presented to the page |
|---|---|
| `navigator.webdriver` | `undefined` |
| `navigator.plugins` | Three entries: Chrome PDF Plugin, Chrome PDF Viewer, Native Client |
| `navigator.languages` | `["en-US", "en"]` |
| `window.chrome` | `{ runtime: {} }` |

Setting `StealthJS` replaces the default entirely; it does not append to it.

## Behavior

### Human-like scrolling

Scroll passes wait 150ms plus up to 300ms of random jitter and scroll smoothly over 300ms instead of jumping, see [Page Loading](/core-concepts-snapshots).

### Headed fallback

If a challenge requires user interaction, see [Manual verification](/core-concepts-routing#manual-verification).
