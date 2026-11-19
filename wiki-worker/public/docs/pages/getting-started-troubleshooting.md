# Troubleshooting

What each error returned by `Fetch` means, where it comes from, and whether a headed retry will happen.

## Status errors

These are `*browser.Error` values; read the code with `errors.As` and `Error.Status`.

| Error | Raised when | Headed retry |
|---|---|---|
| `http 403` | The post-redirect URL has a `403` path segment or query value; the first snapshot's title matches a challenge page (`just a moment`, `attention required`, `checking your browser`, `access denied`, `請稍候`); or no article was extracted and the response status is 403 | Yes, in the default branch when a GUI is available |
| `http 429` / `http 503` | No article was extracted and the response status is 429 or 503 | Yes, in the default branch when a GUI is available |
| `http 404` | The post-redirect URL has a `404` path segment or query value, or no article was extracted and the status is 404 | No |
| `http 204` | Extraction succeeded but the Markdown is empty after deduplication | No |
| other `http 4xx/5xx` | No article was extracted and the response status is 400 or higher | No |

Pre-extraction challenge detection can return `403` in headless mode for any output format. Readability also has a title check and can return `204` or no-article status errors for `TypeMarkdown` and `TypeJSON`. A `Result.Status` of `403`, `429` or `503` also triggers a headed retry when a GUI is available. Only the default headless-first route retries. See [Fetch Routing](/core-concepts-routing).

## Plain errors

If headed mode returns `ErrChallengeRequired`, set a positive `ChallengeWait` on a host with a GUI and complete verification in Chrome. If the wait expires, it returns `ErrChallenge`. Match either with `errors.Is`.

| Message | Cause |
|---|---|
| `url.Parse: ...` | The URL cannot be parsed |
| `invalid url: ...` | The URL has no scheme, or the hostname contains no `.` |
| `readability: no article extracted from N snapshots` | No snapshot parsed into an article and the status was below 400 |
| `acquireSem: context deadline exceeded` | The concurrency cap was full until `timeout` expired; see [Browser Cache](/core-concepts-browser-cache) |
| `launcher.Launch: ...` | Chrome failed to start, usually a missing binary or a sandbox restriction |
| `extract cookies: security find-generic-password: ...` | macOS keychain locked or access denied; only reached with `SameSession` |
| `[Denied] chrome safe storage: no v11 keyring password (gnome-keyring/kwallet): ...` | Neither gnome-keyring nor KWallet returned a Chrome Safe Storage password on Linux; only with `SameSession` |
| `[Denied] chrome safe storage: N keyring password(s) failed to decrypt v11 cookies` | Keyring passwords were found but none decrypts the `v11` cookies (domain-hash mismatch); only with `SameSession` |
| `[Unavailable] cookies db: ...` / `[Upstream] cookies db: ...` | The copied cookie database could not be opened or queried; only with `SameSession` |
| `inject cookies (0/N): ...` | Every cookie failed to inject, even one by one |

A missing Chrome profile wraps `browser.ErrProfileNotFound` internally, but `Fetch` does not return it: the call continues on a regular browser without cookies. The other `SameSession` errors above are returned as-is and do not fall back.

## Headless servers

A Linux host with neither `DISPLAY` nor `WAYLAND_DISPLAY` never launches a headed browser, so blocked responses come back as errors instead of being retried. See [Environment](/configuration-environment).
