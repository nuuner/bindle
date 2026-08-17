// Package unlock implements the shared secret that lifts the daily upload quota for a
// browser. A client posts the password once and gets back a cookie; every later request
// carries that cookie and the quota check lets it through.
package unlock

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/config"
)

const CookieName = "bindle_unlock"

// How long an issued cookie stays valid. The expiry is signed into the token, so a
// client cannot extend it by editing the cookie's own attributes.
const TokenLifetime = 30 * 24 * time.Hour

// Token layout: "<expiryUnix>.<hex HMAC-SHA256 of expiryUnix>". The MAC is keyed with
// the password itself, which gives two things for free: the raw password never sits in
// the cookie, and changing UNLOCK_PASSWORD invalidates every token handed out under the
// old one without the server having to remember which tokens it issued.
func sign(secret string, expiry int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(expiry, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

func IssueToken(secret string, expiresAt time.Time) string {
	expiry := expiresAt.Unix()
	return strconv.FormatInt(expiry, 10) + "." + sign(secret, expiry)
}

func TokenIsValid(secret, token string, now time.Time) bool {
	if secret == "" || token == "" {
		return false
	}

	expiryStr, signature, found := strings.Cut(token, ".")
	if !found {
		return false
	}

	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		return false
	}

	// Check the signature before the expiry: the expiry is client-supplied text until
	// the MAC proves it is the value this server signed.
	if subtle.ConstantTimeCompare([]byte(signature), []byte(sign(secret, expiry))) != 1 {
		return false
	}

	return now.Unix() < expiry
}

func PasswordMatches(secret, provided string) bool {
	if secret == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(secret), []byte(provided)) == 1
}

// IsUnlocked reports whether the request carries a cookie that lifts the upload quota.
// An empty UnlockPassword disables the feature outright, so a deployment that never sets
// one cannot be unlocked by any cookie.
func IsUnlocked(c *fiber.Ctx, cfg *config.Config) bool {
	return TokenIsValid(cfg.UnlockPassword, c.Cookies(CookieName), time.Now())
}

// cookieIsSecure marks the cookie https-only when the deployment is served over https.
// c.Protocol() reflects the proxy's forwarded scheme only when a trusted proxy sets it,
// so ClientOrigin is used as the fallback signal for how the site is actually reached.
func cookieIsSecure(c *fiber.Ctx, cfg *config.Config) bool {
	return c.Protocol() == "https" || strings.HasPrefix(cfg.ClientOrigin, "https://")
}

func SetCookie(c *fiber.Ctx, cfg *config.Config, expiresAt time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:  CookieName,
		Value: IssueToken(cfg.UnlockPassword, expiresAt),
		Path:  "/",
		// Expires rather than session-only: the point of the cookie is that unlocking
		// survives closing the tab.
		Expires: expiresAt,
		// The token is only ever read by the server, so script has no reason to touch it.
		HTTPOnly: true,
		Secure:   cookieIsSecure(c, cfg),
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func ClearCookie(c *fiber.Ctx, cfg *config.Config) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   cookieIsSecure(c, cfg),
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}
