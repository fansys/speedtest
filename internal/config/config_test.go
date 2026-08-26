package config

import (
	"os"
	"testing"
)

func TestLoadSettings(t *testing.T) {
	// 设置测试所需环境变量
	os.Setenv("ADMIN_TOKEN", "admin-secret-token-1234567890")
	os.Setenv("REGISTRATION_TOKEN", "reg-secret-token-1234567890")
	os.Setenv("SECRET_KEY", "master-secret-key-1234567890123")
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_URL", "sqlite:./test_db.sqlite3")

	s, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if s.Port != 9090 {
		t.Fatalf("expected Port 9090, got %d", s.Port)
	}
	if s.DatabasePath() != "./test_db.sqlite3" {
		t.Fatalf("expected DatabasePath './test_db.sqlite3', got %q", s.DatabasePath())
	}
	if len(s.SecretValues()) != 3 {
		t.Fatalf("expected 3 secret values, got %d", len(s.SecretValues()))
	}
}

func TestLoadSettingsValidationFails(t *testing.T) {
	// 相同 token 应当校验失败
	os.Setenv("ADMIN_TOKEN", "same-token-1234567890")
	os.Setenv("REGISTRATION_TOKEN", "same-token-1234567890")
	os.Setenv("SECRET_KEY", "master-secret-key-1234567890123")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when ADMIN_TOKEN == REGISTRATION_TOKEN")
	}

	// token 太短应当校验失败
	os.Setenv("ADMIN_TOKEN", "short")
	os.Setenv("REGISTRATION_TOKEN", "reg-secret-token-1234567890")
	_, err = Load()
	if err == nil {
		t.Fatal("expected error when ADMIN_TOKEN is too short")
	}
}
