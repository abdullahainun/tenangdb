# Backward Compatibility Guide

## Command Changes

### Recommended Usage
```bash
./tenangdb backup --config config.yaml
./tenangdb backup --config config.yaml --dry-run
./tenangdb backup --databases db1,db2 --config config.yaml
```

### Backward Compatible (Deprecated)
```bash
# Old default command (still works with deprecation warning)
./tenangdb --config config.yaml
./tenangdb --config config.yaml --dry-run
```

### Deprecation Warning
When using the old format:
```
WARN: DEPRECATED: Running tenangdb without 'backup' subcommand is deprecated.
```

## Migration Guide

Update your scripts:

```bash
# Old
./tenangdb --config config.yaml

# New
./tenangdb backup --config config.yaml
```

## Benefits of New Syntax

1. **Explicit**: Clear what the command does
2. **Consistent**: All operations have subcommands
3. **Future-proof**: Easy to add new subcommands

## Timeline

- **v1.x**: Both syntaxes work, deprecation warnings shown
- **v2.0**: Old syntax may be removed

## Other Commands (Unchanged)

```bash
./tenangdb cleanup --config config.yaml
./tenangdb restore --backup-path /path --target-database db
```
