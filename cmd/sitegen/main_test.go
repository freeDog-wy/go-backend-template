package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvUsesFileWithoutOverridingProcessEnvironment(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	restoreEnv(t, "SITEGEN_SITE_URL")
	t.Setenv("SITEGEN_CMS_API_BASE_URL", "https://process.example.test")
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte("SITEGEN_CMS_API_BASE_URL=https://dotenv.example.test\nSITEGEN_SITE_URL=http://localhost:9999\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(); err != nil {
		t.Fatal(err)
	}
	if got := envOr("SITEGEN_CMS_API_BASE_URL", ""); got != "https://process.example.test" {
		t.Fatalf("CMS API base URL = %q, want process environment value", got)
	}
	if got := envOr("SITEGEN_SITE_URL", ""); got != "http://localhost:9999" {
		t.Fatalf("site URL = %q, want .env value", got)
	}
}

func restoreEnv(t *testing.T, key string) {
	t.Helper()
	original, exists := os.LookupEnv(key)
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(key, original)
			return
		}
		_ = os.Unsetenv(key)
	})
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}
