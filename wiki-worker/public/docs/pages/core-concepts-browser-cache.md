# Browser Cache

How Chrome instances are cached, evicted and shared, and how the global concurrency cap limits open pages.

## Cache key

Instances are cached under `(headless, userAgent)`; an empty `UserAgent` resolves to `DefaultUserAgent` before the lookup. A different mode or user agent is a different Chrome process. That is what makes the headed retry real: with a single shared instance, the retry would hand back the same headless browser and re-run under identical conditions.

| Parameter | Value |
|---|---|
| Cache key | `(headless, userAgent)` |
| Idle eviction threshold | 5 minutes since last use |
| Eviction check interval | 1 minute |
| Evictor start | On the first cached launch, once per process |

`lastUsed` is refreshed every time a cached instance is handed out.

### Launch flags

Every cached instance starts with `disable-blink-features=AutomationControlled`, `no-sandbox`, `disable-dev-shm-usage`, `window-size=1280,960` and the resolved `user-agent`. Headed instances add `window-position=-32000,-32000`.

## Concurrency cap

A single semaphore limits open pages across every instance, headless and headed alike. The default is 8.

```go
browser.SetMaxConcurrency(4)
```

`SetMaxConcurrency(n)` replaces the semaphore and ignores `n <= 0`. Pages already holding a slot release it to the old semaphore, so the new cap applies to pages that start after the call. A page waiting for a slot gives up when the context ends, returning `acquireSem: context deadline exceeded` or `context canceled`.

## Close

`browser.Close()` closes every cached instance immediately and empties the cache. It is safe to call more than once, and later `Fetch` calls simply launch new instances. The evictor goroutine keeps running.

## SameSession bypasses the cache

`SameSession` browsers are never cached: each call launches a fresh Chrome on a temporary profile, and closes it and deletes the directory when the call returns. They still count against the concurrency cap. See [Cookie Sessions](/core-concepts-cookie-session).
