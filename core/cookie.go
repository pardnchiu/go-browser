//go:build darwin || linux

package browser

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/go-rod/rod/lib/proto"
	goSqlkit "github.com/pardnchiu/go-sqlkit"
	sqlkitCore "github.com/pardnchiu/go-sqlkit/core"
)

const (
	cookieSalt           = "saltysalt" // PBKDF2 salt 固定為 "saltysalt"
	cookieKeyLen         = 16
	prefixLen            = 32          // cookies db version ≥ 24 的前綴為 SHA-256(host_key)，要裁剪掉
	webkitToUnixEpochSec = 11644473600 // chrome 時間是從 1601-01-01 開始計算，要補上偏移
)

var (
	cookieIV = bytes.Repeat([]byte{' '}, aes.BlockSize) // chrome IV 固定為 16 bytes 空白
)

type cookieRaw struct {
	host      string
	name      string
	encrypted []byte
	path      string
	expUTC    int64
	secure    int
	httpOnly  int
	sameSite  int
	plainVal  string
}

type v11sample struct {
	enc  []byte
	host string
}

func extractChromeCookies(ctx context.Context, dbPath string) ([]*proto.NetworkCookieParam, error) {
	conn, err := goSqlkit.New(sqlkitCore.Config{
		Target:       sqlkitCore.SQLite,
		Path:         dbPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("[Unavailable] cookies db: %w", err)
	}
	defer conn.Close()

	rows, err := conn.QueryContext(ctx, `
	SELECT host_key,
			name,
			value,
			encrypted_value,
			path,
			expires_utc,
			is_secure,
			is_httponly,
			samesite
	FROM cookies`)
	if err != nil {
		return nil, fmt.Errorf("[Upstream] cookies db: query: %w", err)
	}
	defer rows.Close()

	var raws []cookieRaw
	var sample *v11sample
	for rows.Next() {
		var row cookieRaw
		if err := rows.Scan(&row.host, &row.name, &row.plainVal, &row.encrypted, &row.path, &row.expUTC, &row.secure, &row.httpOnly, &row.sameSite); err != nil {
			return nil, fmt.Errorf("[Upstream] cookies db: scan: %w", err)
		}
		// * 選一把 v11 作為樣本解析，用於比對後續 keyring 密碼是否正確
		// v10 的 cookie 都是明文或是固定密碼 peanuts
		// 但 linux v11 的 cookie 會有不同的 domain-hash 前綴，所以需要一個樣本來驗證 keyring 正確性
		if sample == nil && len(row.encrypted) >= 3 && string(row.encrypted[:3]) == "v11" {
			sample = &v11sample{enc: row.encrypted, host: row.host}
		}
		raws = append(raws, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("[Upstream] cookies db: rows: %w", err)
	}

	keys, err := cookieKeys(ctx, sample)
	if err != nil {
		return nil, err
	}

	cookies := make([]*proto.NetworkCookieParam, 0, len(raws))
	for _, raw := range raws {
		value := raw.plainVal
		if len(raw.encrypted) > 0 {
			decoded, ok := decryptChromeCookie(keys, raw.encrypted)
			if !ok {
				continue
			}
			value = decoded
		}

		cookie := &proto.NetworkCookieParam{
			Name:   raw.name,
			Value:  value,
			Domain: raw.host,
			Path:   raw.path,
		}
		if raw.expUTC > 0 {
			cookie.Expires = proto.TimeSinceEpoch(float64(raw.expUTC)/1e6 - webkitToUnixEpochSec)
		}

		cookie.Secure = raw.secure == 1
		cookie.HTTPOnly = raw.httpOnly == 1

		if raw.sameSite == 0 {
			cookie.SameSite = proto.NetworkCookieSameSiteNone
		} else if raw.sameSite == 1 {
			cookie.SameSite = proto.NetworkCookieSameSiteLax
		} else if raw.sameSite == 2 {
			cookie.SameSite = proto.NetworkCookieSameSiteStrict
		} else {
			cookie.SameSite = proto.NetworkCookieSameSiteNone
		}
		cookies = append(cookies, cookie)
	}
	return cookies, nil
}

func cookieKeys(ctx context.Context, sample *v11sample) (map[string][]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		passwd, err := darwinKeychainPassword(ctx)
		if err != nil {
			return nil, fmt.Errorf("[Denied] chrome safe storage: %w", err)
		}

		// v10 用 Keychain 密碼，迭代 1003
		v10, _ := pbkdf2.Key(sha1.New, passwd, []byte(cookieSalt), 1003, cookieKeyLen)
		return map[string][]byte{"v10": v10}, nil

	case "linux":
		// v10 用固定值 peanuts，迭代為 1
		v10, _ := pbkdf2.Key(sha1.New, "peanuts", []byte(cookieSalt), 1, cookieKeyLen)
		keys := map[string][]byte{"v10": v10}

		if sample == nil {
			return keys, nil
		}
		passwds, errs := linuxKeyringPasswords(ctx)
		for _, psswd := range passwds {
			// v11 用 keyring 密碼，迭代為 1
			key, _ := pbkdf2.Key(sha1.New, psswd, []byte(cookieSalt), 1, cookieKeyLen)
			if isv11KeyMatch(key, *sample) {
				keys["v11"] = key
				return keys, nil
			}
		}
		if len(passwds) == 0 {
			return nil, fmt.Errorf("[Denied] chrome safe storage: no v11 keyring password (gnome-keyring/kwallet): %w", errors.Join(errs...))
		}
		return nil, fmt.Errorf("[Denied] chrome safe storage: %d keyring password(s) failed to decrypt v11 cookies", len(passwds))

	default:
		return nil, fmt.Errorf("[Unavailable] chrome safe storage: unsupported platform %s", runtime.GOOS)
	}
}

func darwinKeychainPassword(ctx context.Context) (string, error) {
	// * macOS 透過 Keychain 存在 Chrome Safe Storage
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

func linuxKeyringPasswords(ctx context.Context) ([]string, []error) {
	var gnomePasswds, kwalletPasswds []string
	var gnomeErrs, kwalletErrs []error
	var wg sync.WaitGroup

	wg.Add(2)
	// * GNOME／MATE／XFCE／Unity 透過 secret-tool 存在 gnome-keyring
	go func() {
		defer wg.Done()
		for _, app := range []string{"chrome", "chromium"} {
			if passwd, err := secretToolLookup(ctx, app); err == nil {
				gnomePasswds = append(gnomePasswds, passwd)
			} else {
				gnomeErrs = append(gnomeErrs, err)
			}
		}
	}()
	// * KDE 透過 KWallet 存在 kdewallet
	go func() {
		defer wg.Done()
		for _, e := range []struct{ folder, key string }{
			{"Chrome Keys", "Chrome Safe Storage"},
			{"Chromium Keys", "Chromium Safe Storage"},
		} {
			if passwd, err := kwalletRead(ctx, e.folder, e.key); err == nil {
				kwalletPasswds = append(kwalletPasswds, passwd)
			} else {
				kwalletErrs = append(kwalletErrs, err)
			}
		}
	}()
	wg.Wait()

	var passwds []string
	dic := map[string]bool{}
	for _, pw := range append(gnomePasswds, kwalletPasswds...) {
		if pw != "" && !dic[pw] {
			dic[pw] = true
			passwds = append(passwds, pw)
		}
	}
	return passwds, append(gnomeErrs, kwalletErrs...)
}

func secretToolLookup(ctx context.Context, app string) (string, error) {
	cmd := exec.CommandContext(ctx, "secret-tool", "lookup", "application", app)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("secret-tool application=%s: %w: %s", app, err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("secret-tool application=%s: %w", app, err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func kwalletRead(ctx context.Context, folder, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "kwallet-query", "kdewallet", "-f", folder, "-r", key)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", fmt.Errorf("kwallet-query folder=%s: %w: %s", folder, err, strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("kwallet-query folder=%s: %w", folder, err)
	}
	pw := strings.TrimRight(string(out), "\n")
	if pw == "" || pw == "Failed to read entry" {
		return "", fmt.Errorf("kwallet-query folder=%s: empty entry %q", folder, key)
	}
	return pw, nil
}

func isv11KeyMatch(key []byte, s v11sample) bool {
	plain, ok := aesDecryptVerifyCBC(key, s.enc[3:])
	if !ok || len(plain) < prefixLen {
		return false
	}
	sum := sha256.Sum256([]byte(s.host))
	return bytes.Equal(plain[:prefixLen], sum[:])
}

func aesDecryptVerifyCBC(key, body []byte) ([]byte, bool) {
	if len(body) == 0 || len(body)%aes.BlockSize != 0 {
		return nil, false
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}

	plain := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, cookieIV).CryptBlocks(plain, body)

	padLen := int(plain[len(plain)-1])
	if padLen <= 0 || padLen > aes.BlockSize || padLen > len(plain) {
		return nil, false
	}
	for _, b := range plain[len(plain)-padLen:] {
		if int(b) != padLen {
			return nil, false
		}
	}
	return plain[:len(plain)-padLen], true
}

func decryptChromeCookie(keys map[string][]byte, encrypted []byte) (string, bool) {
	if len(encrypted) < 3 {
		return string(encrypted), true
	}

	prefix := string(encrypted[:3])
	key, ok := keys[prefix]
	if !ok {
		return string(encrypted), true
	}

	plain, ok := aesDecryptVerifyCBC(key, encrypted[3:])
	if !ok {
		return "", false
	}
	if len(plain) >= prefixLen {
		plain = plain[prefixLen:]
	}
	return string(plain), true
}
