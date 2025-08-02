# 🗄️ Database Providers Architecture

## Overview

TenangDB v1.1.6 introduces a new database provider architecture that prepares for multi-database support. This foundation enables future PostgreSQL support while maintaining full backward compatibility with existing MySQL configurations.

## Architecture

### Provider Interface

All database providers implement the `Provider` interface:

```go
type Provider interface {
    // Connection management
    TestConnection(ctx context.Context) error
    Close() error

    // Database operations
    ListDatabases(ctx context.Context) ([]*DatabaseInfo, error)
    DatabaseExists(ctx context.Context, dbName string) (bool, error)

    // Backup operations
    CreateBackup(ctx context.Context, opts *BackupOptions) ([]*BackupResult, error)
    
    // Restore operations
    RestoreBackup(ctx context.Context, opts *RestoreOptions) error
    
    // Tool availability
    GetAvailableTools() []string
    ValidateTools() error
    
    // Provider metadata
    GetProviderType() DatabaseType
    GetDefaultPort() int
    GetSystemDatabases() []string
}
```

### Supported Database Types

#### v1.1.6 (Current)
- ✅ **MySQL** - Full support with existing functionality
- ⏳ **PostgreSQL** - Coming in v1.2.0

#### v1.2.0 (Planned)
- ✅ **MySQL** - Enhanced with new features
- ✅ **PostgreSQL** - Full support with pg_dump/psql integration

## Configuration

### New Multi-Database Configuration

```yaml
database:
  type: mysql  # or postgresql (v1.2.0)
  host: localhost
  port: 3306
  username: backup_user
  password: secure_password
  
  # MySQL-specific configuration
  mysql:
    use_mydumper: true
    single_transaction: true
    lock_tables: true
    routines_and_events: true
  
  # PostgreSQL configuration (v1.2.0)
  postgresql:
    format: custom  # plain, custom, directory, tar
    use_pg_dump_parallel: true
```

### Backward Compatibility

Legacy MySQL-only configurations are automatically migrated:

```yaml
# Legacy format (still supported)
database:
  host: localhost
  port: 3306
  username: backup_user
  password: secure_password
```

This is automatically converted to:

```yaml
# New format with auto-detected type
database:
  type: mysql
  host: localhost
  port: 3306
  username: backup_user
  password: secure_password
  mysql:
    use_mydumper: true
    single_transaction: true
    lock_tables: true
    routines_and_events: true
```

## Usage

### Interactive Setup Wizard

The enhanced setup wizard now supports database type selection:

```bash
tenangdb init
```

Example interaction:
```
🗄️  Select Database Type:
  1. MySQL (default)
  2. PostgreSQL (coming in v1.2.0)
  3. Auto-detect from port

Choose database type [1]: 1
✅ Selected: MySQL
```

### Programmatic Usage

```go
import "github.com/abdullahainun/tenangdb/pkg/database"

// Create provider config
config := &database.ProviderConfig{
    Type:     database.MySQL,
    Host:     "localhost",
    Port:     3306,
    Username: "backup_user",
    Password: "secure_password",
}

// Create provider factory
factory := database.NewProviderFactory()

// Create MySQL provider
provider, err := factory.CreateProvider(config)
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Test connection
if err := provider.TestConnection(context.Background()); err != nil {
    log.Fatal(err)
}

// List databases
databases, err := provider.ListDatabases(context.Background())
if err != nil {
    log.Fatal(err)
}

// Create backup
opts := &database.BackupOptions{
    Databases: []string{"myapp_db"},
    Directory: "/backups",
    Timestamp: "20250101-120000",
    Format:    database.SQL,
}

results, err := provider.CreateBackup(context.Background(), opts)
if err != nil {
    log.Fatal(err)
}
```

## MySQL Provider

### Features

- ✅ mysqldump and mydumper support
- ✅ Parallel backup with mydumper
- ✅ Single transaction consistency
- ✅ Routines, events, and triggers included
- ✅ Automatic tool detection
- ✅ Connection validation

### Tool Requirements

**Required:**
- `mysqldump` OR `mydumper`
- `mysql` client

**Optional:**
- `mydumper` - For faster parallel backups
- `myloader` - For faster parallel restores

### Default Configuration

```yaml
mysql:
  use_mydumper: true        # Use mydumper if available
  single_transaction: true  # Consistent point-in-time backup
  lock_tables: true         # Lock tables during backup
  routines_and_events: true # Include stored procedures and events
```

## PostgreSQL Provider (v1.2.0)

### Planned Features

- ✅ pg_dump and pg_restore support
- ✅ Multiple backup formats (plain, custom, directory, tar)
- ✅ Parallel backup with pg_dump --jobs
- ✅ Role and privilege backup
- ✅ Connection validation

### Tool Requirements (v1.2.0)

**Required:**
- `pg_dump`
- `psql` client

**Optional:**
- `pg_basebackup` - For physical backups

### Planned Configuration (v1.2.0)

```yaml
postgresql:
  format: custom              # plain, custom, directory, tar
  use_pg_dump_parallel: true  # Use --jobs for parallel backup
  include_roles: true         # Backup roles and privileges
  ssl_mode: require          # SSL connection mode
```

## Migration Guide

### From Legacy MySQL Config

No action required! Legacy configurations are automatically migrated at runtime.

### Preparing for PostgreSQL (v1.2.0)

To prepare your configuration for PostgreSQL support:

1. **Update configuration format:**
   ```yaml
   # Before
   database:
     host: localhost
     port: 3306
   
   # After  
   database:
     type: mysql
     host: localhost
     port: 3306
     mysql:
       use_mydumper: true
   ```

2. **Install PostgreSQL tools** (when v1.2.0 is released):
   ```bash
   # Ubuntu/Debian
   sudo apt install postgresql-client
   
   # macOS
   brew install postgresql
   ```

## Testing

Run the provider tests:

```bash
go test ./pkg/database/...
```

Example test output:
```
✅ TestDatabaseTypes
✅ TestValidateConfig  
✅ TestMigrateFromLegacyConfig
✅ TestProviderFactory
✅ TestBackupOptions
```

## Error Handling

### Common Errors

**Invalid database type:**
```
Error: unsupported database type: mongodb
```

**Missing connection details:**
```
Error: database host is required
Error: database username is required
```

**Tool validation failures:**
```
Error: neither mysqldump nor mydumper found
```

### Resolution

1. **Check configuration** - Ensure all required fields are provided
2. **Validate tools** - Run `tenangdb init` to check tool availability
3. **Test connection** - Use the interactive wizard to verify connectivity

## Extending for New Databases

Adding support for a new database type:

1. **Implement Provider interface:**
   ```go
   type MongoDBProvider struct {
       // Implementation
   }
   
   func (p *MongoDBProvider) CreateBackup(ctx context.Context, opts *BackupOptions) ([]*BackupResult, error) {
       // MongoDB-specific backup logic
   }
   ```

2. **Add to factory:**
   ```go
   case MongoDB:
       return NewMongoDBProvider(config)
   ```

3. **Update configuration:**
   ```go
   type MongoDBConfig struct {
       AuthSource string `yaml:"auth_source"`
       SSL        bool   `yaml:"ssl"`
   }
   ```

## Best Practices

1. **Always test connections** before creating backups
2. **Use provider factory** for consistent provider creation
3. **Handle errors gracefully** with proper context
4. **Close providers** when done to release connections
5. **Validate tools** before starting backup operations

## Roadmap

### v1.1.6 (Current)
- ✅ Provider architecture foundation
- ✅ MySQL provider implementation  
- ✅ Backward compatibility
- ✅ Enhanced setup wizard

### v1.2.0 (Next)
- 🔄 PostgreSQL provider implementation
- 🔄 Multi-database backup jobs
- 🔄 Cross-database migration tools
- 🔄 Enhanced configuration validation

### Future Versions
- 🚀 MongoDB support
- 🚀 Redis backup integration
- 🚀 Cloud database providers (RDS, Cloud SQL)
- 🚀 Database-agnostic restore operations