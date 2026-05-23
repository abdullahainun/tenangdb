package database

import "context"

type DatabaseClient interface {
	CreateBackup(ctx context.Context, dbName, backupDir string) (string, error)
	CreateDirectory(path string) error
	RestoreBackup(ctx context.Context, backupPath, dbName string) error
	Close() error
	ListDatabases(ctx context.Context) ([]string, error)
}

var _ DatabaseClient = (*MySQLClient)(nil)
