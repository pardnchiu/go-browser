# Getting Started

From `go get` to a first Markdown result: prerequisites, install, one `Fetch` call, and shutting browsers down.

## Prerequisites

| Item | Requirement | Why |
|---|---|---|
| Go | 1.25.1 or newer | The version declared in `go.mod` |
| OS | macOS or Linux | Chrome path lookup and cookie decryption are implemented for these two platforms; elsewhere `Fetch` still runs headless with go-rod's own browser lookup, without `SameSession` or headed retry |
| Chrome / Chromium | Installed | macOS looks for `/Applications/Google Chrome.app` or `Chromium.app`; Linux tries `google-chrome`, `google-chrome-stable`, `chromium`, `chromium-browser` in order |
| `security` (macOS) | Built in | Retrieves the Chrome Safe Storage password, only for `SameSession` |
| `secret-tool` and/or `kwallet-query` (Linux) | `libsecret-tools` / `kwallet` | Retrieves the Chrome Safe Storage password from gnome-keyring or KWallet, only for `SameSession` |

When no Chrome binary is found, go-rod falls back to a browser it downloads itself. `SameSession` still needs a local Chrome profile, because that is where the cookies come from.

## Install

```bash
go get github.com/pardnchiu/go-browser/core
```

The package lives in the `core` subdirectory and its package name is `browser`:

```go
import browser "github.com/pardnchiu/go-browser/core"
```

## First call

```go
package main

import (
	"context"
	"fmt"
	"time"

	browser "github.com/pardnchiu/go-browser/core"
)

func main() {
	defer browser.Close()

	result, err := browser.Fetch(context.Background(), "https://example.com", 30*time.Second, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Title)
	fmt.Println(result.Content)
}
```

Passing `nil` for `opt` takes every default: headless first, up to three scroll passes, Markdown output, a two second idle wait. The URL needs a scheme and a dotted hostname, so `http://localhost:8080` is rejected with `invalid url`.

### Call Close before exit

Browser instances are cached and reused, and a background evictor reclaims them after five idle minutes. Calling `browser.Close()` before exit shuts all cached instances down immediately; skipping it leaves Chrome processes running until the evictor or the process exit cleans them up.

## Next

- [Troubleshooting](/getting-started-troubleshooting): what each error means
- [Output Formats](/configuration-output-formats): Markdown, HTML or a JSON tree
- [Cookie Sessions](/core-concepts-cookie-session): reading pages behind a login
- [Option Fields](/configuration): every `Option` field and its default
