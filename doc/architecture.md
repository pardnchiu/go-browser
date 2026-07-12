# go-browser - Architecture

> Back to [README](../README.md)

## Overview

```mermaid
graph TB
    A[Fetch / CreateTab] --> B[Launcher]
    B --> C{Headless?}
    C -->|Yes| D[Stealth JS]
    C -->|No| E[Cookie Session]
    D --> F[Page Navigate + Scroll]
    E --> F
    F --> G[Snapshot Merge]
    G --> H[Readability + Dedup]
    H --> I[Markdown / HTML / JSON]
```

## Module: Launcher

Manages Chrome browser lifecycle — singleton headless instance with idle eviction, or temporary profile-snapshot instances with cookie injection.

```mermaid
graph TB
    subgraph Launcher
        A[ensureBrowser] --> B{Browser exists?}
        B -->|Yes| C[Reuse singleton]
        B -->|No| D[launcher.New]
        D --> E[Set flags: headless, no-sandbox, user-agent]
        E --> F[chromePath lookup]
        F --> G[Launch + Connect]
        G --> H[Store as singleton]
        H --> I[Start evictor goroutine]
        C --> J[Return browser]
        I --> J
    end
    K[launchWithSnapshot] --> L[Copy Chrome profile cookies]
    L --> M[Decrypt cookies via OS keychain]
    M --> N[Launch temp browser with cookies]
    N --> O[SetCookies on browser]
    O --> P[Return browser + cleanup]
```

## Module: Fetch

Core content extraction pipeline — navigates to a URL, simulates scrolling, captures multiple snapshots, and produces Markdown/HTML/JSON output.

```mermaid
graph TB
    subgraph Fetch
        A[parseHref] --> B{requiresSession?}
        B -->|Yes| C[fetchWith: SameSession]
        B -->|No| D{Headless?}
        D -->|Yes| E[fetchWith: headless]
        D -->|No| F[fetchWith: headed first]
        F --> G{needsRetry?}
        G -->|Yes| H[fetchWith: headed fallback]
        G -->|No| I[Return result]
        E --> I
        C --> I
    end
    J[load] --> K[Page create + SetViewport]
    K --> L[StealthJS: EvalOnNewDocument]
    L --> M[Navigate + WaitLoad]
    M --> N[Check final URL for 404/403]
    N --> O[WaitDOMStable + SettleJS]
    O --> P[Capture initial snapshot]
    P --> Q[Scroll loop: random delay + scroll]
    Q --> R[Capture snapshots per scroll]
    R --> S{Type?}
    S -->|HTML| T[Merge snapshots → InlineTimeElements]
    S -->|Markdown| U[Per-snapshot: readability → merge content]
    S -->|JSON| V[Per-snapshot: readability → HTMLToNode]
    T --> W[Return HTML]
    U --> X[HTMLToMarkdown → DedupParagraphs]
    V --> Y[JSON marshal]
    X --> Z[Return Markdown]
    Y --> AA[Return JSON]
```

## Module: Interactive Tabs

Stateful tab management for multi-step browser interactions — click, type, scroll, eval, and snapshot.

```mermaid
graph TB
    subgraph InteractiveTabs
        A[CreateTab] --> B{Interactive browser exists?}
        B -->|No| C[Launch browser: SameSession or headless]
        B -->|Yes| D[Reuse browser]
        C --> E[Create page + SetViewport + StealthJS]
        D --> E
        E --> F[navigate: Navigate + WaitLoad + Settle]
        F --> G[Store tab in map]
        G --> H[Return tabID]
    end
    I[TabClick] --> J[Eval: querySelector.click]
    K[TabType] --> L[Eval: focus + set value + dispatch events]
    M[TabScroll] --> N[Eval: scrollTo + random delay loop]
    O[TabEval] --> P[Eval: custom JS]
    Q[TabSnapshot] --> R[HTML → InlineTimeElements → readability → Markdown]
    S[CloseTab] --> T[Close page + release sem]
    T --> U{No tabs left?}
    U -->|Yes| V[Close interactive browser]
    U -->|No| W[Keep browser alive]
```

## Module: Cookie Extraction

Cross-platform Chrome cookie decryption — extracts cookies from the local Chrome profile SQLite database and decrypts them using OS keychain credentials.

```mermaid
graph TB
    subgraph CookieExtraction
        A[extractChromeCookies] --> B[chromeSafeStoragePassword]
        B --> C{OS?}
        C -->|macOS| D[security find-generic-password]
        C -->|Linux| E[secret-tool lookup]
        D --> F[Derive key: PBKDF2-SHA1]
        E --> F
        F --> G[sqlite3: SELECT cookies]
        G --> H{encrypted_value?}
        H -->|Yes| I[AES-CBC decrypt: v10 prefix]
        H -->|No| J[Use plaintext value]
        I --> K[Strip padding + prefix]
        J --> L[Build NetworkCookieParam]
        K --> L
        L --> M[Return cookies]
    end
```

## Module: Content Processing

HTML processing pipeline — snapshot merging, time element inlining, readability extraction, Markdown conversion, and deduplication.

```mermaid
graph TB
    subgraph ContentProcessing
        A[Merge] --> B[Parse first snapshot as base]
        B --> C[For each subsequent snapshot]
        C --> D[Extract body children]
        D --> E[Append to base body]
        E --> F[Render merged HTML]
        G[InlineTimeElements] --> H[Find time nodes]
        H --> I[Extract datetime attr + inner text]
        I --> J[Replace with text node]
        J --> K[Handle anchor-wrapped times]
        K --> L[Render HTML]
        M[HTMLToMarkdown] --> N[Walk HTML tree]
        N --> O[Map tags to Markdown syntax]
        O --> P[collapse whitespace]
        Q[DedupMarkdownParagraphs] --> R[Split by paragraphs]
        R --> S[Hash each paragraph]
        S --> T[Skip duplicates]
        T --> U[Join unique paragraphs]
        V[HTMLToNode] --> W[Walk HTML tree → Node tree]
        W --> X[DedupTree: hash-based dedup]
        X --> Y[flattenChildren: unwrap single-child nodes]
    end
```

## Data Flow

```mermaid
sequenceDiagram
    participant Caller
    participant Fetch
    participant Launcher
    participant Page
    participant Readability
    participant Markdown

    Caller->>Fetch: Fetch(ctx, url, timeout, opt)
    Fetch->>Fetch: parseHref + prepareOpt
    Fetch->>Launcher: ensureBrowser / launchWithSnapshot
    Launcher->>Launcher: Launch Chrome + connect
    Launcher-->>Fetch: *Browser
    Fetch->>Page: Page create + SetViewport
    Fetch->>Page: EvalOnNewDocument(stealthJS)
    Fetch->>Page: Navigate(url) + WaitLoad
    Fetch->>Page: Check final URL for 404/403
    Fetch->>Page: WaitDOMStable + SettleJS
    Fetch->>Page: HTML() → initial snapshot
    loop ScrollCount times
        Fetch->>Page: Eval(smooth scroll)
        Fetch->>Page: WaitDOMStable
        Fetch->>Page: HTML() → snapshot
    end
    Fetch->>Readability: FromReader(per-snapshot HTML)
    Readability-->>Fetch: Article (Title, Content, Byline)
    Fetch->>Markdown: HTMLToMarkdown(merged content)
    Markdown-->>Fetch: Markdown string
    Fetch->>Markdown: DedupMarkdownParagraphs
    Markdown-->>Fetch: Deduped Markdown
    Fetch-->>Caller: *Result{Content, Title, ...}
```

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)