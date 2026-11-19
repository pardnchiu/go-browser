# 同意橫幅

`Fetch` 如何在萃取前嘗試關閉 Cookie 同意橫幅、哪些東西不碰，以及結果如何回報。

這是萃取品質的前置條件，不是瀏覽器操作功能。許多同意管理平台在同意之後才載入正文，只移除遮罩會拿到空殼。

## 三階段

內嵌的 `consent.js` 在等待穩定後執行，任一階段有動作即停：

1. **已知平台選擇器**：OneTrust、Cookiebot、Didomi、Usercentrics、Quantcast、Sourcepoint、Osano、TrustArc、CookieYes、Complianz、HubSpot、cookieconsent 的穩定 id 與 data 屬性，搜尋範圍含文件與所有 open shadow root
2. **遮罩內的同意字樣**：先定位阻擋層，再在其中找按鈕、連結或送出鈕，標籤含中英日韓同意字樣且不含拒絕、管理、設定類字樣；超過 40 字元的標籤忽略
3. **降級移除**：移除阻擋層，並清除 `html` 與 `body` 的 `overflow`、`position`、`height` 以還原捲動

## 什麼算阻擋層

需同時滿足：

| 條件 | 細節 |
|---|---|
| 定位 | `position` 為 `fixed` 或 `sticky` 且可見 |
| 尺寸 | 覆蓋視窗 5% 以上，或捲動被鎖且 z-index ≥ 1000 |
| 位置 | 不在 `header` 或 `nav` 之內 |
| 詞彙 | 文字含 cookie、consent、privacy、GDPR、tracking、隱私、同意等詞 |

依 z-index、再依面積只取前三個候選。詞彙閘是必要的：只看版面特徵會把一般的 sticky 容器當成遮罩並刪除真正的內容。shadow root 走訪在 4000 個元素後停止。

### 登入牆與付費牆

文字含 sign in、log in、subscribe、register、登入、訂閱且無同意字樣的阻擋層屬於閘門，不是同意橫幅。閘門回報 `skipped`，不做文字點擊也不移除；但第 1 階段的已知平台選擇器在閘門判斷之前就會執行。

## 輪次與回報

最多執行兩輪，有動作的那輪結束後會再等待穩定，因為有些平台點完第一層才出現第二層。點過的元素標上 `data-gb-consent`，第二輪不會再點同一個控制項。

| `Result.Consent` | 意義 |
|---|---|
| `selector` | 點了已知平台的按鈕 |
| `text` | 依標籤找到同意按鈕並點擊 |
| `removed` | 移除了阻擋層 |
| `none` | 沒有找到同意層 |
| `skipped` | 找到登入牆或付費牆，未處理 |
| 空字串 | 腳本執行失敗，或結果為 JSON／XML 文件 |

多輪結果以逗號串接，例如 `selector,text`。

擷取內容看起來是空的時，`Result.Consent` 會顯示是哪個階段動作（或都沒有）。
