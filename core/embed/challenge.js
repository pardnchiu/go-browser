(cfg) => {
  const title = (document.title || "").toLowerCase().trim();
  if (cfg.titles.some((t) => title.includes(t))) {
    return true;
  }

  const visible = (el) => {
    const r = el.getBoundingClientRect();
    if (r.width < 20 || r.height < 20) {
      return false;
    }
    if (r.bottom <= 0 || r.right <= 0 || r.top >= innerHeight || r.left >= innerWidth) {
      return false;
    }
    for (let n = el; n && n.nodeType === 1; n = n.parentElement) {
      const s = getComputedStyle(n);
      if (s.display === "none" || s.visibility === "hidden" || parseFloat(s.opacity) === 0) {
        return false;
      }
    }
    return true;
  };

  const inPopup = (el) => {
    for (let n = el; n && n.nodeType === 1; n = n.parentElement) {
      if (getComputedStyle(n).position === "fixed") {
        return true;
      }
    }
    return false;
  };

  const any = (selectors, test) =>
    selectors.some((sel) => Array.from(document.querySelectorAll(sel)).some(test));

  if (any(cfg.popups, visible)) {
    return true;
  }

  if (any(cfg.responses, (el) => (el.value || "").trim() !== "")) {
    return false;
  }

  const textLength = ((document.body && document.body.innerText) || "").trim().length;
  return any(cfg.widgets, (el) => visible(el) && (textLength < cfg.interstitialText || inPopup(el)));
}
