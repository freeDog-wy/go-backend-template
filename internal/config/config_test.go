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

func TestLoadValidatesCORSOrigins(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	t.Run("normalizes and accepts the configured admin origin", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "valid.yaml")
		content := "app:\n  mode: production\nserver:\n  cors_allowed_origins:\n    - https://admin.example.test/\nauth:\n  admin_origin: https://admin.example.test\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Auth.AdminOrigin != "https://admin.example.test" || len(cfg.Server.CORSAllowedOrigins) != 1 || cfg.Server.CORSAllowedOrigins[0] != "https://admin.example.test" {
			t.Fatalf("CORS config = %#v, auth config = %#v", cfg.Server.CORSAllowedOrigins, cfg.Auth)
		}
	})

	t.Run("rejects an admin origin outside the allowlist", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "mismatch.yaml")
		content := "server:\n  cors_allowed_origins:\n    - http://admin.example.test\nauth:\n  admin_origin: http://other.example.test\n"
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(configPath); err == nil {
			t.Fatal("Load() error = nil, want CORS allowlist mismatch")
		}
	})

	t.Run("reads a comma-separated CORS allowlist from the environment", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "environment.yaml")
		if err := os.WriteFile(configPath, []byte("{}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SERVER_CORS_ALLOWED_ORIGINS", "https://admin.example.test, https://preview.example.test/")
		cfg, err := Load(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.Server.CORSAllowedOrigins; len(got) != 2 || got[0] != "https://admin.example.test" || got[1] != "https://preview.example.test" {
			t.Fatalf("CORS origins = %#v", got)
		}
	})
}
