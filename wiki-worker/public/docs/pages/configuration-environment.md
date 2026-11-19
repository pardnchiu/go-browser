# Environment

The environment variables, Chrome binary locations and profile paths go-browser reads from the host.

## Environment variables

| Variable | Effect |
|---|---|
| `DISPLAY` | On Linux, any non-empty value means a GUI is available, enabling the headed retry, `Option.Visible` and headed session domains |
| `WAYLAND_DISPLAY` | Same |

macOS is always treated as having a GUI. A Linux host with neither variable set stays headless and never fails by trying to open a headed browser. go-browser reads no other variable itself; `HOME` and `TMPDIR` are used implicitly for the profile root and the temporary profile, and go-rod also honours its own `-rod` flag.

## Chrome locations

### Binary lookup

| Platform | Order |
|---|---|
| macOS | `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome` → `/Applications/Chromium.app/Contents/MacOS/Chromium` |
| Linux | `google-chrome` → `google-chrome-stable` → `chromium` → `chromium-browser` on `PATH` |

When none is found, go-rod decides where the browser comes from, downloading one if needed.

### Profile root

| Platform | Path |
|---|---|
| macOS | `~/Library/Application Support/Google/Chrome` |
| Linux | `~/.config/google-chrome` |

`SameSession` uses the subdirectory named by `Option.Profile`. On other platforms, or when the directory is missing, the call continues without cookies.

## External commands

Only `SameSession` runs external commands:

| Command | Platform | Purpose |
|---|---|---|
| `security` | macOS | Read the Chrome Safe Storage password from the keychain |
| `secret-tool` | Linux | Read the password from the gnome-keyring Secret Service |
| `kwallet-query` | Linux | Read the password from KWallet (KDE) |

## Containers

In Docker or CI, install Chrome or Chromium on `PATH`, leave `DISPLAY` unset to stay headless, and expect blocked sites to return errors instead of a headed retry. `no-sandbox` and `disable-dev-shm-usage` are always set, so a small `/dev/shm` is not a problem.
