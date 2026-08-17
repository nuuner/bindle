package limiter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/pkg/unlock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Each connection gets its own private in-memory database.
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.UploadedFile{}, &models.User{},
		&models.AccountIpConnection{}, &models.UploadSession{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func seedUser(t *testing.T, db *gorm.DB, accountId string) models.User {
	t.Helper()
	user := models.User{AccountId: accountId}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
	return user
}

func TestGetUsedBytesCountsCompletedFiles(t *testing.T) {
	db := newTestDB(t)
	user := seedUser(t, db, "aaaaaaaaaaaaaaaaaaaaaa")

	seed := []models.UploadedFile{
		{FileId: "f1", FilePath: "a.bin", Size: 1000, OwnerID: user.ID},
		{FileId: "f2", FilePath: "b.bin", Size: 500, OwnerID: user.ID},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("failed to seed file: %v", err)
		}
	}

	used, err := getUsedBytes(db, []uint{user.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 1500 {
		t.Errorf("expected 1500 bytes used, got %d", used)
	}
}

// Files older than the window have rolled off the daily quota.
func TestGetUsedBytesIgnoresFilesOutsideWindow(t *testing.T) {
	db := newTestDB(t)
	user := seedUser(t, db, "aaaaaaaaaaaaaaaaaaaaaa")

	stale := models.UploadedFile{FileId: "f1", FilePath: "a.bin", Size: 1000, OwnerID: user.ID}
	if err := db.Create(&stale).Error; err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}
	if err := db.Model(&stale).Update("created_at", time.Now().Add(-25*time.Hour)).Error; err != nil {
		t.Fatalf("failed to backdate file: %v", err)
	}

	used, err := getUsedBytes(db, []uint{user.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 0 {
		t.Errorf("expected a file older than 24h to be excluded, got %d bytes", used)
	}
}

// The bypass this guards against: chunked sessions produce no UploadedFile row until
// they complete, so counting only completed files let a client open many sessions
// concurrently and have each pass the quota check against the same stale total.
func TestGetUsedBytesCountsInFlightSessions(t *testing.T) {
	db := newTestDB(t)
	user := seedUser(t, db, "aaaaaaaaaaaaaaaaaaaaaa")

	sessions := []models.UploadSession{
		{SessionID: "s1", AccountID: user.ID, FileSize: 900, Status: models.UploadSessionStatusActive,
			ExpiresAt: time.Now().Add(time.Hour)},
		{SessionID: "s2", AccountID: user.ID, FileSize: 900, Status: models.UploadSessionStatusActive,
			ExpiresAt: time.Now().Add(time.Hour)},
	}
	for i := range sessions {
		if err := db.Create(&sessions[i]).Error; err != nil {
			t.Fatalf("failed to seed session: %v", err)
		}
	}

	used, err := getUsedBytes(db, []uint{user.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 1800 {
		t.Errorf("expected both in-flight sessions to reserve quota (1800), got %d", used)
	}
}

// A completed session has become an UploadedFile row, so counting it again would
// charge the same upload twice.
func TestGetUsedBytesIgnoresSettledAndExpiredSessions(t *testing.T) {
	db := newTestDB(t)
	user := seedUser(t, db, "aaaaaaaaaaaaaaaaaaaaaa")

	sessions := []models.UploadSession{
		{SessionID: "s1", AccountID: user.ID, FileSize: 900, Status: models.UploadSessionStatusCompleted,
			ExpiresAt: time.Now().Add(time.Hour)},
		{SessionID: "s2", AccountID: user.ID, FileSize: 900, Status: models.UploadSessionStatusCancelled,
			ExpiresAt: time.Now().Add(time.Hour)},
		{SessionID: "s3", AccountID: user.ID, FileSize: 900, Status: models.UploadSessionStatusActive,
			ExpiresAt: time.Now().Add(-time.Hour)}, // abandoned, past its expiry
	}
	for i := range sessions {
		if err := db.Create(&sessions[i]).Error; err != nil {
			t.Fatalf("failed to seed session: %v", err)
		}
	}

	used, err := getUsedBytes(db, []uint{user.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 0 {
		t.Errorf("expected settled and expired sessions to reserve nothing, got %d bytes", used)
	}
}

// Accounts are linked transitively through shared IPs, and the quota follows the
// whole linked set rather than the single account making the request.
func TestGetUsedBytesSumsAcrossLinkedAccounts(t *testing.T) {
	db := newTestDB(t)
	alice := seedUser(t, db, "aaaaaaaaaaaaaaaaaaaaaa")
	bob := seedUser(t, db, "bbbbbbbbbbbbbbbbbbbbbb")

	seed := []models.UploadedFile{
		{FileId: "f1", FilePath: "a.bin", Size: 1000, OwnerID: alice.ID},
		{FileId: "f2", FilePath: "b.bin", Size: 250, OwnerID: bob.ID},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("failed to seed file: %v", err)
		}
	}

	used, err := getUsedBytes(db, []uint{alice.ID, bob.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if used != 1250 {
		t.Errorf("expected 1250 bytes across both accounts, got %d", used)
	}
}

// throttleWithCookie runs ShouldThrottle against a request carrying the given cookies,
// for an account that has already used up the whole daily quota.
func throttleWithCookie(t *testing.T, cfg *config.Config, cookie *fiber.Cookie) bool {
	t.Helper()

	db := newTestDB(t)
	user := seedUser(t, db, "aaaaaaaaaaaaaaaaaaaaaa")
	spent := models.UploadedFile{FileId: "f1", FilePath: "a.bin",
		Size: cfg.UploadLimitMBPerDay * 1000 * 1000, OwnerID: user.ID}
	if err := db.Create(&spent).Error; err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	var throttled bool
	app := fiber.New()
	app.Post("/upload", func(c *fiber.Ctx) error {
		// The quota follows the accounts linked to the request's IP, which is only known
		// once the request is in flight.
		if err := db.Create(&models.AccountIpConnection{AccountID: user.ID, IPAddress: c.IP()}).Error; err != nil {
			t.Errorf("failed to seed IP connection: %v", err)
		}
		throttled = ShouldThrottle(c, db, cfg, 1000)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("POST", "/upload", nil)
	if cookie != nil {
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
	}
	if _, err := app.Test(req); err != nil {
		t.Fatalf("test request failed: %v", err)
	}

	return throttled
}

func TestShouldThrottleWhenQuotaIsSpent(t *testing.T) {
	cfg := &config.Config{UploadLimitMBPerDay: 10, UnlockPassword: "s3cret"}

	if !throttleWithCookie(t, cfg, nil) {
		t.Error("expected an account over its daily quota to be throttled")
	}
}

func TestUnlockCookieLiftsTheQuota(t *testing.T) {
	cfg := &config.Config{UploadLimitMBPerDay: 10, UnlockPassword: "s3cret"}
	valid := &fiber.Cookie{
		Name:  unlock.CookieName,
		Value: unlock.IssueToken(cfg.UnlockPassword, time.Now().Add(time.Hour)),
	}

	if throttleWithCookie(t, cfg, valid) {
		t.Error("expected a valid unlock cookie to lift the daily quota")
	}
}

func TestForgedUnlockCookieDoesNotLiftTheQuota(t *testing.T) {
	cfg := &config.Config{UploadLimitMBPerDay: 10, UnlockPassword: "s3cret"}
	forged := &fiber.Cookie{
		Name:  unlock.CookieName,
		Value: unlock.IssueToken("guessed password", time.Now().Add(time.Hour)),
	}

	if !throttleWithCookie(t, cfg, forged) {
		t.Error("expected a cookie signed with the wrong password to be ignored")
	}
}
