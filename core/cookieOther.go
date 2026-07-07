//go:build !darwin && !linux

package goBrowser

import (
	"context"
	"fmt"
	"runtime"

	"github.com/go-rod/rod/lib/proto"
)

func extractChromeCookies(_ context.Context, _ string) ([]*proto.NetworkCookieParam, error) {
	return nil, fmt.Errorf("SameSession not supported on %s; only darwin and linux", runtime.GOOS)
}

