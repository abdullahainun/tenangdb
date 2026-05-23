package database

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdullahainun/tenangdb/internal/config"
)

func newTestPostgresConfig() *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Type:     "postgresql",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Password: "testpass",
		Timeout:  30,
	}
}

func TestNewPostgreSQLClient(t *testing.T) {
	cfg := newTestPostgresConfig()
	client, err := NewPostgreSQLClient(cfg)
	if err != nil {
		t.Fatalf("NewPostgreSQLClient failed: %v", err)
	}
	if client.config != cfg {
		t.Fatal("config not stored correctly")
	}
}

func TestPostgreSQLClient_envVars(t *testing.T) {
	client, _ := NewPostgreSQLClient(newTestPostgresConfig())
	env := client.envVars()

	checkEnv := func(key, expected string) {
		for _, e := range env {
			if strings.HasPrefix(e, key+"=") {
				val := strings.TrimPrefix(e, key+"=")
				if val != expected {
					t.Errorf("%s = %q, want %q", key, val, expected)
				}
				return
			}
		}
		t.Errorf("%s not found in env vars", key)
	}

	checkEnv("PGHOST", "localhost")
	checkEnv("PGPORT", "5432")
	checkEnv("PGUSER", "testuser")
	checkEnv("PGPASSWORD", "testpass")
}

func TestPostgreSQLClient_Close(t *testing.T) {
	client, _ := NewPostgreSQLClient(newTestPostgresConfig())
	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestPostgreSQLClient_CreateDirectory(t *testing.T) {
	client, _ := NewPostgreSQLClient(newTestPostgresConfig())
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "nested", "dir")

	if err := client.CreateDirectory(testDir); err != nil {
		t.Fatalf("CreateDirectory failed: %v", err)
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatal("CreateDirectory did not create directory")
	}
}

func TestPostgreSQLClient_verifyDumpFile(t *testing.T) {
	client, _ := NewPostgreSQLClient(newTestPostgresConfig())
	tmpDir := t.TempDir()

	t.Run("file not found", func(t *testing.T) {
		err := client.verifyDumpFile(filepath.Join(tmpDir, "nonexistent.dump"))
		if err == nil {
			t.Fatal("expected error for non-existent file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		emptyFile := filepath.Join(tmpDir, "empty.dump")
		if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		err := client.verifyDumpFile(emptyFile)
		if err == nil {
			t.Fatal("expected error for empty file")
		}
	})

	t.Run("valid file", func(t *testing.T) {
		validFile := filepath.Join(tmpDir, "valid.dump")
		if err := os.WriteFile(validFile, []byte("dummy data"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := client.verifyDumpFile(validFile); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestPostgreSQLClient_CreateBackup(t *testing.T) {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		t.Skip("pg_dump not found, skipping integration test")
	}

	client, _ := NewPostgreSQLClient(newTestPostgresConfig())
	tmpDir := t.TempDir()

	_, err := client.CreateBackup(context.Background(), "nonexistent_db", tmpDir)
	if err == nil {
		t.Fatal("expected error for nonexistent database")
	}

	if !strings.Contains(err.Error(), "pg_dump failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestPostgreSQLClient_RestoreBackup(t *testing.T) {
	if _, err := exec.LookPath("pg_restore"); err != nil {
		t.Skip("pg_restore not found, skipping integration test")
	}

	client, _ := NewPostgreSQLClient(newTestPostgresConfig())

	err := client.RestoreBackup(context.Background(), "/nonexistent/backup.dump", "test_db")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	if !strings.Contains(err.Error(), "pg_restore failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestPostgreSQLClient_ListDatabases(t *testing.T) {
	if _, err := exec.LookPath("psql"); err != nil {
		t.Skip("psql not found, skipping integration test")
	}

	client, _ := NewPostgreSQLClient(newTestPostgresConfig())

	_, err := client.ListDatabases(context.Background())
	if err == nil {
		t.Fatal("expected error when connecting to nonexistent postgres")
	}

	if !strings.Contains(err.Error(), "psql failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
