//go:build !darwin && !linux

package browser

import (
	"context"
	"fmt"
	"runtime"

	"github.com/go-rod/rod/lib/proto"
)

// * 非 darwin/linux 不支援 SameSession，直接回傳錯誤
func extractChromeCookies(_ context.Context, _ string) ([]*proto.NetworkCookieParam, error) {
	return nil, fmt.Errorf("[Unavailable] same session: unsupported platform %s (only darwin/linux)", runtime.GOOS)
}
