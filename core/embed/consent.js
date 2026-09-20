() => {
  const ACCEPT_SELECTORS = [
    "#onetrust-accept-btn-handler",
    "#accept-recommended-btn-handler",
    "#CybotCookiebotDialogBodyLevelButtonLevelOptinAllowAll",
    "#CybotCookiebotDialogBodyButtonAccept",
    "#CybotCookiebotDialogBodyLevelButtonAccept",
    "#didomi-notice-agree-button",
    "button#didomi-notice-agree-button",
    "[data-testid='uc-accept-all-button']",
    "[data-testid='uc-customize-anchor'] ~ button",
    "#uc-btn-accept-banner",
    ".qc-cmp2-summary-buttons button[mode='primary']",
    "button.sp_choice_type_11",
    ".osano-cm-accept-all",
    "#truste-consent-button",
    ".cky-btn-accept",
    ".cmplz-accept",
    "[data-cookiebanner='accept_button']",
    "[data-cky-tag='accept-button']",
    "#hs-eu-confirmation-button",
    ".cc-allow",
    ".js-accept-all-cookies",
    "[aria-label='Accept all'][role='button']",
  ];

  const ACCEPT_TEXT = [
    "accept all",
    "allow all",
    "accept cookies",
    "allow cookies",
    "i accept",
    "i agree",
    "accept",
    "agree",
    "allow",
    "got it",
    "understood",
    "全部同意",
    "我同意",
    "同意",
    "接受",
    "允許",
    "允许",
    "すべて同意",
    "同意する",
    "모두 동의",
    "동의",
  ];

  const REJECT_TEXT = [
    "reject",
    "decline",
    "deny",
    "disagree",
    "necessary only",
    "only necessary",
    "essential only",
    "manage",
    "settings",
    "preferences",
    "customize",
    "customise",
    "more options",
    "learn more",
    "拒絕",
    "拒绝",
    "不同意",
    "管理",
    "設定",
    "设置",
    "自訂",
    "更多選項",
    "詳細設定",
    "거부",
  ];

  const GATE_TEXT = [
    "sign in",
    "log in",
    "login",
    "subscribe",
    "subscription",
    "paywall",
    "create account",
    "register",
    "登入",
    "登录",
    "訂閱",
    "订阅",
    "註冊",
    "注册",
    "會員",
    "会员",
  ];

  const CONSENT_TEXT = [
    "cookie",
    "consent",
    "privacy",
    "gdpr",
    "tracking",
    "personal data",
    "legitimate interest",
    "隱私",
    "隐私",
    "個資",
    "个人资料",
    "追蹤",
    "追踪",
    "同意",
    "クッキー",
    "プライバシー",
    "쿠키",
    "개인정보",
  ];

  const MAX_NODES = 4000;

  const roots = () => {
    const list = [document];
    const stack = [document];
    let seen = 0;
    while (stack.length) {
      const root = stack.pop();
      for (const el of root.querySelectorAll("*")) {
        if (++seen > MAX_NODES) return list;
        if (el.shadowRoot) {
          list.push(el.shadowRoot);
          stack.push(el.shadowRoot);
        }
      }
    }
    return list;
  };

  const visible = (el) => {
    const style = getComputedStyle(el);
    if (style.display === "none" || style.visibility === "hidden" || Number(style.opacity) === 0) return false;
    const rect = el.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0;
  };

  const label = (el) => (el.innerText || el.value || el.getAttribute("aria-label") || "").replace(/\s+/g, " ").trim().toLowerCase();

  const isAccept = (text) => {
    if (!text || text.length > 40) return false;
    if (REJECT_TEXT.some((t) => text.includes(t))) return false;
    return ACCEPT_TEXT.some((t) => text.includes(t));
  };

  const scrollLocked = () => {
    for (const el of [document.documentElement, document.body]) {
      const style = getComputedStyle(el);
      if (style.overflow === "hidden" || style.overflowY === "hidden" || style.position === "fixed") return true;
    }
    return false;
  };

  const overlays = () => {
    const vw = innerWidth;
    const vh = innerHeight;
    const locked = scrollLocked();
    const found = [];
    for (const root of roots()) {
      for (const el of root.querySelectorAll("*")) {
        const style = getComputedStyle(el);
        if (style.position !== "fixed" && style.position !== "sticky") continue;
        if (!visible(el)) continue;
        const rect = el.getBoundingClientRect();
        const area = rect.width * rect.height;
        const ratio = area / (vw * vh);
        const z = parseInt(style.zIndex, 10) || 0;
        if (ratio < 0.05 && !(locked && z >= 1000)) continue;
        if (el.closest("header, nav")) continue;
        found.push({ el, z, area });
      }
    }
    return found.sort((a, b) => b.z - a.z || b.area - a.area).slice(0, 3).map((o) => o.el);
  };

  const bySelector = () => {
    for (const root of roots()) {
      for (const sel of ACCEPT_SELECTORS) {
        let el;
        try {
          el = root.querySelector(sel);
        } catch (e) {
          continue;
        }
        if (el && visible(el) && !el.hasAttribute("data-gb-consent")) {
          el.setAttribute("data-gb-consent", "1");
          el.click();
          return true;
        }
      }
    }
    return false;
  };

  const byText = (layers) => {
    for (const layer of layers) {
      for (const el of layer.querySelectorAll("button, [role='button'], a, input[type='button'], input[type='submit']")) {
        if (!visible(el) || el.hasAttribute("data-gb-consent")) continue;
        if (isAccept(label(el))) {
          el.setAttribute("data-gb-consent", "1");
          el.click();
          return true;
        }
      }
    }
    return false;
  };

  const strip = (layers) => {
    let removed = false;
    for (const layer of layers) {
      layer.remove();
      removed = true;
    }
    if (removed) {
      for (const el of [document.documentElement, document.body]) {
        el.style.removeProperty("overflow");
        el.style.removeProperty("overflow-y");
        el.style.removeProperty("position");
        el.style.removeProperty("height");
      }
    }
    return removed;
  };

  if (bySelector()) return "selector";

  const gate = (el) => {
    const text = (el.innerText || "").toLowerCase();
    return GATE_TEXT.some((t) => text.includes(t)) && !ACCEPT_TEXT.some((t) => text.includes(t));
  };

  const all = overlays();
  const layers = all.filter((el) => CONSENT_TEXT.some((t) => (el.innerText || "").toLowerCase().includes(t)));
  if (layers.length === 0) return all.some(gate) ? "skipped" : "none";
  if (layers.some(gate)) return "skipped";

  if (byText(layers)) return "text";
  if (strip(layers)) return "removed";
  return "none";
}
