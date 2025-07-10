# MySQL User Setup for TenangDB

This guide shows how to create a MySQL user with the proper privileges for TenangDB backup and restore operations.

## 🔑 Create TenangDB User

### **Single User Setup (Recommended)**

Create one user that supports all TenangDB operations: mysqldump, mydumper, myloader, and mysql restore.

```sql
-- Connect as MySQL root user
mysql -u root -p

-- Create dedicated TenangDB user
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'your_secure_password_here';

-- Grant backup privileges
GRANT SELECT ON *.* TO 'tenangdb'@'%';
GRANT SHOW DATABASES ON *.* TO 'tenangdb'@'%';
GRANT SHOW VIEW ON *.* TO 'tenangdb'@'%';
GRANT LOCK TABLES ON *.* TO 'tenangdb'@'%';
GRANT EVENT ON *.* TO 'tenangdb'@'%';
GRANT TRIGGER ON *.* TO 'tenangdb'@'%';
GRANT ROUTINE ON *.* TO 'tenangdb'@'%';
GRANT RELOAD ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';

-- Grant restore privileges
GRANT INSERT, UPDATE, DELETE ON *.* TO 'tenangdb'@'%';
GRANT CREATE, DROP, ALTER ON *.* TO 'tenangdb'@'%';
GRANT INDEX, REFERENCES ON *.* TO 'tenangdb'@'%';
GRANT CREATE TEMPORARY TABLES ON *.* TO 'tenangdb'@'%';
GRANT CREATE VIEW ON *.* TO 'tenangdb'@'%';

-- Apply changes
FLUSH PRIVILEGES;

-- Verify user was created
SELECT User, Host FROM mysql.user WHERE User = 'tenangdb';
```

### **Security-Enhanced Setup (Host Restrictions)**

For better security, restrict access to specific hosts:

```sql
-- Create user with host restrictions
CREATE USER 'tenangdb'@'localhost' IDENTIFIED BY 'your_secure_password_here';
CREATE USER 'tenangdb'@'192.168.1.%' IDENTIFIED BY 'your_secure_password_here';

-- Grant same privileges as above for each host
GRANT SELECT, SHOW DATABASES, SHOW VIEW, LOCK TABLES, EVENT, TRIGGER, ROUTINE, RELOAD, REPLICATION CLIENT ON *.* TO 'tenangdb'@'localhost';
GRANT INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX, REFERENCES, CREATE TEMPORARY TABLES, CREATE VIEW ON *.* TO 'tenangdb'@'localhost';

GRANT SELECT, SHOW DATABASES, SHOW VIEW, LOCK TABLES, EVENT, TRIGGER, ROUTINE, RELOAD, REPLICATION CLIENT ON *.* TO 'tenangdb'@'192.168.1.%';
GRANT INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX, REFERENCES, CREATE TEMPORARY TABLES, CREATE VIEW ON *.* TO 'tenangdb'@'192.168.1.%';

FLUSH PRIVILEGES;
```

## 🔍 Privilege Explanation

### **Backup Operations (mysqldump & mydumper)**
- `SELECT` - Read table data
- `SHOW DATABASES` - List available databases
- `SHOW VIEW` - Export view definitions
- `LOCK TABLES` - Ensure backup consistency
- `EVENT` - Export scheduled events
- `TRIGGER` - Export table triggers
- `ROUTINE` - Export stored procedures and functions
- `RELOAD` - Execute FLUSH TABLES WITH READ LOCK
- `REPLICATION CLIENT` - Get binary log position for consistency

### **Restore Operations (myloader & mysql)**
- `INSERT, UPDATE, DELETE` - Restore table data
- `CREATE, DROP, ALTER` - Recreate database structures
- `INDEX, REFERENCES` - Handle indexes and foreign keys
- `CREATE TEMPORARY TABLES` - myloader temporary operations
- `CREATE VIEW` - Restore view definitions

## 🧪 Test User Privileges

### **Test Backup Access**
```bash
# Test mysqldump
mysqldump -h your_host -u tenangdb -p your_database > test_backup.sql

# Test mydumper
mydumper -h your_host -u tenangdb -p your_password -B your_database -o /tmp/test_backup/

# Test connection
mysql -h your_host -u tenangdb -p -e "SHOW DATABASES;"
```

### **Test Restore Access**
```bash
# Test mysql restore
mysql -h your_host -u tenangdb -p your_database < test_backup.sql

# Test myloader
myloader -h your_host -u tenangdb -p your_password -B your_database -d /tmp/test_backup/
```

## ⚙️ TenangDB Configuration

Update your `config.yaml` with the new user credentials:

```yaml
# Database connection settings
database:
  host: your_mysql_host
  port: 3306
  username: tenangdb
  password: your_secure_password_here
  timeout: 30

  mydumper:
    enabled: true
    
    myloader:
      enabled: true
      threads: 4
```

## 🛡️ Security Best Practices

### **1. Use Strong Passwords**
```sql
-- Generate secure password
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'Mg8$kL2#pX9@vN4!qR7&';
```

### **2. Database-Specific Privileges (Optional)**
For maximum security, grant privileges only to specific databases:
```sql
-- Grant privileges per database instead of global
GRANT SELECT, SHOW VIEW, LOCK TABLES, EVENT, TRIGGER ON database1.* TO 'tenangdb'@'%';
GRANT INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX ON database1.* TO 'tenangdb'@'%';

GRANT SELECT, SHOW VIEW, LOCK TABLES, EVENT, TRIGGER ON database2.* TO 'tenangdb'@'%';
GRANT INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX ON database2.* TO 'tenangdb'@'%';
```

### **3. Regular Privilege Review**
```sql
-- Review user privileges
SHOW GRANTS FOR 'tenangdb'@'%';

-- Check user connections
SELECT User, Host, Time FROM INFORMATION_SCHEMA.PROCESSLIST WHERE User = 'tenangdb';
```

### **4. Connection Limits (Optional)**
```sql
-- Limit concurrent connections
ALTER USER 'tenangdb'@'%' WITH MAX_CONNECTIONS_PER_HOUR 100;
ALTER USER 'tenangdb'@'%' WITH MAX_USER_CONNECTIONS 5;
```

## 🔄 User Management

### **Change Password**
```sql
ALTER USER 'tenangdb'@'%' IDENTIFIED BY 'new_secure_password';
FLUSH PRIVILEGES;
```

### **Remove User**
```sql
DROP USER 'tenangdb'@'%';
FLUSH PRIVILEGES;
```

### **Disable User Temporarily**
```sql
ALTER USER 'tenangdb'@'%' ACCOUNT LOCK;
-- Enable again
ALTER USER 'tenangdb'@'%' ACCOUNT UNLOCK;
```

## ✅ Quick Setup Script

Save this as `setup_mysql_user.sql`:

```sql
-- TenangDB MySQL User Setup
-- Replace 'your_secure_password_here' with actual secure password

CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'your_secure_password_here';

-- Backup privileges
GRANT SELECT ON *.* TO 'tenangdb'@'%';
GRANT SHOW DATABASES ON *.* TO 'tenangdb'@'%';
GRANT SHOW VIEW ON *.* TO 'tenangdb'@'%';
GRANT LOCK TABLES ON *.* TO 'tenangdb'@'%';
GRANT EVENT ON *.* TO 'tenangdb'@'%';
GRANT TRIGGER ON *.* TO 'tenangdb'@'%';
GRANT ROUTINE ON *.* TO 'tenangdb'@'%';
GRANT RELOAD ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';

-- Restore privileges
GRANT INSERT, UPDATE, DELETE ON *.* TO 'tenangdb'@'%';
GRANT CREATE, DROP, ALTER ON *.* TO 'tenangdb'@'%';
GRANT INDEX, REFERENCES ON *.* TO 'tenangdb'@'%';
GRANT CREATE TEMPORARY TABLES ON *.* TO 'tenangdb'@'%';
GRANT CREATE VIEW ON *.* TO 'tenangdb'@'%';

FLUSH PRIVILEGES;

-- Verify setup
SELECT User, Host FROM mysql.user WHERE User = 'tenangdb';
SHOW GRANTS FOR 'tenangdb'@'%';
```

Execute with:
```bash
mysql -u root -p < setup_mysql_user.sql
```

## 🎉 You're Ready!

Your TenangDB user is now configured with the proper privileges for all backup and restore operations. The user supports:

- ✅ **mysqldump** - Traditional MySQL backup
- ✅ **mydumper** - Fast, parallel backup
- ✅ **myloader** - Fast, parallel restore
- ✅ **mysql** - Traditional MySQL restore

Update your `config.yaml` with the new credentials and start using TenangDB!