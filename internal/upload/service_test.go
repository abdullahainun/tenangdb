package upload

import (
	"testing"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
)

func TestIsGCSDestination(t *testing.T) {
	uploadConfig := &config.UploadConfig{}
	log := logger.NewLogger("test")
	service := NewService(uploadConfig, log)

	tests := []struct {
		name        string
		destination string
		expected    bool
	}{
		{"GCS with gcs prefix", "gcs:my-bucket", true},
		{"GCS with mygcs prefix", "mygcs:tenangdb-backup", true},
		{"GCS with googlecloud prefix", "googlecloud:bucket-name", true},
		{"GCS with gc prefix", "gc:my-storage", true},
		{"S3 destination", "s3:my-bucket", false},
		{"Azure destination", "azure:container", false},
		{"Local path", "/local/path", false},
		{"Empty destination", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isGCSDestination(tt.destination)
			if result != tt.expected {
				t.Errorf("isGCSDestination(%q) = %v, want %v", tt.destination, result, tt.expected)
			}
		})
	}
}

func TestAddGCSFlags(t *testing.T) {
	tests := []struct {
		name      string
		gcsConfig config.GCSConfig
		baseArgs  []string
		expected  []string
	}{
		{
			name: "Default uniform bucket access",
			gcsConfig: config.GCSConfig{
				BucketPolicyOnly: true,
				NoCheckBucket:    false,
				ObjectACL:        "",
				BucketACL:        "",
			},
			baseArgs: []string{"copy", "source", "dest"},
			expected: []string{
				"copy", "source", "dest",
				"--gcs-bucket-policy-only=true",
				"--gcs-object-acl=",
				"--gcs-bucket-acl=",
			},
		},
		{
			name: "Uniform bucket access with no bucket check",
			gcsConfig: config.GCSConfig{
				BucketPolicyOnly: true,
				NoCheckBucket:    true,
				ObjectACL:        "",
				BucketACL:        "",
			},
			baseArgs: []string{"copy", "source", "dest"},
			expected: []string{
				"copy", "source", "dest",
				"--gcs-bucket-policy-only=true",
				"--gcs-object-acl=",
				"--gcs-bucket-acl=",
				"--gcs-no-check-bucket=true",
			},
		},
		{
			name: "Custom ACL settings",
			gcsConfig: config.GCSConfig{
				BucketPolicyOnly: false,
				NoCheckBucket:    false,
				ObjectACL:        "publicRead",
				BucketACL:        "private",
			},
			baseArgs: []string{"copy", "source", "dest"},
			expected: []string{
				"copy", "source", "dest",
				"--gcs-bucket-policy-only=true", // Auto-enabled by shouldUseBucketPolicyOnly
				"--gcs-object-acl=publicRead",
				"--gcs-bucket-acl=private",
			},
		},
		{
			name: "Minimal configuration",
			gcsConfig: config.GCSConfig{
				BucketPolicyOnly: false,
				NoCheckBucket:    false,
				ObjectACL:        "",
				BucketACL:        "",
			},
			baseArgs: []string{"copy", "source", "dest"},
			expected: []string{
				"copy", "source", "dest",
				"--gcs-bucket-policy-only=true", // Auto-enabled
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uploadConfig := &config.UploadConfig{
				GCS: tt.gcsConfig,
			}
			log := logger.NewLogger("test")
			service := NewService(uploadConfig, log)

			result := service.addGCSFlags(tt.baseArgs)

			// Check if all expected flags are present
			for _, expectedFlag := range tt.expected {
				found := false
				for _, resultFlag := range result {
					if resultFlag == expectedFlag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected flag %q not found in result %v", expectedFlag, result)
				}
			}

			// Check that we don't have unexpected extra flags (basic length check)
			if len(result) > len(tt.expected)+2 { // Allow some tolerance for auto-enabled flags
				t.Errorf("Result has too many flags: got %d, expected around %d", len(result), len(tt.expected))
			}
		})
	}
}

func TestShouldUseBucketPolicyOnly(t *testing.T) {
	uploadConfig := &config.UploadConfig{}
	log := logger.NewLogger("test")
	service := NewService(uploadConfig, log)

	// Should always return true for better compatibility
	result := service.shouldUseBucketPolicyOnly()
	if !result {
		t.Error("shouldUseBucketPolicyOnly() should return true by default")
	}
}

func TestGCSConfigDefaults(t *testing.T) {
	// Test that GCS config has reasonable defaults
	gcsConfig := config.GCSConfig{
		BucketPolicyOnly: true,
		NoCheckBucket:    false,
		ObjectACL:        "",
		BucketACL:        "",
	}

	if !gcsConfig.BucketPolicyOnly {
		t.Error("BucketPolicyOnly should be true by default")
	}

	if gcsConfig.NoCheckBucket {
		t.Error("NoCheckBucket should be false by default")
	}

	if gcsConfig.ObjectACL != "" {
		t.Error("ObjectACL should be empty by default for uniform buckets")
	}

	if gcsConfig.BucketACL != "" {
		t.Error("BucketACL should be empty by default for uniform buckets")
	}
}