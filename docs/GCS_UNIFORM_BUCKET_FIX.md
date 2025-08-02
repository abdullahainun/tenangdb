# 🔧 GCS Uniform Bucket-Level Access Fix

## Problem Description

When uploading backups to Google Cloud Storage buckets with **uniform bucket-level access** enabled, users encountered errors such as:

```
googleapi: Error 400: Cannot insert legacy ACL for an object when uniform bucket-level access is enabled
```

This occurs because uniform bucket-level access disables Access Control Lists (ACLs) and relies solely on IAM permissions.

## Root Cause

The issue was caused by rclone attempting to set object-level ACLs on GCS buckets configured with uniform bucket-level access. TenangDB's upload service wasn't automatically detecting GCS destinations or applying the appropriate rclone flags.

## Solution Implemented

### 🏗️ **Automatic GCS Detection & Configuration**

TenangDB now automatically:
1. **Detects GCS destinations** by checking common prefixes (`gcs:`, `mygcs:`, `googlecloud:`, `gc:`)
2. **Applies appropriate rclone flags** for uniform bucket-level access
3. **Uses secure defaults** that work with modern GCS buckets

### ⚙️ **New Configuration Options**

Added GCS-specific configuration under `upload.gcs`:

```yaml
upload:
  enabled: true
  destination: "mygcs:tenangdb-backup"
  
  # GCS-specific settings
  gcs:
    bucket_policy_only: true    # Enable for uniform bucket-level access
    no_check_bucket: false      # Skip bucket existence check  
    object_acl: ""             # Empty for uniform buckets
    bucket_acl: ""             # Empty for uniform buckets
```

### 🚀 **Smart Defaults**

- **`bucket_policy_only: true`** - Automatically enabled for GCS destinations
- **Empty ACLs** - Object and bucket ACLs disabled for uniform buckets
- **Backward compatibility** - Legacy GCS configurations still supported

## Technical Implementation

### 🔍 **GCS Detection Logic**

```go
func (s *Service) isGCSDestination(destination string) bool {
    prefixes := []string{"gcs:", "mygcs:", "googlecloud:", "gc:"}
    for _, prefix := range prefixes {
        if strings.HasPrefix(destination, prefix) {
            return true
        }
    }
    return false
}
```

### 🛠️ **Automatic Flag Addition**

When GCS destination is detected, TenangDB automatically adds:

```bash
rclone copy source dest \
  --gcs-bucket-policy-only=true \
  --gcs-object-acl= \
  --gcs-bucket-acl= \
  --progress \
  --stats 10s \
  --checksum
```

### 🔧 **Configuration Examples**

#### **Modern GCS (Uniform Bucket-Level Access)**
```yaml
upload:
  enabled: true
  destination: "mygcs:my-backup-bucket"
  gcs:
    bucket_policy_only: true  # Recommended for modern buckets
    object_acl: ""           # Empty for uniform access
    bucket_acl: ""           # Empty for uniform access
```

#### **Legacy GCS (ACL-based)**
```yaml
upload:
  enabled: true
  destination: "mygcs:legacy-bucket"
  gcs:
    bucket_policy_only: false
    object_acl: "publicRead"
    bucket_acl: "private"
```

#### **Limited Permissions**
```yaml
upload:
  enabled: true
  destination: "mygcs:restricted-bucket"
  gcs:
    bucket_policy_only: true
    no_check_bucket: true    # Skip bucket existence check
```

## Testing & Validation

### 🧪 **Automated Tests**

Added comprehensive tests covering:
- GCS destination detection
- Flag generation for different configurations
- Default behavior validation
- Edge cases and error handling

### ✅ **Test Coverage**

```bash
# Run GCS-specific tests
go test ./internal/upload/ -v

# Test Results:
✅ TestIsGCSDestination
✅ TestAddGCSFlags  
✅ TestShouldUseBucketPolicyOnly
✅ TestGCSConfigDefaults
```

### 🔄 **Backward Compatibility**

- **Existing configurations** continue to work unchanged
- **Legacy GCS setups** with ACLs still supported
- **Automatic migration** to secure defaults for new setups

## Benefits

### 🛡️ **Security Improvements**
- **Uniform bucket-level access** provides better security model
- **IAM-based permissions** instead of ACLs
- **Principle of least privilege** enforcement

### 🚀 **Reliability Enhancements**
- **Automatic error prevention** for uniform buckets
- **Smart defaults** reduce configuration errors
- **Better error messages** for troubleshooting

### 🔧 **Operational Benefits**
- **Zero configuration** required for most GCS setups
- **Automatic detection** eliminates manual flag management
- **Future-proof** configuration for GCS best practices

## Migration Guide

### **For Existing Users**

If you're experiencing GCS upload errors:

1. **Update TenangDB** to latest version
2. **No configuration changes required** - automatic detection applies fixes
3. **Verify uploads work** with existing config

### **For New GCS Setups**

1. **Enable uniform bucket-level access** on your GCS bucket (recommended)
2. **Use default configuration** - TenangDB automatically applies correct settings
3. **Set appropriate IAM permissions** on your service account

### **Troubleshooting**

If you still encounter issues:

1. **Check bucket configuration**:
   ```bash
   gsutil uniformbucketlevelaccess get gs://your-bucket-name
   ```

2. **Verify service account permissions**:
   - `roles/storage.objectUser` (recommended)
   - Or custom role with `storage.objects.*` permissions

3. **Test rclone configuration**:
   ```bash
   rclone lsd mygcs:your-bucket-name
   ```

4. **Enable debug logging**:
   ```yaml
   logging:
     level: debug
   ```

## Example Error Messages (Resolved)

### ❌ **Before Fix**
```
rclone command failed: exit status 1 (output: 
ERROR : Failed to copy: googleapi: Error 400: Cannot insert legacy ACL 
for an object when uniform bucket-level access is enabled. 
Read more at https://cloud.google.com/storage/docs/uniform-bucket-level-access)
```

### ✅ **After Fix**
```
🔧 Using GCS uniform bucket-level access mode
🔧 Disabled GCS ACLs for uniform bucket access
☁️  Upload completed successfully
```

## Best Practices

### 🛡️ **Security**
- **Enable uniform bucket-level access** on new GCS buckets
- **Use service accounts** with minimal required permissions
- **Rotate service account keys** regularly

### ⚙️ **Configuration**
- **Use default settings** for most GCS setups
- **Test uploads** after initial configuration
- **Monitor upload logs** for any issues

### 🔄 **Maintenance**
- **Keep TenangDB updated** for latest GCS compatibility
- **Review GCS bucket policies** periodically
- **Validate service account permissions** regularly

---

This fix ensures TenangDB works seamlessly with modern Google Cloud Storage buckets while maintaining backward compatibility and providing enhanced security through uniform bucket-level access! 🚀