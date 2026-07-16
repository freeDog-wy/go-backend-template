package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesDotEnvWithoutOverridingProcessEnvironment(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	originalPassword, passwordExists := os.LookupEnv("EMAIL_SMTP_PASSWORD")
	t.Cleanup(func() {
		if passwordExists {
			_ = os.Setenv("EMAIL_SMTP_PASSWORD", originalPassword)
			return
		}
		_ = os.Unsetenv("EMAIL_SMTP_PASSWORD")
	})
	if err := os.WriteFile(".env", []byte("EMAIL_SMTP_PASSWORD=from-dotenv\nEMAIL_SMTP_HOST=smtp.local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("email:\n  smtp_password: from-yaml\n  smtp_host: yaml.local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EMAIL_SMTP_HOST", "from-process")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Email.SmtpPassword != "from-dotenv" {
		t.Fatalf("smtp password = %q, want from-dotenv", cfg.Email.SmtpPassword)
	}
	if cfg.Email.SmtpHost != "from-process" {
		t.Fatalf("smtp host = %q, want from-process", cfg.Email.SmtpHost)
	}
}

func TestLoadBindsS3Environment(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("storage:\n  s3:\n    endpoint: http://from-yaml\n    bucket: yaml-bucket\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STORAGE_S3_ENDPOINT", "http://from-environment")
	t.Setenv("STORAGE_S3_REGION", "us-east-1")
	t.Setenv("STORAGE_S3_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("STORAGE_S3_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("STORAGE_S3_BUCKET", "environment-bucket")
	t.Setenv("STORAGE_S3_USE_PATH_STYLE", "true")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.S3.Endpoint != "http://from-environment" || cfg.Storage.S3.Bucket != "environment-bucket" {
		t.Fatalf("S3 config = %#v, want environment overrides", cfg.Storage.S3)
	}
	if cfg.Storage.S3.Region != "us-east-1" || !cfg.Storage.S3.UsePathStyle {
		t.Fatalf("S3 config = %#v, want region and path-style binding", cfg.Storage.S3)
	}
}

func TestLoadDoesNotSearchInternalSourceDirectories(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	internalConfigDir := filepath.Join(tempDir, "internal", "config")
	if err := os.MkdirAll(internalConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(internalConfigDir, "config.yaml"), []byte("app:\n  mode: production\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load("config.yaml"); err == nil {
		t.Fatal("Load() error = nil, want no implicit internal/config lookup")
	}
}
