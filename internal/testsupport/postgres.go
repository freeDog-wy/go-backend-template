// Package testsupport provides shared helpers for integration tests.
package testsupport

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/pkg/envfile"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const databaseDSNEnv = "TEST_DATABASE_DSN"

// OpenPostgres opens the PostgreSQL database explicitly configured for tests.
// Integration tests must not silently use a developer's default database.
func OpenPostgres(t testing.TB) *gorm.DB {
	t.Helper()
	if err := envfile.LoadNearest(".env"); err != nil {
		t.Fatalf("load nearest .env: %v", err)
	}

	dsn := strings.TrimSpace(os.Getenv(databaseDSNEnv))
	if dsn == "" {
		t.Fatalf("%s must be set for PostgreSQL integration tests", databaseDSNEnv)
	}

	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres with %s: %v", databaseDSNEnv, err)
	}

	adminSQLDB, err := adminDB.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := adminSQLDB.PingContext(ctx); err != nil {
		_ = adminSQLDB.Close()
		t.Fatalf("ping postgres with %s: %v", databaseDSNEnv, err)
	}

	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		_ = adminSQLDB.Close()
		t.Fatalf("create test schema: %v", err)
	}

	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		_ = adminDB.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
		_ = adminSQLDB.Close()
		t.Fatalf("open test schema: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		_ = adminDB.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
		_ = adminSQLDB.Close()
		t.Fatalf("get test schema database handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		_ = adminDB.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
		_ = adminSQLDB.Close()
	})
	return db
}
