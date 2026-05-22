package backup

import (
	"context"
	"os"
	"path/filepath"
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

	testFilePath := filepath.Join(testDir, "testfile.sql.gz")
	testFile, err := os.Create(testFilePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	testFile.Close()

	modTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(testFilePath, modTime, modTime); err != nil {
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
		if file != testFilePath {
			t.Errorf("Unexpected file in old files list: %s", file)
		}
	}
}

func TestVerifyFileExistsInCloud_Disabled(t *testing.T) {
	mockConfig := &config.CleanupConfig{
		MaxAgeDays:        30,
		VerifyCloudExists: false,
	}
	mockUploadConfig := &config.UploadConfig{
		Enabled:     true,
		RclonePath:  "/usr/bin/rclone",
		Destination: "remote:/backup",
	}
	mockLogger := logger.NewLogger("info")

	cleanupService := NewCleanupService(mockConfig, mockUploadConfig, mockLogger)

	testDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFilePath := filepath.Join(testDir, "testfile.sql.gz")
	f, err := os.Create(testFilePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	exists := cleanupService.verifyFileExistsInCloud(testFilePath, testDir)
	if exists {
		t.Errorf("Expected false when VerifyCloudExists is disabled, got true")
	}
}

func TestCleanupAgeBasedFiles(t *testing.T) {
	mockConfig := &config.CleanupConfig{
		MaxAgeDays:      30,
		AgeBasedCleanup: true,
		VerifyCloudExists: false,
	}
	mockUploadConfig := &config.UploadConfig{
		Enabled:     false,
		Destination: "remote:/backup",
	}
	mockLogger := logger.NewLogger("info")

	cleanupService := NewCleanupService(mockConfig, mockUploadConfig, mockLogger)

	testDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFilePath := filepath.Join(testDir, "testfile.sql.gz")
	f, err := os.Create(testFilePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	modTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(testFilePath, modTime, modTime); err != nil {
		t.Fatalf("Failed to set modification time for test file: %v", err)
	}

	err = cleanupService.CleanupAgeBasedFiles(context.Background(), testDir, []string{})
	if err != nil {
		t.Errorf("CleanupAgeBasedFiles returned an error: %v", err)
	}

	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected old file to be deleted, but it still exists")
	}
}
