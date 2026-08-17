package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/pkg/limiter"
	"github.com/nuuner/bindle-server/pkg/unlock"
	"gorm.io/gorm"
)

const unlockPassword = "letmein"

// unlockTestApp wires the unlock routes together with a route standing in for an upload,
// so a test can follow the cookie from the password all the way to the quota check.
func unlockTestApp(db *gorm.DB, cfg *config.Config) *fiber.App {
	app := fiber.New()
	app.Post("/api/unlock", func(c *fiber.Ctx) error {
		return UnlockLimits(c, cfg)
	})
	app.Delete("/api/unlock", func(c *fiber.Ctx) error {
		return LockLimits(c, cfg)
	})
	app.Post("/api/upload", func(c *fiber.Ctx) error {
		if limiter.ShouldThrottle(c, db, cfg, 1000) {
			return c.SendStatus(fiber.StatusTooManyRequests)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

// seedSpentQuota gives the requesting IP an account that has already used the whole
// daily allowance, so any upload is throttled unless something lifts the quota.
func seedSpentQuota(t *testing.T, db *gorm.DB, cfg *config.Config) {
	t.Helper()

	user := models.User{AccountId: "aaaaaaaaaaaaaaaaaaaaaa"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	spent := models.UploadedFile{FileId: "f1", FilePath: "a.bin",
		Size: cfg.UploadLimitMBPerDay * 1000 * 1000, OwnerID: user.ID}
	if err := db.Create(&spent).Error; err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}
	// fiber reports 0.0.0.0 as the client address for in-process test requests.
	if err := db.Create(&models.AccountIpConnection{AccountID: user.ID, IPAddress: "0.0.0.0"}).Error; err != nil {
		t.Fatalf("failed to seed IP connection: %v", err)
	}
}

func postPassword(t *testing.T, app *fiber.App, password string) *http.Response {
	t.Helper()

	req := httptest.NewRequest("POST", "/api/unlock", strings.NewReader(`{"password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("unlock request failed: %v", err)
	}
	return res
}

func unlockCookie(res *http.Response) *http.Cookie {
	for _, cookie := range res.Cookies() {
		if cookie.Name == unlock.CookieName {
			return cookie
		}
	}
	return nil
}

func upload(t *testing.T, app *fiber.App, cookie *http.Cookie) int {
	t.Helper()

	req := httptest.NewRequest("POST", "/api/upload", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	return res.StatusCode
}

// The whole feature in one pass: the quota bites, the password lifts it, and the cookie
// that does so is the one the unlock response handed out.
func TestUnlockCookieLiftsTheDailyQuota(t *testing.T) {
	cfg := &config.Config{UploadLimitMBPerDay: 10, UnlockPassword: unlockPassword}
	db := newTestDB(t)
	seedSpentQuota(t, db, cfg)
	app := unlockTestApp(db, cfg)

	if status := upload(t, app, nil); status != fiber.StatusTooManyRequests {
		t.Fatalf("expected a spent quota to throttle the upload, got %d", status)
	}

	res := postPassword(t, app, unlockPassword)
	if res.StatusCode != fiber.StatusOK {
		t.Fatalf("expected the correct password to be accepted, got %d", res.StatusCode)
	}

	cookie := unlockCookie(res)
	if cookie == nil {
		t.Fatal("expected an unlock cookie to be set")
	}
	if !cookie.HttpOnly {
		t.Error("expected the unlock cookie to be HttpOnly")
	}
	if cookie.Value == unlockPassword {
		t.Error("expected the cookie to carry a signed token rather than the password itself")
	}

	if status := upload(t, app, cookie); status != fiber.StatusOK {
		t.Errorf("expected the unlock cookie to lift the quota, got %d", status)
	}
}

func TestWrongPasswordIssuesNoCookie(t *testing.T) {
	cfg := &config.Config{UploadLimitMBPerDay: 10, UnlockPassword: unlockPassword}
	db := newTestDB(t)
	seedSpentQuota(t, db, cfg)
	app := unlockTestApp(db, cfg)

	res := postPassword(t, app, "not the password")
	if res.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401 for a wrong password, got %d", res.StatusCode)
	}
	if unlockCookie(res) != nil {
		t.Error("expected no cookie to be issued for a wrong password")
	}
}

// A deployment that never sets UNLOCK_PASSWORD has no unlock at all, so an empty password
// must not be accepted as a match for an empty secret.
func TestUnlockIsUnavailableWithoutAConfiguredPassword(t *testing.T) {
	cfg := &config.Config{UploadLimitMBPerDay: 10}
	db := newTestDB(t)
	seedSpentQuota(t, db, cfg)
	app := unlockTestApp(db, cfg)

	res := postPassword(t, app, "")
	if res.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected 403 when unlocking is not configured, got %d", res.StatusCode)
	}
	if unlockCookie(res) != nil {
		t.Error("expected no cookie when unlocking is not configured")
	}
}

func TestLockingRestoresTheQuota(t *testing.T) {
	cfg := &config.Config{UploadLimitMBPerDay: 10, UnlockPassword: unlockPassword}
	db := newTestDB(t)
	seedSpentQuota(t, db, cfg)
	app := unlockTestApp(db, cfg)

	cookie := unlockCookie(postPassword(t, app, unlockPassword))
	if cookie == nil {
		t.Fatal("expected an unlock cookie to be set")
	}

	req := httptest.NewRequest("DELETE", "/api/unlock", nil)
	req.AddCookie(cookie)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("lock request failed: %v", err)
	}

	cleared := unlockCookie(res)
	if cleared == nil || cleared.Value != "" {
		t.Fatal("expected locking to clear the unlock cookie")
	}
	if status := upload(t, app, cleared); status != fiber.StatusTooManyRequests {
		t.Errorf("expected the quota to apply again once locked, got %d", status)
	}
}
