package handlers

import (
	"testing"

	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
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

func TestComputeAdminStatsEmptyDatabase(t *testing.T) {
	stats, err := ComputeAdminStats(newTestDB(t), &config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Notably AverageFileBytes must not divide by a zero file count.
	if stats != (AdminStatsDTO{StorageBackend: "Filesystem"}) {
		t.Errorf("expected zeroed stats on an empty database, got %+v", stats)
	}
}

// Uploads are content-addressed, so two records can share one stored blob. These are the
// numbers the admin overview exists to surface.
func TestComputeAdminStatsDeduplicatesAndIgnoresDeleted(t *testing.T) {
	db := newTestDB(t)

	alice := models.User{AccountId: "aaaaaaaaaaaaaaaaaaaaaa"}
	bob := models.User{AccountId: "bbbbbbbbbbbbbbbbbbbbbb"}
	ghost := models.User{AccountId: "cccccccccccccccccccccc"} // owns no files
	for _, user := range []*models.User{&alice, &bob, &ghost} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("failed to seed user: %v", err)
		}
	}

	// Both users uploaded identical content: 2 records, 1 blob of 1000 bytes.
	seed := []models.UploadedFile{
		{FileId: "f1", FilePath: "sha-shared.bin", Size: 1000, OwnerID: alice.ID},
		{FileId: "f2", FilePath: "sha-shared.bin", Size: 1000, OwnerID: bob.ID},
		{FileId: "f3", FilePath: "sha-solo.bin", Size: 500, OwnerID: alice.ID},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("failed to seed file: %v", err)
		}
	}

	// A soft-deleted file, which must not appear in any total.
	deleted := models.UploadedFile{FileId: "f4", FilePath: "sha-gone.bin", Size: 99999, OwnerID: bob.ID}
	if err := db.Create(&deleted).Error; err != nil {
		t.Fatalf("failed to seed deleted file: %v", err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatalf("failed to soft-delete file: %v", err)
	}

	stats, err := ComputeAdminStats(db, &config.Config{S3Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := AdminStatsDTO{
		FileRecords:      3,    // the soft-deleted record is excluded
		UniqueFiles:      2,    // the shared blob counts once
		LogicalBytes:     2500, // 1000 + 1000 + 500
		StoredBytes:      1500, // one copy of the shared blob + the solo file
		DedupSavedBytes:  1000,
		TotalUsers:       3,
		UsersWithFiles:   2,    // ghost owns nothing
		AverageFileBytes: 833,  // 2500 / 3, integer division
		LargestFileBytes: 1000, // not the soft-deleted 99999
		StorageBackend:   "S3",
	}
	if stats != want {
		t.Errorf("stats mismatch\n got: %+v\nwant: %+v", stats, want)
	}
}
