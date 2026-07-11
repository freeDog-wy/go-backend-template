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

	cfg := Load(configPath)
	if cfg.Email.SmtpPassword != "from-dotenv" {
		t.Fatalf("smtp password = %q, want from-dotenv", cfg.Email.SmtpPassword)
	}
	if cfg.Email.SmtpHost != "from-process" {
		t.Fatalf("smtp host = %q, want from-process", cfg.Email.SmtpHost)
	}
}
