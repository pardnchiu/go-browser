package goBrowser

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"
	"time"

	gorod "github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	readability "github.com/go-shiori/go-readability"
)

const (
	TypeMarkdown = iota
	TypeHTML
	TypeJSON
)

const defaultScrollCount = 3

//go:embed embed/stealth.js
var defaultStealthJS string

//go:embed embed/listener.js
var defaultListenerJS string

type Viewport struct {
	Width             int
	Height            int
	DeviceScaleFactor float64
}

type Option struct {
	IdleWait    time.Duration
	MaxLength   int
	UserAgent   string
	KeepLinks   bool
	StealthJS   string
	SettleJS    string
	Viewport    *Viewport
	SameSession bool
	Profile     string
	Type        int
	ScrollCount int
}

type Result struct {
	Href        string
	FinalURL    string
	Content     string
	Title       string
	Author      string
	PublishedAt string
	Excerpt     string
	Status      int
	Tree        []*Node `json:",omitempty"`
}

type Error struct {
	Status int
	Href   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("http %d: %s", e.Status, e.Href)
}

const (
	defaultIdleWait  = 2 * time.Second
	defaultMaxLength = 1 << 20
)

func prepareOpt(opt *Option) *Option {
	o := Option{}
	if opt != nil {
		o = *opt
	}
	if o.IdleWait == 0 {
		o.IdleWait = defaultIdleWait
	}
	if o.MaxLength == 0 {
		o.MaxLength = defaultMaxLength
	}
	if o.StealthJS == "" {
		o.StealthJS = defaultStealthJS
	}
	if o.SettleJS == "" {
		o.SettleJS = defaultListenerJS
	}
	if o.Viewport == nil {
		o.Viewport = &Viewport{Width: 1280, Height: 960}
	}
	if o.Profile == "" {
		o.Profile = "Default"
	}
	if o.ScrollCount == 0 {
		o.ScrollCount = defaultScrollCount
	}
	return &o
}

func parseHref(href string) (*url.URL, error) {
	parsed, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("url.Parse: %w", err)
	}
	if parsed.Scheme == "" || !strings.Contains(parsed.Hostname(), ".") {
		return nil, fmt.Errorf("invalid url: %s", href)
	}
	return parsed, nil
}

func Fetch(ctx context.Context, href string, timeout time.Duration, opt *Option) (*Result, error) {
	o := prepareOpt(opt)
	parsed, err := parseHref(href)
	if err != nil {
		return nil, err
	}

	if o.SameSession {
		b, cleanup, err := launchWithSnapshot(ctx, o.Profile, o.UserAgent, !hasDisplay())
		if err != nil {
			return nil, err
		}
		defer cleanup()
		return load(ctx, b, href, parsed, timeout, o)
	}

	b, err := ensureBrowser(o.UserAgent, !hasDisplay())
	if err != nil {
		return nil, err
	}
	return load(ctx, b, href, parsed, timeout, o)
}

func load(ctx context.Context, b *gorod.Browser, href string, parsed *url.URL, timeout time.Duration, opt *Option) (*Result, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	release, err := acquireSem(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquireSem: %w", err)
	}
	defer release()

	page, err := b.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("browser.Page: %w", err)
	}
	defer func() { _ = page.Close() }()

	page = page.Context(ctx)

	if opt.Viewport != nil {
		scale := opt.Viewport.DeviceScaleFactor
		if scale == 0 {
			scale = 1
		}
		if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
			Width:             opt.Viewport.Width,
			Height:            opt.Viewport.Height,
			DeviceScaleFactor: scale,
		}); err != nil {
			return nil, fmt.Errorf("page.SetViewport: %w", err)
		}
	}

	if opt.StealthJS != "" {
		if _, err := page.EvalOnNewDocument(opt.StealthJS); err != nil {
			return nil, fmt.Errorf("page.EvalOnNewDocument: %w", err)
		}
	}

	if err := page.Navigate(href); err != nil {
		return nil, fmt.Errorf("page.Navigate: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("page.WaitLoad: %w", err)
	}

	finalURL := href
	if info, err := page.Info(); err == nil && info.URL != "" {
		finalURL = info.URL
		if u, err := url.Parse(finalURL); err == nil {
			code := func(s string) int {
				for _, e := range []string{"404", "403"} {
					if s == e {
						return 400 + int(e[2]-'0')
					}
				}
				return 0
			}
			for seg := range strings.SplitSeq(u.Path, "/") {
				if c := code(seg); c != 0 {
					return nil, &Error{Status: c, Href: href}
				}
			}
			for _, vals := range u.Query() {
				for _, v := range vals {
					if c := code(v); c != 0 {
						return nil, &Error{Status: c, Href: href}
					}
				}
			}
		}
	}

	status := 0
	if v, err := page.Eval(`() => { const e = performance.getEntriesByType("navigation")[0]; return e ? e.responseStatus : 0 }`); err == nil {
		status = v.Value.Int()
	}

	_ = page.WaitDOMStable(opt.IdleWait, 0.01)

	if opt.SettleJS != "" {
		settleCtx, settleCancel := context.WithTimeout(ctx, opt.IdleWait)
		_, _ = page.Context(settleCtx).Eval(opt.SettleJS)
		settleCancel()
	}

	initial, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("page.HTML: %w", err)
	}
	snapshots := []string{initial}

scrollLoop:
	for i := 0; i < opt.ScrollCount; i++ {
		delay := time.Duration(1+rand.IntN(3)) * time.Second
		select {
		case <-ctx.Done():
			break scrollLoop
		case <-time.After(delay):
		}
		_, _ = page.Eval(`() => new Promise(r => { const s = window.scrollY, e = document.documentElement.scrollHeight, d = 300, t0 = performance.now(); const step = t => { const k = Math.min((t - t0) / d, 1); window.scrollTo(0, s + (e - s) * k); k < 1 ? requestAnimationFrame(step) : r(); }; requestAnimationFrame(step); })`)
		_ = page.WaitDOMStable(opt.IdleWait, 0.01)
		snap, err := page.HTML()
		if err != nil {
			break scrollLoop
		}
		snapshots = append(snapshots, snap)
	}

	if opt.Type == TypeHTML {
		merged, err := Merge(snapshots)
		if err != nil {
			return nil, fmt.Errorf("Merge: %w", err)
		}
		htmlSrc, err := InlineTimeElements(merged)
		if err != nil {
			return nil, fmt.Errorf("InlineTimeElements: %w", err)
		}
		return &Result{
			Href:     href,
			FinalURL: finalURL,
			Content:  htmlSrc,
			Status:   status,
		}, nil
	}

	var firstArticle *readability.Article
	contentParts := make([]string, 0, len(snapshots))
	for _, snap := range snapshots {
		inlined, err := InlineTimeElements(snap)
		if err != nil {
			continue
		}
		article, err := readability.FromReader(strings.NewReader(inlined), parsed)
		if err != nil {
			continue
		}
		if firstArticle == nil {
			a := article
			firstArticle = &a
			lowTitle := strings.ToLower(strings.TrimSpace(article.Title))
			for _, pat := range []string{"just a moment", "attention required", "checking your browser", "access denied", "請稍候"} {
				if strings.Contains(lowTitle, pat) {
					return nil, &Error{Status: 403, Href: href}
				}
			}
		}
		if c := strings.TrimSpace(article.Content); c != "" {
			contentParts = append(contentParts, c)
		}
	}
	if firstArticle == nil {
		if status >= 400 {
			return nil, &Error{Status: status, Href: href}
		}
		return nil, fmt.Errorf("readability: no article extracted from %d snapshots", len(snapshots))
	}
	article := *firstArticle

	content := strings.Join(contentParts, "\n")
	if strings.TrimSpace(content) == "" {
		merged, _ := Merge(snapshots)
		inlined, _ := InlineTimeElements(merged)
		content = inlined
	}
	md, err := HTMLToMarkdown(content, href, opt.KeepLinks)
	if err != nil {
		return nil, fmt.Errorf("HTMLToMarkdown: %w", err)
	}
	md = DedupMarkdownParagraphs(md)
	if md == "" {
		return nil, &Error{Status: 204, Href: href}
	}
	if opt.MaxLength > 0 && len(md) > opt.MaxLength {
		md = md[:opt.MaxLength]
	}

	result := &Result{
		Href:     href,
		FinalURL: finalURL,
		Content:  md,
		Title:    article.Title,
		Author:   article.Byline,
		Excerpt:  article.Excerpt,
		Status:   status,
	}
	if article.PublishedTime != nil {
		result.PublishedAt = article.PublishedTime.Format(time.RFC3339)
	}

	if opt.Type == TypeJSON {
		tree, err := HTMLToNode(content, href, opt.KeepLinks)
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
			Href:     href,
			FinalURL: finalURL,
			Content:  string(b),
			Status:   status,
		}, nil
	}
	return result, nil
}
