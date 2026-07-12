# go-browser - 架構

> 返回 [README](./README.zh.md)

## 概覽

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

## 模組：Launcher

負責啟動與管理 Chrome 瀏覽器實例，支援 headless 與非 headless 模式，並提供閒置自動回收機制。

```mermaid
graph TB
    subgraph Launcher
        A[ensureBrowser] --> B[launcher.New]
        B --> C[chromePath 偵測]
        C --> D[Launch]
        D --> E[Browser 連線]
        E --> F[Browser Pool]
        F --> G[Evictor]
        G -->|5 分鐘閒置| H[Close Browser]
    end
    I[SetMaxConcurrency] --> F
    J[acquireSem] --> F
```

## 模組：Cookie Session

從本機 Chrome 設定檔提取並解密 Cookie，注入至臨時瀏覽器實例以存取需登入的頁面。

```mermaid
graph TB
    subgraph CookieSession
        A[launchWithSnapshot] --> B[複製 Chrome 設定檔]
        B --> C[extractChromeCookies]
        C --> D[chromeSafeStoragePassword]
        D --> E[deriveChromeCookieKey]
        E --> F[decryptChromeCookie]
        F --> G[SetCookies]
    end
    H[macOS Keychain] --> D
    I[Linux secret-tool] --> D
    J[sqlite3] --> C
```

## 模組：Fetch

核心擷取流程，負責導航、滾動、快照合併與內容提取。

```mermaid
graph TB
    subgraph Fetch
        A[Fetch] --> B{requiresSession?}
        B -->|Yes| C[launchWithSnapshot]
        B -->|No| D{attemptHeadless}
        D --> E[ensureBrowser]
        C --> F[load]
        E --> F
        F --> G[Navigate + WaitLoad]
        G --> H[Stealth JS 注入]
        H --> I[Settle JS]
        I --> J[滾動迴圈]
        J --> K[快照收集]
        K --> L{Type}
        L -->|Markdown| M[Readability + HTMLToMarkdown]
        L -->|HTML| N[Merge + InlineTimeElements]
        L -->|JSON| O[HTMLToNode]
        M --> P[DedupMarkdownParagraphs]
        N --> Q[Result]
        O --> Q
        P --> Q
    end
```

## 模組：Interactive Tabs

互動式分頁管理，支援建立、點擊、輸入、滾動、執行 JavaScript 與快照擷取。

```mermaid
graph TB
    subgraph InteractiveTabs
        A[CreateTab] --> B[互動式 Browser]
        B --> C[Page 建立]
        C --> D[navigate]
        D --> E[WaitLoad + Settle]
        E --> F[tab 註冊]
        F --> G[TabClick]
        F --> H[TabType]
        F --> I[TabScroll]
        F --> J[TabEval]
        F --> K[TabSnapshot]
        F --> L[TabNavigate]
        G --> M[page.Eval]
        H --> M
        I --> M
        J --> M
        K --> N[snapshot]
        L --> D
    end
    O[CloseTab] --> P[釋放資源]
    P --> Q{無分頁?}
    Q -->|Yes| R[關閉 Browser]
```

## 模組：Content Processing

HTML 處理與內容轉換，包含快照合併、時間元素內聯、Markdown 轉換與去重。

```mermaid
graph TB
    subgraph ContentProcessing
        A[Merge] --> B[HTML Parse]
        B --> C[findBody]
        C --> D[合併 Body 子節點]
        D --> E[HTML Render]
        F[InlineTimeElements] --> G[收集 time 節點]
        G --> H[替換為文字節點]
        H --> I[HTML Render]
        J[HTMLToMarkdown] --> K[HTML Walk]
        K --> L[標題/段落/清單轉換]
        L --> M[collapse]
        N[HTMLToNode] --> O[HTML Walk]
        O --> P[Node 樹建構]
        P --> Q[DedupTree]
        Q --> R[flattenChildren]
        S[DedupMarkdownParagraphs] --> T[段落去重]
    end
```

## 資料流

```mermaid
sequenceDiagram
    participant Caller
    participant Fetch
    participant Launcher
    participant Page
    participant Readability
    participant Markdown

    Caller->>Fetch: Fetch(ctx, url, timeout, opt)
    Fetch->>Fetch: prepareOpt(opt)
    Fetch->>Fetch: parseHref(url)
    Fetch->>Launcher: ensureBrowser / launchWithSnapshot
    Launcher->>Page: Page(TargetCreateTarget)
    Page->>Page: SetViewport
    Page->>Page: EvalOnNewDocument(StealthJS)
    Page->>Page: Navigate(url)
    Page->>Page: WaitLoad
    Page->>Page: WaitDOMStable
    Page->>Page: SettleJS
    loop ScrollCount 次
        Page->>Page: Eval(scroll)
        Page->>Page: WaitDOMStable
        Page->>Page: HTML() → snapshot
    end
    Fetch->>Fetch: Merge(snapshots)
    Fetch->>Fetch: InlineTimeElements
    Fetch->>Readability: FromReader(html)
    Readability-->>Fetch: Article
    Fetch->>Markdown: HTMLToMarkdown
    Markdown->>Markdown: DedupMarkdownParagraphs
    Fetch-->>Caller: Result
```

## 狀態機

```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Launching: Fetch / CreateTab
    Launching --> Headless: attemptHeadless=true
    Launching --> Session: SameSession / requiresSession
    Headless --> Navigating
    Session --> Navigating
    Navigating --> Scrolling: WaitLoad + Settle
    Scrolling --> Extracting: ScrollCount 完成
    Extracting --> Done: Markdown / HTML / JSON
    Done --> [*]
    Navigating --> Retry: 403 / 429 / 503
    Retry --> Session: hasDisplay
    Retry --> Done: 無法重試
```

***

©️ 2025 [邱敬幃 Pardn Chiu](https://linkedin.com/in/pardnchiu)