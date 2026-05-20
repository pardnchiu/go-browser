//go:build darwin || linux

package goBrowser

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-rod/rod/lib/proto"
)

const (
	chromeCookieSalt         = "saltysalt"
	chromeCookieIterations   = 1003
	chromeCookieKeyLen       = 16
	chromeCookiePrefix       = "v10"
	chromePlaintextPrefixLen = 32
	webkitToUnixEpochSec     = 11644473600
	fieldSep                 = "\x1f"
)

var (
	chromeCookieIV = bytes.Repeat([]byte{' '}, aes.BlockSize)
)

func chromeSafeStoragePassword(ctx context.Context) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return chromeSafeStoragePasswordDarwin(ctx)
	case "linux":
		return chromeSafeStoragePasswordLinux(ctx)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func chromeSafeStoragePasswordDarwin(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "security",
		"find-generic-password", "-w",
		"-s", "Chrome Safe Storage",
		"-a", "Chrome",
	)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("security find-generic-password: %w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("security find-generic-password: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func chromeSafeStoragePasswordLinux(ctx context.Context) (string, error) {
	var lastErr error
	for _, app := range []string{"chrome", "chromium"} {
		cmd := exec.CommandContext(ctx, "secret-tool", "lookup", "application", app)
		out, err := cmd.Output()
		if err == nil {
			pw := strings.TrimSpace(string(out))
			if pw != "" {
				return pw, nil
			}
			lastErr = fmt.Errorf("secret-tool: empty password for application=%s", app)
			continue
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			lastErr = fmt.Errorf("secret-tool application=%s: %w: %s", app, err, strings.TrimSpace(string(ee.Stderr)))
		} else {
			lastErr = fmt.Errorf("secret-tool application=%s: %w", app, err)
		}
	}
	return "", lastErr
}

func deriveChromeCookieKey(password string) ([]byte, error) {
	return pbkdf2.Key(sha1.New, password, []byte(chromeCookieSalt), chromeCookieIterations, chromeCookieKeyLen)
}

func decryptChromeCookie(key, encrypted []byte) (string, error) {
	if len(encrypted) < len(chromeCookiePrefix) || string(encrypted[:3]) != chromeCookiePrefix {
		return string(encrypted), nil
	}
	body := encrypted[3:]
	if len(body) == 0 || len(body)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid ciphertext length: %d", len(body))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}
	plain := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, chromeCookieIV).CryptBlocks(plain, body)

	padLen := int(plain[len(plain)-1])
	if padLen > 0 && padLen <= aes.BlockSize && padLen <= len(plain) {
		valid := true
		for _, b := range plain[len(plain)-padLen:] {
			if int(b) != padLen {
				valid = false
				break
			}
		}
		if valid {
			plain = plain[:len(plain)-padLen]
		}
	}

	if len(plain) > chromePlaintextPrefixLen {
		plain = plain[chromePlaintextPrefixLen:]
	}
	return string(plain), nil
}

func extractChromeCookies(ctx context.Context, dbPath string) ([]*proto.NetworkCookieParam, error) {
	password, err := chromeSafeStoragePassword(ctx)
	if err != nil {
		return nil, err
	}
	key, err := deriveChromeCookieKey(password)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sqlite3",
		"-separator", fieldSep,
		dbPath,
		"SELECT host_key, name, value, hex(encrypted_value), path, expires_utc, is_secure, is_httponly, samesite FROM cookies",
	)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("sqlite3: %w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("sqlite3: %w", err)
	}

	var cookies []*proto.NetworkCookieParam
	for line := range strings.SplitSeq(string(out), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, fieldSep)
		if len(fields) < 9 {
			continue
		}
		host, name, plainVal, encHex, path := fields[0], fields[1], fields[2], fields[3], fields[4]
		expStr, secStr, httpStr, sameStr := fields[5], fields[6], fields[7], fields[8]

		value := plainVal
		if encHex != "" {
			encrypted, err := hex.DecodeString(encHex)
			if err != nil {
				continue
			}
			decoded, err := decryptChromeCookie(key, encrypted)
			if err != nil {
				continue
			}
			value = decoded
		}

		c := &proto.NetworkCookieParam{
			Name:   name,
			Value:  value,
			Domain: host,
			Path:   path,
		}
		if expUTC, err := strconv.ParseInt(expStr, 10, 64); err == nil && expUTC > 0 {
			c.Expires = proto.TimeSinceEpoch(float64(expUTC)/1e6 - webkitToUnixEpochSec)
		}
		c.Secure = secStr == "1"
		c.HTTPOnly = httpStr == "1"
		switch sameStr {
		case "0":
			c.SameSite = proto.NetworkCookieSameSiteNone
		case "1":
			c.SameSite = proto.NetworkCookieSameSiteLax
		case "2":
			c.SameSite = proto.NetworkCookieSameSiteStrict
		}
		cookies = append(cookies, c)
	}
	return cookies, nil
}
