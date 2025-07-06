# 🔄 Backward Compatibility Guide

TenangDB maintains backward compatibility while introducing the new `backup` subcommand.

## 📋 Command Changes

### ✅ **New Recommended Usage**
```bash
# Explicit backup subcommand (recommended)
./tenangdb backup --config config.yaml
./tenangdb backup --config config.yaml --dry-run
./tenangdb backup --databases db1,db2 --config config.yaml
```

### 🔄 **Backward Compatible (Deprecated)**
```bash
# Old default command (still works with deprecation warning)
./tenangdb --config config.yaml
./tenangdb --config config.yaml --dry-run
./tenangdb --databases db1,db2 --config config.yaml
```

### ⚠️ **Deprecation Warning**
When using the old format, you'll see:
```
WARN: DEPRECATED: Running tenangdb without 'backup' subcommand is deprecated. Use 'tenangdb backup' instead.
```

## 🔧 **Migration Guide**

### **Production Systems**
Your existing production setups will continue to work:

```bash
# These will keep working (with deprecation warnings):
/opt/tenangdb/tenangdb --config /etc/tenangdb/config.yaml
/opt/tenangdb/tenangdb --config /etc/tenangdb/config.yaml --dry-run
```

### **Recommended Migration**
Update your scripts and systemd files to use the new format:

```bash
# Old systemd ExecStart
ExecStart=/opt/tenangdb/tenangdb --config /etc/tenangdb/config.yaml

# New systemd ExecStart
ExecStart=/opt/tenangdb/tenangdb backup --config /etc/tenangdb/config.yaml
```

### **Scripts Migration**
```bash
# Update your backup scripts:
# Old
./tenangdb --config config.yaml

# New  
./tenangdb backup --config config.yaml
```

## 🎯 **Benefits of New Syntax**

1. **Explicit**: Clear what the command does
2. **Consistent**: All operations have subcommands
3. **Professional**: Follows modern CLI conventions
4. **Self-documenting**: Easy to understand
5. **Future-proof**: Easy to add new subcommands

## 🗓️ **Migration Timeline**

- **v1.0+**: Both syntaxes work, deprecation warnings shown
- **v2.0** (future): Old syntax may be removed
- **Now**: Start migrating to new syntax

## 🔄 **Other Commands (Unchanged)**
These commands remain the same:
```bash
./tenangdb cleanup --config config.yaml
./tenangdb restore --backup-path /path --target-database db
./tenangdb exporter --port 9090
```

## 🆘 **Troubleshooting**

**Q: My production system shows deprecation warnings**  
A: This is normal. The old syntax still works. Update when convenient.

**Q: Should I update immediately?**  
A: No rush. Update at your convenience during maintenance windows.

**Q: Will old syntax break?**  
A: Not in v1.x versions. You have time to migrate gradually.

**Q: How to silent deprecation warnings?**  
A: Use the new `backup` subcommand syntax.