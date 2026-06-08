package goBrowser

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	gorod "github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	readability "github.com/go-shiori/go-readability"
)

type tab struct {
	page        *gorod.Page
	opt         *Option
	href        string
	parsed      *url.URL
	finalURL    string
	status      int
	contentType string
	release     func()
	ctx         context.Context
}

var (
	interactiveMu      sync.Mutex
	interactiveBrowser *gorod.Browser
	interactiveCleanup func()
	tabs               = map[string]*tab{}
	tabCounter         int64
)

func CreateTab(ctx context.Context, href string, opt *Option) (string, error) {
	o := prepareOpt(opt)
	parsed, err := parseHref(href)
	if err != nil {
		return "", err
	}

	release, err := acquireSem(ctx)
	if err != nil {
		return "", fmt.Errorf("acquireSem: %w", err)
	}

	interactiveMu.Lock()
	defer interactiveMu.Unlock()

	if interactiveBrowser == nil {
		var b *gorod.Browser
		var cleanup func()
		if o.SameSession {
			b, cleanup, err = launchWithSnapshot(ctx, o.Profile, o.UserAgent, !hasDisplay())
			if err != nil {
				release()
				return "", err
			}
		} else {
			b, err = ensureBrowser(o.UserAgent, !hasDisplay())
			if err != nil {
				release()
				return "", err
			}
		}
		interactiveBrowser = b
		interactiveCleanup = cleanup
	}

	page, err := interactiveBrowser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		release()
		return "", fmt.Errorf("browser.Page: %w", err)
	}
	page = page.Context(ctx)

	if o.Viewport != nil {
		scale := o.Viewport.DeviceScaleFactor
		if scale == 0 {
			scale = 1
		}
		if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:             o.Viewport.Width,
			Height:            o.Viewport.Height,
			DeviceScaleFactor: scale,
		}); err != nil {
			_ = page.Close()
			release()
			return "", fmt.Errorf("page.SetViewport: %w", err)
		}
	}

	if o.StealthJS != "" {
		if _, err := page.EvalOnNewDocument(o.StealthJS); err != nil {
			_ = page.Close()
			release()
			return "", fmt.Errorf("page.EvalOnNewDocument: %w", err)
		}
	}

	t := &tab{
		page:    page,
		opt:     o,
		parsed:  parsed,
		release: release,
		ctx:     ctx,
	}
	if err := t.navigate(href); err != nil {
		_ = page.Close()
		release()
		return "", err
	}

	tabCounter++
	id := "tab-" + strconv.FormatInt(tabCounter, 10)
	tabs[id] = t
	return id, nil
}

func CloseTab(tabID string) error {
	interactiveMu.Lock()
	defer interactiveMu.Unlock()

	t, ok := tabs[tabID]
	if !ok {
		return fmt.Errorf("tab %q not found", tabID)
	}
	_ = t.page.Close()
	t.release()
	delete(tabs, tabID)

	if len(tabs) == 0 {
		if interactiveCleanup != nil {
			interactiveCleanup()
			interactiveCleanup = nil
		}
		interactiveBrowser = nil
	}
	return nil
}

func TabSnapshot(tabID string) (*Result, error) {
	t, err := getTab(tabID)
	if err != nil {
		return nil, err
	}
	return t.snapshot()
}

func TabClick(tabID, selector string) error {
	t, err := getTab(tabID)
	if err != nil {
		return err
	}
	return t.click(selector)
}

func TabType(tabID, selector, text string) error {
	t, err := getTab(tabID)
	if err != nil {
		return err
	}
	return t.typeText(selector, text)
}

func TabScroll(tabID string, count int) error {
	t, err := getTab(tabID)
	if err != nil {
		return err
	}
	return t.scroll(count)
}

func TabNavigate(tabID, href string) error {
	t, err := getTab(tabID)
	if err != nil {
		return err
	}
	return t.navigate(href)
}

func TabEval(tabID, js string) (string, error) {
	t, err := getTab(tabID)
	if err != nil {
		return "", err
	}
	return t.eval(js)
}

func getTab(tabID string) (*tab, error) {
	interactiveMu.Lock()
	defer interactiveMu.Unlock()
	t, ok := tabs[tabID]
	if !ok {
		return nil, fmt.Errorf("tab %q not found", tabID)
	}
	return t, nil
}

func (t *tab) navigate(href string) error {
	if err := t.page.Navigate(href); err != nil {
		return fmt.Errorf("page.Navigate: %w", err)
	}
	if err := t.page.WaitLoad(); err != nil {
		return fmt.Errorf("page.WaitLoad: %w", err)
	}

	// Wait for DOM hydration so subsequent Click/Type/Snapshot see mounted elements.
	_ = t.page.WaitDOMStable(t.opt.IdleWait, 0.01)
	if t.opt.SettleJS != "" {
		settleCtx, settleCancel := context.WithTimeout(t.ctx, t.opt.IdleWait)
		_, _ = t.page.Context(settleCtx).Eval(t.opt.SettleJS)
		settleCancel()
	}

	t.href = href
	if parsed, err := url.Parse(href); err == nil {
		t.parsed = parsed
	}
	if info, err := t.page.Info(); err == nil && info.URL != "" {
		t.finalURL = info.URL
	} else {
		t.finalURL = href
	}
	if v, err := t.page.Eval(`() => { const e = performance.getEntriesByType("navigation")[0]; return e ? e.responseStatus : 0 }`); err == nil {
		t.status = v.Value.Int()
	}
	if v, err := t.page.Eval(`() => document.contentType`); err == nil {
		t.contentType = v.Value.String()
	}
	return nil
}

func (t *tab) click(selector string) error {
	if _, err := t.page.Eval(`(s) => document.querySelector(s)?.click()`, selector); err != nil {
		return fmt.Errorf("page.Eval click: %w", err)
	}
	t.sleep(t.opt.IdleWait)
	return nil
}

func (t *tab) typeText(selector, text string) error {
	js := `(s, t) => { const e = document.querySelector(s); if (!e) return false; e.focus(); const p = e.tagName === 'TEXTAREA' ? window.HTMLTextAreaElement.prototype : window.HTMLInputElement.prototype; Object.getOwnPropertyDescriptor(p, 'value').set.call(e, t); e.dispatchEvent(new Event('input', {bubbles: true})); e.dispatchEvent(new Event('change', {bubbles: true})); return true }`
	if _, err := t.page.Eval(js, selector, text); err != nil {
		return fmt.Errorf("page.Eval type: %w", err)
	}
	t.sleep(t.opt.IdleWait)
	return nil
}

func (t *tab) scroll(count int) error {
	if count == 0 {
		count = t.opt.ScrollCount
	}
scrollLoop:
	for i := 0; i < count; i++ {
		if _, err := t.page.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`); err != nil {
			return fmt.Errorf("page.Eval scroll: %w", err)
		}
		delay := time.Duration(1+rand.IntN(5)) * time.Second
		select {
		case <-t.ctx.Done():
			break scrollLoop
		case <-time.After(delay):
		}
	}
	return nil
}

func (t *tab) eval(js string) (string, error) {
	v, err := t.page.Eval("() => (" + js + ")")
	if err != nil {
		return "", fmt.Errorf("page.Eval: %w", err)
	}
	return v.Value.String(), nil
}

func (t *tab) snapshot() (*Result, error) {
	if strings.Contains(t.contentType, "json") || strings.Contains(t.contentType, "xml") {
		raw := ""
		if strings.Contains(t.contentType, "json") {
			if v, err := t.page.Eval(`() => document.body.innerText`); err == nil {
				raw = v.Value.String()
			}
		} else {
			if v, err := t.page.Eval(`() => { const s = document.getElementById('webkit-xml-viewer-source-xml'); return s ? s.innerHTML : new XMLSerializer().serializeToString(document) }`); err == nil {
				raw = v.Value.String()
			}
		}
		if raw != "" {
			return &Result{
				Href:        t.href,
				FinalURL:    t.finalURL,
				Content:     raw,
				ContentType: t.contentType,
				Status:      t.status,
			}, nil
		}
	}

	htmlSrc, err := t.page.HTML()
	if err != nil {
		return nil, fmt.Errorf("page.HTML: %w", err)
	}

	htmlSrc, err = InlineTimeElements(htmlSrc)
	if err != nil {
		return nil, fmt.Errorf("InlineTimeElements: %w", err)
	}

	if t.opt.Type == TypeHTML {
		return &Result{
			Href:     t.href,
			FinalURL: t.finalURL,
			Content:  htmlSrc,
			Status:   t.status,
		}, nil
	}

	article, err := readability.FromReader(strings.NewReader(htmlSrc), t.parsed)
	if err != nil {
		return nil, fmt.Errorf("readability: %w", err)
	}
	content := strings.TrimSpace(article.Content)
	if content == "" {
		content = htmlSrc
	}
	md, err := HTMLToMarkdown(content, t.href, t.opt.KeepLinks)
	if err != nil {
		return nil, fmt.Errorf("HTMLToMarkdown: %w", err)
	}
	if t.opt.MaxLength > 0 && len(md) > t.opt.MaxLength {
		md = md[:t.opt.MaxLength]
	}

	result := &Result{
		Href:     t.href,
		FinalURL: t.finalURL,
		Content:  md,
		Title:    article.Title,
		Author:   article.Byline,
		Excerpt:  article.Excerpt,
		Status:   t.status,
	}
	if article.PublishedTime != nil {
		result.PublishedAt = article.PublishedTime.Format(time.RFC3339)
	}

	if t.opt.Type == TypeJSON {
		tree, err := HTMLToNode(content, t.href, t.opt.KeepLinks)
		if err != nil {
			return nil, fmt.Errorf("HTMLToNode: %w", err)
		}
		result.Tree = tree
		result.Content = ""
		b, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal json: %w", err)
		}
		return &Result{
			Href:     t.href,
			FinalURL: t.finalURL,
			Content:  string(b),
			Status:   t.status,
		}, nil
	}
	return result, nil
}

func (t *tab) sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	select {
	case <-t.ctx.Done():
	case <-time.After(d):
	}
}
