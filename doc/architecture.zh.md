# go-browser - 架構

> 返回 [README](./README.zh.md)

## 概覽

```mermaid
graph TB
    A[Fetch] --> B[Launcher]
    B --> C{Headless 金鑰?}
    C -->|是| D[依 headless 與 UA 隔離的瀏覽器]
    C -->|否| E[Chrome Cookie 工作階段]
    D --> F[導覽與捲動]
    E --> F
    F --> G[嘗試關閉 Cookie 同意]
    G --> H[快照合併]
    H --> I[Readability 與去重]
    I --> J[Markdown / HTML / JSON]
```

## 模組：Launcher

依 headless 與 User-Agent 管理 Chrome 生命週期，提供可重用實例與可注入 Cookie 的暫時設定檔。

```mermaid
graph TB
    subgraph Launcher
        A[ensureBrowser] --> B{快取存在?}
        B -->|是| C[重用並更新 lastUsed]
        B -->|否| D[launcher.New]
        D --> E[設定 headless、UA、no-sandbox]
        E --> F[chromePath 尋找]
        F --> G[Launch 與 Connect]
        G --> H[寫入 browsers map]
        H --> I[啟動 idle evictor]
        C --> J[回傳 browser]
        I --> J
    end
    K[launchWithSnapshot] --> L[複製 Cookies 檔]
    L --> M[由 OS keychain 解密]
    M --> N[暫存設定檔啟動]
    N --> O[SetCookies]
    O --> P[回傳 browser 與 cleanup]
```

## 模組：Fetch

核心內容萃取管線：導覽、穩定等待、嘗試關閉同意橫幅、多快照合併，並輸出 Markdown、HTML 或 JSON。

```mermaid
graph TB
    subgraph Fetch
        A[parseHref] --> B{requiresSession?}
        B -->|是| C[fetchWith SameSession]
        B -->|否| D{Headless 強制?}
        D -->|是| E[fetchWith headless]
        D -->|否| F[先 headless]
        F --> G{被擋 403/429/503?}
        G -->|是且有顯示| H[headed fallback]
        G -->|否| I[回傳結果]
        E --> I
        C --> I
        H --> I
    end
    J[load] --> K[建立 Page 與 Viewport]
    K --> L[StealthJS EvalOnNewDocument]
    L --> M[Navigate 與 WaitLoad]
    M --> N[檢查最終 URL 與狀態]
    N --> O[WaitDOMStable 與 SettleJS]
    O --> P[handleConsent]
    P --> Q[初始 HTML 快照]
    Q --> R[捲動迴圈與多快照]
    R --> S{Type?}
    S -->|HTML| T[Merge 與 InlineTime]
    S -->|Markdown| U[Readability 合併再轉 Markdown]
    S -->|JSON| V[Readability 再 HTMLToNode]
    T --> W[回傳 HTML]
    U --> X[去重後 Markdown]
    V --> Y[JSON 序列化]
```

## 模組：Cookie

從本機 Chrome 設定檔解密 Cookie，供 SameSession 模式注入暫存瀏覽器。

```mermaid
graph TB
    subgraph Cookie
        A[chromeSafeStoragePassword] --> B{平台}
        B -->|darwin| C[security find-generic-password]
        B -->|linux| D[secret-tool lookup]
        C --> E[deriveChromeCookieKey PBKDF2]
        D --> E
        E --> F[sqlite3 讀 Cookies]
        F --> G[decryptChromeCookie AES-CBC]
        G --> H[NetworkCookieParam 清單]
    end
    External[Chrome Profile] --> A
    H --> Inject[Browser.SetCookies]
```

## 模組：Markdown

HTML 合併、時間元素內嵌、結構化節點與 Markdown 去重。

```mermaid
graph TB
    subgraph Markdown
        A[Merge] --> B[解析多個 body 並附加]
        C[InlineTimeElements] --> D[time 改為文字節點]
        E[HTMLToNode] --> F[Node 樹]
        F --> G[DedupTree]
        H[HTMLToMarkdown] --> I[DedupMarkdownParagraphs]
    end
    Snapshots[HTML 快照] --> A
    ArticleHTML[文章 HTML] --> C
    C --> E
    C --> H
```

## 資料流

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
    else 一般
        Fetch->>Launcher: ensureBrowser(headless, UA)
        Launcher->>Chrome: 重用或新建實例
    end
    Fetch->>Chrome: Navigate + WaitLoad
    Fetch->>Chrome: settle + consent attempt
    loop ScrollCount
        Fetch->>Chrome: scroll + HTML snapshot
    end
    Fetch->>Markdown: Merge / Readability / HTMLToMarkdown 或 HTMLToNode
    Markdown-->>Fetch: content
    Fetch-->>Caller: Result
```

## 狀態機：瀏覽器快取

```mermaid
stateDiagram-v2
    [*] --> Empty
    Empty --> Active: ensureBrowser 建立
    Active --> Active: 相同 headless+UA 重用
    Active --> Evicted: idle > 5 分鐘
    Evicted --> Empty: Close 並自 map 刪除
    Active --> Closed: Close()
    Closed --> [*]
```

***

©️ 2026 [邱敬幃 Pardn Chiu](https://www.linkedin.com/in/pardnchiu)
