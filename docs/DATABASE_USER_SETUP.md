# Database User Setup

Practical guide to create database users for TenangDB backup & restore.

## MySQL/MariaDB

### Minimal Permissions

```sql
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'your-strong-password';

-- Backup: SELECT + SHOW DATABASES + LOCK TABLES
GRANT SELECT, SHOW DATABASES, LOCK TABLES, EVENT, TRIGGER ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';

-- Restore: CREATE + DROP + INSERT
GRANT CREATE, DROP, ALTER, INDEX, INSERT, UPDATE, DELETE ON *.* TO 'tenangdb'@'%';

FLUSH PRIVILEGES;
```

> ⚠️ `GRANT ... ON *.*` grants access to all databases. For narrower scope, replace `*.*` with `db_name.*` per database.

### Verification

```bash
mysql -u tenangdb -p -h 127.0.0.1 -e "SHOW DATABASES;"
mysql -u tenangdb -p -h 127.0.0.1 -e "SELECT @@version;"
```

### Permission Details

| Grant | Purpose |
|-------|---------|
| `SELECT` | Read data for backup |
| `SHOW DATABASES` | Detect available databases |
| `LOCK TABLES` | Consistent backup (mydumper) |
| `EVENT, TRIGGER` | Backup event scheduler + triggers |
| `REPLICATION CLIENT` | Check binary log position (mydumper) |
| `CREATE, DROP, ALTER` | Create/replace database during restore |
| `INSERT` | Insert data during restore |
| `INDEX` | Rebuild indexes after restore |

## PostgreSQL

### Minimal Permissions (PG15+)

```sql
CREATE ROLE tenangdb WITH LOGIN PASSWORD 'your-strong-password';

-- Backup: read all data
GRANT pg_read_all_data TO tenangdb;

-- Restore: write all data
GRANT pg_write_all_data TO tenangdb;
```

> PostgreSQL 15+ provides built-in `pg_read_all_data` and `pg_write_all_data` roles. For PG <15, use the alternative below.

### Alternative (PG <15)

```sql
CREATE ROLE tenangdb WITH LOGIN PASSWORD 'your-strong-password';

-- Per database (repeat for each database to back up)
\c your_database
GRANT CONNECT ON DATABASE your_database TO tenangdb;
GRANT USAGE ON SCHEMA public TO tenangdb;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO tenangdb;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO tenangdb;

-- For restore
GRANT CREATE ON SCHEMA public TO tenangdb;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO tenangdb;
```

### Verification

```bash
PGPASSWORD='your-strong-password' psql -h 127.0.0.1 -U tenangdb -d postgres -c "\l"
PGPASSWORD='your-strong-password' psql -h 127.0.0.1 -U tenangdb -d postgres -c "SELECT version();"
```

### Permission Details

| Grant | Purpose |
|-------|---------|
| `pg_read_all_data` | Read all data/tables (PG15+) |
| `pg_write_all_data` | Write/create tables during restore (PG15+) |
| `CONNECT` | Connect to database |
| `USAGE ON SCHEMA` | Access schema objects |
| `SELECT` | Read data for backup |
| `CREATE` | Create tables during restore |

## Docker Networking

For Docker users, ensure the container can reach the database server:

```yaml
# docker-compose.yml
services:
  tenangdb:
    image: ghcr.io/abdullahainun/tenangdb:latest
    network_mode: host   # or join the same network
```

See [DOCKER.md](../DOCKER.md#networking) for detailed networking options.

## Testing

Run backup to verify the user works:

```bash
# Check connection + list databases
tenangdb init --config config.yaml

# Backup all databases
tenangdb backup --yes --config config.yaml

# Check results
ls -la ./backups/
```
