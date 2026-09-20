# go-browser - Architecture

> Back to [README](../README.md)

## Overview

```mermaid
graph TB
    A[Fetch] --> B[Launcher]
    B --> C{Headless key?}
    C -->|Yes| D[Isolated browser by headless and UA]
    C -->|No| E[Chrome cookie session]
    D --> F[Navigate and scroll]
    E --> F
    F --> G[Attempt cookie consent dismissal]
    G --> H[Snapshot merge]
    H --> I[Readability and dedup]
    I --> J[Markdown / HTML / JSON]
```

## Module: Launcher

Manages Chrome lifecycle by headless and User-Agent keys, with reusable instances and temporary profiles that can receive injected cookies.

```mermaid
graph TB
    subgraph Launcher
        A[ensureBrowser] --> B{Cache hit?}
        B -->|Yes| C[Reuse and touch lastUsed]
        B -->|No| D[launcher.New]
        D --> E[Set headless, UA, no-sandbox]
        E --> F[chromePath lookup]
        F --> G[Launch and Connect]
        G --> H[Store in browsers map]
        H --> I[Start idle evictor]
        C --> J[Return browser]
        I --> J
    end
    K[launchWithSnapshot] --> L[Copy Cookies files]
    L --> M[Decrypt via OS keychain]
    M --> N[Launch temp profile]
    N --> O[SetCookies]
    O --> P[Return browser and cleanup]
```

## Module: Fetch

Core extraction pipeline: navigate, wait for stability, attempt consent dismissal, merge snapshots, and emit Markdown, HTML, or JSON.

```mermaid
graph TB
    subgraph Fetch
        A[parseHref] --> B{requiresSession?}
        B -->|Yes| C[fetchWith SameSession]
        B -->|No| D{Force headless?}
        D -->|Yes| E[fetchWith headless]
        D -->|No| F[Try headless first]
        F --> G{Blocked 403/429/503?}
        G -->|Yes and display exists| H[headed fallback]
        G -->|No| I[Return result]
        E --> I
        C --> I
        H --> I
    end
    J[load] --> K[Create page and viewport]
    K --> L[StealthJS EvalOnNewDocument]
    L --> M[Navigate and WaitLoad]
    M --> N[Check final URL and status]
    N --> O[WaitDOMStable and SettleJS]
    O --> P[handleConsent]
    P --> Q[Initial HTML snapshot]
    Q --> R[Scroll loop and multi-snapshot]
    R --> S{Type?}
    S -->|HTML| T[Merge and InlineTime]
    S -->|Markdown| U[Readability merge then Markdown]
    S -->|JSON| V[Readability then HTMLToNode]
    T --> W[Return HTML]
    U --> X[Deduped Markdown]
    V --> Y[JSON serialize]
```

## Module: Cookie

Decrypts cookies from the local Chrome profile for SameSession injection into a temporary browser.

```mermaid
graph TB
    subgraph Cookie
        A[chromeSafeStoragePassword] --> B{Platform}
        B -->|darwin| C[security find-generic-password]
        B -->|linux| D[secret-tool lookup]
        C --> E[deriveChromeCookieKey PBKDF2]
        D --> E
        E --> F[sqlite3 read Cookies]
        F --> G[decryptChromeCookie AES-CBC]
        G --> H[NetworkCookieParam list]
    end
    External[Chrome Profile] --> A
    H --> Inject[Browser.SetCookies]
```

## Module: Markdown

HTML merge, time-element inlining, structured nodes, and Markdown paragraph deduplication.

```mermaid
graph TB
    subgraph Markdown
        A[Merge] --> B[Parse bodies and append]
        C[InlineTimeElements] --> D[Replace time with text]
        E[HTMLToNode] --> F[Node tree]
        F --> G[DedupTree]
        H[HTMLToMarkdown] --> I[DedupMarkdownParagraphs]
    end
    Snapshots[HTML snapshots] --> A
    ArticleHTML[Article HTML] --> C
    C --> E
    C --> H
```

## Data Flow

```mermaid
sequenceDiagram
    participant Caller
    participant Fetch
    participant Launcher
    participant Chrome
    participant Cookie
    participant Markdown
    Caller->>Fetch: Fetch(ctx, href, timeout, opt)
    Fetch->>Fetch: prepareOpt / parseHref
    alt SameSession
        Fetch->>Launcher: launchWithSnapshot
        Launcher->>Cookie: extractChromeCookies
        Cookie-->>Launcher: cookies
        Launcher->>Chrome: temp profile + SetCookies
    else Normal
        Fetch->>Launcher: ensureBrowser(headless, UA)
        Launcher->>Chrome: reuse or create instance
    end
    Fetch->>Chrome: Navigate + WaitLoad
    Fetch->>Chrome: settle + consent attempt
    loop ScrollCount
        Fetch->>Chrome: scroll + HTML snapshot
    end
    Fetch->>Markdown: Merge / Readability / HTMLToMarkdown or HTMLToNode
    Markdown-->>Fetch: content
    Fetch-->>Caller: Result
```

## State Machine: Browser Cache

```mermaid
stateDiagram-v2
    [*] --> Empty
    Empty --> Active: ensureBrowser creates
    Active --> Active: reuse same headless+UA
    Active --> Evicted: idle > 5 minutes
    Evicted --> Empty: Close and delete from map
    Active --> Closed: Close()
    Closed --> [*]
```

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
