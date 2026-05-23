# Database User Setup

Panduan praktis membuat user database untuk TenangDB backup & restore.

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

> ⚠️ `GRANT ... ON *.*` membutuhkan akses ke semua database. Untuk scope lebih sempit, ganti `*.*` dengan `db_name.*` per database.

### Verifikasi

```bash
mysql -u tenangdb -p -h 127.0.0.1 -e "SHOW DATABASES;"
mysql -u tenangdb -p -h 127.0.0.1 -e "SELECT @@version;"
```

### Permission Details

| Hak | Kegunaan |
|-----|----------|
| `SELECT` | Membaca data untuk backup |
| `SHOW DATABASES` | Mendeteksi database yang tersedia |
| `LOCK TABLES` | Konsistensi backup (mydumper) |
| `EVENT, TRIGGER` | Backup event scheduler + trigger |
| `REPLICATION CLIENT` | Cek posisi binary log (mydumper) |
| `CREATE, DROP, ALTER` | Membuat/mengganti database saat restore |
| `INSERT` | Memasukkan data saat restore |
| `INDEX` | Rebuild index setelah restore |

## PostgreSQL

### Minimal Permissions

```sql
CREATE ROLE tenangdb WITH LOGIN PASSWORD 'your-strong-password';

-- Backup: read all data
GRANT pg_read_all_data TO tenangdb;

-- Restore: write all data
GRANT pg_write_all_data TO tenangdb;
```

> ⚠️ PostgreSQL 15+ memiliki role bawaan `pg_read_all_data` dan `pg_write_all_data` yang merupakan cara termudah. Untuk PostgreSQL <15, perlu grant manual per tabel/schema.

### Alternatif (PostgreSQL <15)

```sql
CREATE ROLE tenangdb WITH LOGIN PASSWORD 'your-strong-password';

-- Per database (ulangi untuk setiap database yang akan di-backup)
\c your_database
GRANT CONNECT ON DATABASE your_database TO tenangdb;
GRANT USAGE ON SCHEMA public TO tenangdb;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO tenangdb;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO tenangdb;

-- Untuk restore
GRANT CREATE ON SCHEMA public TO tenangdb;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO tenangdb;
```

### Verifikasi

```bash
PGPASSWORD='your-strong-password' psql -h 127.0.0.1 -U tenangdb -d postgres -c "\l"
PGPASSWORD='your-strong-password' psql -h 127.0.0.1 -U tenangdb -d postgres -c "SELECT version();"
```

### Permission Details

| Hak | Kegunaan |
|-----|----------|
| `pg_read_all_data` | Membaca semua data/tabel (PG15+) |
| `pg_write_all_data` | Menulis/membuat tabel saat restore (PG15+) |
| `CONNECT` | Koneksi ke database |
| `USAGE ON SCHEMA` | Akses schema |
| `SELECT` | Membaca data untuk backup |
| `CREATE` | Membuat tabel saat restore |

## Docker

Untuk pengguna Docker, pastikan network container bisa reach database server:

```yaml
# docker-compose.yml
services:
  tenangdb:
    image: ghcr.io/abdullahainun/tenangdb:latest
    network_mode: host   # atau gabung di network yang sama
```

## Testing

Jalankan backup untuk verifikasi user berfungsi:

```bash
# Cek koneksi + daftar database
tenangdb init --config config.yaml

# Backup semua database
tenangdb backup --yes --config config.yaml

# Cek hasil
ls -la ./backups/
```
