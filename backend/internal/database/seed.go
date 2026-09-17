package database

import (
	"fmt"

	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedServiceAccount upserts an MQTT service account (e.g. the backend's own
// broker identity). The raw password is bcrypt-hashed; the broker verifies
// against this hash. Idempotent: safe to call on every startup.
func SeedServiceAccount(db *gorm.DB, username, rawPassword string, superuser bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash service account password: %w", err)
	}

	account := domain.ServiceAccount{
		Username:     username,
		PasswordHash: string(hash),
		Superuser:    superuser,
	}

	// On conflict, refresh the hash so a changed password propagates.
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "username"}},
		DoUpdates: clause.AssignmentColumns([]string{"password_hash", "superuser", "updated_at"}),
	}).Create(&account).Error
}
