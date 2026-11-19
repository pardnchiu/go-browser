package browser

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	gorod "github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

//go:embed embed/challenge.js
var challengeJS string

//go:embed embed/challenge.json
var challengeJSON string

var challengeConfig = func() map[string]any {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(challengeJSON), &cfg); err != nil {
		panic(fmt.Sprintf("[Internal] embed/challenge.json: %v", err))
	}
	return cfg
}()

var ErrChallenge = errors.New("[Timeout] challenge: not completed by the user")

var ErrChallengeRequired = errors.New("[Denied] challenge: manual verification required")

var ErrChallengeClosed = errors.New("[Canceled] challenge: window closed by the user")

const (
	challengePoll    = time.Second
	challengeGrace   = 5 * time.Second
	challengeSettled = 2
	offscreenLimit   = -10000
	revealWindowLeft = 80
	revealWindowTop  = 80
	restoreTimeout   = 2 * time.Second
)

func hasChallenge(page *gorod.Page) (bool, error) {
	v, err := page.Eval(challengeJS, challengeConfig)
	if err != nil {
		return false, err
	}
	return v.Value.Bool(), nil
}

func targetClosed(page *gorod.Page) bool {
	ctx, cancel := context.WithTimeout(context.Background(), restoreTimeout)
	defer cancel()
	_, err := proto.TargetGetTargetInfo{TargetID: page.TargetID}.Call(page.Browser().Context(ctx))
	return err != nil
}

func revealWindow(page, base *gorod.Page) func() {
	win, err := proto.BrowserGetWindowForTarget{TargetID: page.TargetID}.Call(page)
	if err != nil || win.Bounds == nil || win.Bounds.Left == nil || *win.Bounds.Left > offscreenLimit {
		_, _ = page.Activate()
		return func() {}
	}
	left, top := revealWindowLeft, revealWindowTop
	_ = proto.BrowserSetWindowBounds{
		WindowID: win.WindowID,
		Bounds:   &proto.BrowserBounds{WindowState: proto.BrowserWindowStateNormal},
	}.Call(page)
	_ = proto.BrowserSetWindowBounds{
		WindowID: win.WindowID,
		Bounds:   &proto.BrowserBounds{Left: &left, Top: &top},
	}.Call(page)
	_, _ = page.Activate()
	prevLeft, prevTop := *win.Bounds.Left, 0
	if win.Bounds.Top != nil {
		prevTop = *win.Bounds.Top
	}
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), restoreTimeout)
		defer cancel()
		_ = proto.BrowserSetWindowBounds{
			WindowID: win.WindowID,
			Bounds:   &proto.BrowserBounds{Left: &prevLeft, Top: &prevTop},
		}.Call(base.Context(ctx))
	}
}

func waitChallenge(ctx context.Context, page *gorod.Page, href string, wait time.Duration) error {
	waitCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	p := page.Context(waitCtx)

	start := time.Now()
	restore := func() {}
	defer func() { restore() }()
	revealed := false
	settled := 0

	ticker := time.NewTicker(challengePoll)
	defer ticker.Stop()
	for {
		select {
		case <-waitCtx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%w: %s after %s", ErrChallenge, href, wait)
		case <-ticker.C:
		}

		pending, err := hasChallenge(p)
		if err != nil {
			if waitCtx.Err() == nil && targetClosed(page) {
				return fmt.Errorf("%w: %s", ErrChallengeClosed, href)
			}
			settled = 0
			continue
		}
		if !pending {
			if settled++; settled >= challengeSettled {
				return nil
			}
			continue
		}
		settled = 0
		if !revealed && time.Since(start) >= challengeGrace {
			restore = revealWindow(p, page)
			revealed = true
		}
	}
}
