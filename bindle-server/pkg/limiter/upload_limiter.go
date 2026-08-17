package limiter

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nuuner/bindle-server/internal/config"
	"github.com/nuuner/bindle-server/internal/models"
	"github.com/nuuner/bindle-server/pkg/unlock"
	"gorm.io/gorm"
)

func getAllConnectedAccountsAndIPs(db *gorm.DB, ipAddress string) ([]uint, []string, error) {
	var allAccountIDs = make(map[uint]bool)
	var allIPs = make(map[string]bool)

	// Start with the initial IP
	toCheckIPs := []string{ipAddress}
	allIPs[ipAddress] = true

	// Keep going until we find no new connections
	for len(toCheckIPs) > 0 {
		// Find all accounts for current IPs
		var ipConnections []models.AccountIpConnection
		if err := db.Where("ip_address IN ?", toCheckIPs).Find(&ipConnections).Error; err != nil {
			return nil, nil, err
		}

		toCheckIPs = nil // Clear for next iteration
		var toCheckAccounts []uint

		// Add new accounts to check
		for _, conn := range ipConnections {
			if !allAccountIDs[conn.AccountID] {
				allAccountIDs[conn.AccountID] = true
				toCheckAccounts = append(toCheckAccounts, conn.AccountID)
			}
		}

		// If we found new accounts, find their IPs
		if len(toCheckAccounts) > 0 {
			var newConnections []models.AccountIpConnection
			if err := db.Where("account_id IN ?", toCheckAccounts).Find(&newConnections).Error; err != nil {
				return nil, nil, err
			}

			// Add new IPs to check
			for _, conn := range newConnections {
				if !allIPs[conn.IPAddress] {
					allIPs[conn.IPAddress] = true
					toCheckIPs = append(toCheckIPs, conn.IPAddress)
				}
			}
		}
	}

	// Convert maps to slices
	accountIDs := make([]uint, 0, len(allAccountIDs))
	ips := make([]string, 0, len(allIPs))
	for id := range allAccountIDs {
		accountIDs = append(accountIDs, id)
	}
	for ip := range allIPs {
		ips = append(ips, ip)
	}

	return accountIDs, ips, nil
}

// getUsedBytes returns the bytes the given accounts have consumed against the daily
// quota: files completed in the last 24 hours, plus the sizes reserved by chunked
// upload sessions that are still in flight. In-flight sessions have no UploadedFile
// row yet, so counting only completed files would let a client open many sessions
// concurrently and have each one pass the check against the same stale total.
func getUsedBytes(db *gorm.DB, accountIDs []uint) (int64, error) {
	oneDayAgo := time.Now().Add(-24 * time.Hour)

	var completedSize int64
	err := db.Model(&models.UploadedFile{}).
		Where("owner_id IN ? AND created_at > ?", accountIDs, oneDayAgo).
		Select("COALESCE(SUM(size), 0)").
		Scan(&completedSize).Error
	if err != nil {
		return 0, err
	}

	var reservedSize int64
	err = db.Model(&models.UploadSession{}).
		Where("account_id IN ? AND status = ? AND expires_at > ?",
			accountIDs, models.UploadSessionStatusActive, time.Now()).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&reservedSize).Error
	if err != nil {
		return 0, err
	}

	return completedSize + reservedSize, nil
}

func ShouldThrottle(c *fiber.Ctx, db *gorm.DB, config *config.Config, fileSize int64) bool {
	// A client holding a valid unlock cookie has no daily quota at all. Checked here
	// rather than at each call site so every upload path is covered by construction.
	if unlock.IsUnlocked(c, config) {
		return false
	}

	ipAddress := c.IP()

	// Get all connected accounts and IPs
	accountIDs, _, err := getAllConnectedAccountsAndIPs(db, ipAddress)
	if err != nil {
		return true // If there's an error, throttle to be safe
	}

	// If no accounts found, allow (this would be their first upload)
	if len(accountIDs) == 0 {
		return false
	}

	usedSize, err := getUsedBytes(db, accountIDs)
	if err != nil {
		return true // If there's an error, throttle to be safe
	}

	// Check if total size exceeds the limit
	return usedSize+fileSize >= int64(config.UploadLimitMBPerDay*1000*1000)
}

func GetUploadedSizeForIP(db *gorm.DB, ipAddress string) (int64, error) {
	// Get all connected accounts and IPs
	accountIDs, _, err := getAllConnectedAccountsAndIPs(db, ipAddress)
	if err != nil {
		return 0, err
	}

	// If no accounts found, return 0
	if len(accountIDs) == 0 {
		return 0, nil
	}

	return getUsedBytes(db, accountIDs)
}
