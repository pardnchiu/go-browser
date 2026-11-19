# Option Fields

Every field of `browser.Option`, the value used when it is left at its zero value, and how fields interact.

## Fields

| Field | Type | When unset | Notes |
|---|---|---|---|
| `IdleWait` | `time.Duration` | `2s` | Cap for each settle, `SettleJS` and consent evaluation; the DOM stability window is `min(400ms, IdleWait)` |
| `MaxLength` | `int` | `1048576` (`1 << 20`) | Markdown output is cut to this many **bytes**, which can split a multi-byte character; a negative value disables truncation; ignored by `TypeHTML` and `TypeJSON` |
| `UserAgent` | `string` | `DefaultUserAgent` | Also part of the browser cache key |
| `KeepLinks` | `bool` | `false` | Keep links, images and `nav` / `header` / `footer` / `aside` in converted output |
| `StealthJS` | `string` | embedded `stealth.js` | Injected with `EvalOnNewDocument`; replaces the default, an empty string restores it, see [Anti-Detection](/core-concepts-anti-detection) |
| `SettleJS` | `string` | embedded `listener.js` | Evaluated once after the DOM settles; an empty string restores the default |
| `Viewport` | `*Viewport` | `1280 × 960` | `DeviceScaleFactor` 0 is treated as 1 |
| `SameSession` | `bool` | `false` | Temporary profile with local Chrome cookies, see [Cookie Sessions](/core-concepts-cookie-session) |
| `Headless` | `bool` | `false` | Forces headless with no headed retry |
| `Visible` | `bool` | `false` | Goes headed directly with no retry when a GUI is available; without a GUI it falls back to headless first. Ignored when `Headless` is true, see [Fetch Routing](/core-concepts-routing) |
| `ChallengeWait` | `time.Duration` | `0` | Maximum time for the user to complete a detected challenge in headed mode. Zero or negative returns `ErrChallengeRequired`. See [Manual verification](/core-concepts-routing#manual-verification) |
| `Profile` | `string` | last used profile | Chrome profile directory name, used only with `SameSession`; empty reads `profile.last_used` from Chrome's `Local State`, falling back to `Default` |
| `Type` | `int` | `TypeMarkdown` | `TypeMarkdown`, `TypeHTML` or `TypeJSON`, see [Output Formats](/configuration-output-formats) |
| `ScrollCount` | `int` | `3` | A negative value means no scrolling |

`Fetch` copies the `Option` before filling defaults, so the value you pass is never modified and can be reused across calls.

## Viewport

```go
type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}
```

## Field interactions

| Combination | Effect |
|---|---|
| `Headless: true` + session domain | Headless wins; the session domain list is ignored |
| `Headless: true` + `Visible: true` | Headless wins; `Visible` is ignored |
| `Visible: true` without a GUI | Same as unset: headless first, headed retry not possible |
| `Headless: true` + `SameSession: true` | A headless browser on the temporary cookie profile |
| `SameSession: false` + `Profile` | `Profile` has no effect |
| `ScrollCount: 0` | Treated as unset, so 3 passes; use `-1` to disable |
| `IdleWait` below 400ms | The DOM stability window shrinks to `IdleWait` |

## Example

```go
result, err := browser.Fetch(ctx, href, 45*time.Second, &browser.Option{
	Type:        browser.TypeMarkdown,
	KeepLinks:   true,
	ScrollCount: 5,
	IdleWait:    3 * time.Second,
	Viewport:    &browser.Viewport{Width: 1440, Height: 2000},
})
```
