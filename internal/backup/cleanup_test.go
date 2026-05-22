package backup

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
)

func TestGetOldFiles(t *testing.T) {
	mockConfig := &config.CleanupConfig{
		MaxAgeDays: 30,
	}
	mockUploadConfig := &config.UploadConfig{
		Enabled:          true,
		RclonePath:       "/usr/bin/rclone",
		Destination:      "remote:/backup",
		RcloneConfigPath: "",
	}
	mockLogger := logger.NewLogger("info")

	cleanupService := NewCleanupService(mockConfig, mockUploadConfig, mockLogger)

	testDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile, err := os.CreateTemp(testDir, "testfile.sql.gz")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile.Name())

	modTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(testFile.Name(), modTime, modTime); err != nil {
		t.Fatalf("Failed to set modification time for test file: %v", err)
	}

	oldFiles, err := cleanupService.GetOldFiles(testDir, 30)
	if err != nil {
		t.Errorf("GetOldFiles returned an error: %v", err)
	}
	if len(oldFiles) == 0 {
		t.Errorf("Expected at least one old file, got none")
	}

	for _, file := range oldFiles {
		if file != testFile.Name() {
			t.Errorf("Unexpected file in old files list: %s", file)
		}
	}
}

func TestVerifyFileExistsInCloud(t *testing.T) {
	mockConfig := &config.CleanupConfig{
		MaxAgeDays: 30,
		VerifyCloudExists: true,
	}
	mockUploadConfig := &config.UploadConfig{
		Enabled:          true,
		RclonePath:       "/usr/bin/rclone",
		Destination:      "remote:/backup",
		RcloneConfigPath: "",
	}
	mockLogger := logger.NewLogger("info")

	cleanupService := NewCleanupService(mockConfig, mockUploadConfig, mockLogger)

	testDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile, err := os.CreateTemp(testDir, "testfile.sql.gz")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile.Name())

	// Mock rclone output
	rcloneOutput := []byte("remote:/backup/testfile.sql.gz")

	// Mock exec.Command to return mock output
	_ = func(command string, args ...string) *exec.Cmd {
		return exec.CommandContext(context.Background(), "sh", "-c", "echo "+string(rcloneOutput))
	}

	cleanupService.uploadConfig.RclonePath = "/mock/rclone"
	cleanupService.uploadConfig.Enabled = true

	exists := cleanupService.verifyFileExistsInCloud(testFile.Name(), testDir)
	if !exists {
		t.Errorf("Expected file to exist in cloud, got false")
	}
}

func TestCleanupAgeBasedFiles(t *testing.T) {
	mockConfig := &config.CleanupConfig{
		MaxAgeDays: 30,
		VerifyCloudExists: true,
		AgeBasedCleanup: true,
	}
	mockUploadConfig := &config.UploadConfig{
		Enabled:          true,
		RclonePath:       "/usr/bin/rclone",
		Destination:      "remote:/backup",
		RcloneConfigPath: "",
	}
	mockLogger := logger.NewLogger("info")

	cleanupService := NewCleanupService(mockConfig, mockUploadConfig, mockLogger)

	testDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile, err := os.CreateTemp(testDir, "testfile.sql.gz")
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile.Name())

	modTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(testFile.Name(), modTime, modTime); err != nil {
		t.Fatalf("Failed to set modification time for test file: %v", err)
	}

	err = cleanupService.CleanupAgeBasedFiles(context.Background(), testDir, []string{"database_name"})
	if err != nil {
		t.Errorf("CleanupAgeBasedFiles returned an error: %v", err)
	}
}
