package database

import (
	"testing"
)

func TestDatabaseTypes(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DatabaseType
		expected string
	}{
		{"MySQL type", MySQL, "mysql"},
		{"PostgreSQL type", PostgreSQL, "postgresql"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.dbType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.dbType))
			}
		})
	}
}

func TestBackupFormats(t *testing.T) {
	tests := []struct {
		name     string
		format   BackupFormat
		expected string
	}{
		{"SQL format", SQL, "sql"},
		{"Custom format", Custom, "custom"},
		{"Binary format", Binary, "binary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.format) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.format))
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name: "Valid MySQL config",
			config: &ProviderConfig{
				Type:     MySQL,
				Host:     "localhost",
				Port:     3306,
				Username: "test",
				Password: "test",
			},
			wantErr: false,
		},
		{
			name: "Missing host",
			config: &ProviderConfig{
				Type:     MySQL,
				Username: "test",
				Password: "test",
			},
			wantErr: true,
		},
		{
			name: "Missing username",
			config: &ProviderConfig{
				Type: MySQL,
				Host: "localhost",
				Port: 3306,
			},
			wantErr: true,
		},
		{
			name: "Auto-set default port for MySQL",
			config: &ProviderConfig{
				Type:     MySQL,
				Host:     "localhost",
				Username: "test",
				Password: "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check if default port was set
			if !tt.wantErr && tt.config.Type == MySQL && tt.config.Port == 0 {
				if tt.config.Port != 3306 {
					t.Errorf("Expected port to be set to 3306, got %d", tt.config.Port)
				}
			}
		})
	}
}

func TestMigrateFromLegacyConfig(t *testing.T) {
	legacy := &LegacyDatabaseConfig{
		Host:          "localhost",
		Port:          3306,
		Username:      "testuser",
		Password:      "testpass",
		Timeout:       30,
		MysqldumpPath: "/usr/bin/mysqldump",
		MysqlPath:     "/usr/bin/mysql",
	}

	config := MigrateFromLegacyConfig(legacy)

	// Test basic migration
	if config.Type != MySQL {
		t.Errorf("Expected type MySQL, got %s", config.Type)
	}
	if config.Host != legacy.Host {
		t.Errorf("Expected host %s, got %s", legacy.Host, config.Host)
	}
	if config.Port != legacy.Port {
		t.Errorf("Expected port %d, got %d", legacy.Port, config.Port)
	}
	if config.Username != legacy.Username {
		t.Errorf("Expected username %s, got %s", legacy.Username, config.Username)
	}
	if config.Password != legacy.Password {
		t.Errorf("Expected password %s, got %s", legacy.Password, config.Password)
	}

	// Test timeout conversion
	expectedTimeout := "30s"
	if config.Timeout != expectedTimeout {
		t.Errorf("Expected timeout %s, got %s", expectedTimeout, config.Timeout)
	}

	// Test tool paths
	if config.DumpToolPath != legacy.MysqldumpPath {
		t.Errorf("Expected dump tool path %s, got %s", legacy.MysqldumpPath, config.DumpToolPath)
	}
	if config.ClientToolPath != legacy.MysqlPath {
		t.Errorf("Expected client tool path %s, got %s", legacy.MysqlPath, config.ClientToolPath)
	}

	// Test MySQL-specific config creation
	if config.MySQL == nil {
		t.Error("Expected MySQL config to be created")
	} else {
		if !config.MySQL.UseMyDumper {
			t.Error("Expected UseMyDumper to be true")
		}
		if !config.MySQL.SingleTransaction {
			t.Error("Expected SingleTransaction to be true")
		}
		if !config.MySQL.LockTables {
			t.Error("Expected LockTables to be true")
		}
		if !config.MySQL.RoutinesAndEvents {
			t.Error("Expected RoutinesAndEvents to be true")
		}
	}
}

func TestProviderFactory(t *testing.T) {
	factory := NewProviderFactory()

	// Test supported types
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) == 0 {
		t.Error("Expected at least one supported database type")
	}

	found := false
	for _, dbType := range supportedTypes {
		if dbType == MySQL {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected MySQL to be in supported types")
	}

	// Test MySQL provider creation
	config := &ProviderConfig{
		Type:     MySQL,
		Host:     "localhost",
		Port:     3306,
		Username: "test",
		Password: "test",
	}

	provider, err := factory.CreateProvider(config)
	if err != nil {
		t.Fatalf("Failed to create MySQL provider: %v", err)
	}
	if provider == nil {
		t.Fatal("Provider is nil")
	}
	if provider.GetProviderType() != MySQL {
		t.Errorf("Expected provider type MySQL, got %s", provider.GetProviderType())
	}

	// Clean up
	provider.Close()

	// Test PostgreSQL provider creation (should fail for now)
	config.Type = PostgreSQL
	_, err = factory.CreateProvider(config)
	if err == nil {
		t.Error("Expected PostgreSQL provider creation to fail in v1.1.6")
	}

	// Test invalid type
	config.Type = "invalid"
	_, err = factory.CreateProvider(config)
	if err == nil {
		t.Error("Expected invalid type to fail")
	}
}

func TestDatabaseInfo(t *testing.T) {
	dbInfo := &DatabaseInfo{
		Name:       "testdb",
		Size:       1024,
		TableCount: 5,
		IsSystem:   false,
		Charset:    "utf8mb4",
	}

	if dbInfo.Name != "testdb" {
		t.Errorf("Expected name testdb, got %s", dbInfo.Name)
	}
	if dbInfo.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", dbInfo.Size)
	}
	if dbInfo.IsSystem {
		t.Error("Expected IsSystem to be false")
	}
}

func TestBackupOptions(t *testing.T) {
	opts := &BackupOptions{
		Databases:     []string{"db1", "db2"},
		Directory:     "/tmp/backups",
		Format:        SQL,
		UseParallel:   true,
		Compression:   true,
		IncludeData:   true,
		IncludeSchema: true,
	}

	if len(opts.Databases) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(opts.Databases))
	}
	if opts.Format != SQL {
		t.Errorf("Expected SQL format, got %s", opts.Format)
	}
	if !opts.UseParallel {
		t.Error("Expected UseParallel to be true")
	}
}

func TestRestoreOptions(t *testing.T) {
	opts := &RestoreOptions{
		BackupPath:   "/tmp/backup.sql",
		TargetDB:     "restored_db",
		DropIfExists: true,
	}

	if opts.BackupPath != "/tmp/backup.sql" {
		t.Errorf("Expected backup path /tmp/backup.sql, got %s", opts.BackupPath)
	}
	if opts.TargetDB != "restored_db" {
		t.Errorf("Expected target DB restored_db, got %s", opts.TargetDB)
	}
	if !opts.DropIfExists {
		t.Error("Expected DropIfExists to be true")
	}
}