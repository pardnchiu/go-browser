# Consent Banners

How `Fetch` attempts to dismiss cookie consent banners before extracting, what it refuses to touch, and how the outcome is reported.

This is a precondition for extraction quality, not a browser automation feature. Many consent platforms load the article body only after consent, so removing the overlay alone yields an empty shell.

## Three stages

The embedded `consent.js` runs after settling and stops at the first stage that acts:

1. **Known platform selectors**: stable ids and data attributes of OneTrust, Cookiebot, Didomi, Usercentrics, Quantcast, Sourcepoint, Osano, TrustArc, CookieYes, Complianz, HubSpot and cookieconsent, searched in the document and every open shadow root
2. **Accept text inside the overlay**: locate the blocking layers first, then look for a button, link or submit whose label contains accept wording in English, Chinese, Japanese or Korean, and no reject, decline, manage or settings wording; labels longer than 40 characters are ignored
3. **Removal as a fallback**: remove the blocking layers and clear `overflow`, `position` and `height` from `html` and `body` to restore scrolling

## What counts as a blocking layer

All of the following must hold:

| Condition | Detail |
|---|---|
| Positioning | `position` is `fixed` or `sticky`, and the element is visible |
| Size | Covers at least 5% of the viewport, or scrolling is locked and z-index is 1000 or higher |
| Location | Not inside a `header` or `nav` |
| Vocabulary | Its text contains consent wording such as cookie, consent, privacy, GDPR, tracking, 隱私 or 同意 |

At most the top three candidates by z-index, then area, are considered. The vocabulary gate is required: layout-only heuristics classify ordinary sticky containers as overlays and delete real content. Shadow root traversal stops after 4000 elements.

### Login walls and paywalls

A layer whose text contains sign in, log in, subscribe, register, 登入 or 訂閱 and no accept wording is a gate, not a consent banner. Gates are reported as `skipped` and are neither text-clicked nor removed; stage 1 known-platform selectors still run before the gate check.

## Passes and reporting

Up to two passes run, with a settle after each one that acts, because some platforms show a second layer after the first click. Clicked elements are marked with `data-gb-consent`, so a second pass never clicks the same control twice.

| `Result.Consent` | Meaning |
|---|---|
| `selector` | Clicked a known platform button |
| `text` | Clicked an accept button found by its label |
| `removed` | Removed the blocking layers |
| `none` | No consent layer found |
| `skipped` | A login wall or paywall was found and left alone |
| empty | The script failed to evaluate, or the result is a JSON/XML document |

Values from multiple passes are comma-joined, for example `selector,text`.

When extracted content looks empty, `Result.Consent` shows which stage acted, if any.
