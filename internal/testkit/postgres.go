// Package testkit provides shared helpers for integration tests.
package testkit

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const databaseDSNEnv = "TEST_DATABASE_DSN"

// OpenPostgres opens the PostgreSQL database explicitly configured for tests.
// Integration tests must not silently use a developer's default database.
func OpenPostgres(t testing.TB) *gorm.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv(databaseDSNEnv))
	if dsn == "" {
		t.Fatalf("%s must be set for PostgreSQL integration tests", databaseDSNEnv)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres with %s: %v", databaseDSNEnv, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("ping postgres with %s: %v", databaseDSNEnv, err)
	}

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}
