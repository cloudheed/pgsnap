# pgsnap

[![CI](https://github.com/cloudheed/pgsnap/actions/workflows/ci.yml/badge.svg)](https://github.com/cloudheed/pgsnap/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudheed/pgsnap)](https://goreportcard.com/report/github.com/cloudheed/pgsnap)
[![Go Reference](https://pkg.go.dev/badge/github.com/cloudheed/pgsnap.svg)](https://pkg.go.dev/github.com/cloudheed/pgsnap)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

A fast, reliable PostgreSQL backup and restore tool.

## Features

- **Multiple storage backends**: Local filesystem, Amazon S3, S3-compatible (MinIO, DigitalOcean Spaces)
- **Compression**: Gzip compression to reduce backup size
- **Encryption**: AES-256-GCM encryption with password-based key derivation
- **Backup verification**: Verify backup integrity without restoring
- **Retention policies**: Automatic cleanup of old backups
- **Scheduled backups**: Built-in scheduler for automated backups
- **Simple CLI**: Easy to use command-line interface

## Installation

```bash
go install github.com/cloudheed/pgsnap/cmd/pgsnap@latest
```

Or download from [releases](https://github.com/cloudheed/pgsnap/releases).

Or build from source:

```bash
git clone https://github.com/cloudheed/pgsnap.git
cd pgsnap
make build
```

## Quick Start

1. Create a configuration file:

```bash
cp pgsnap.example.yaml pgsnap.yaml
# Edit pgsnap.yaml with your database settings
```

2. Create a backup:

```bash
pgsnap backup
```

3. List available backups:

```bash
pgsnap list
```

4. Verify a backup:

```bash
pgsnap verify <backup-id>
```

5. Restore a backup:

```bash
pgsnap restore <backup-id>
```

## Commands

| Command    | Description                              |
|------------|------------------------------------------|
| `backup`   | Create a database backup                 |
| `restore`  | Restore a database backup                |
| `list`     | List available backups                   |
| `verify`   | Verify backup integrity                  |
| `prune`    | Delete old backups per retention policy  |
| `schedule` | Run scheduled backups                    |

## Configuration

pgsnap looks for configuration in the following locations:

1. Path specified by `--config` flag
2. `./pgsnap.yaml`
3. `~/.pgsnap.yaml`
4. `/etc/pgsnap/pgsnap.yaml`

### Environment Variables

All settings can be overridden via environment variables with `PGSNAP_` prefix:

```bash
export PGSNAP_POSTGRES_HOST=localhost
export PGSNAP_POSTGRES_PASSWORD=secret
export PGSNAP_ENCRYPTION_PASSWORD=my-secure-password
```

### Example Configuration

```yaml
postgres:
  host: localhost
  port: 5432
  user: postgres
  password: ""
  database: myapp
  sslmode: prefer

storage:
  type: local  # or "s3"
  local:
    path: ./backups
  s3:
    bucket: my-backup-bucket
    region: us-east-1

backup:
  compress: true
  encrypt: false
  retention_days: 30
```

See [pgsnap.example.yaml](pgsnap.example.yaml) for all available options.

## Encryption

To enable encryption, set `backup.encrypt: true` and provide a password:

```bash
export PGSNAP_ENCRYPTION_PASSWORD="your-secure-password"
pgsnap backup
```

Backups are encrypted using AES-256-GCM with PBKDF2 key derivation. The salt is stored in the backup file header, so you only need to remember your password.

## S3 Storage

To use Amazon S3 or S3-compatible storage:

```yaml
storage:
  type: s3
  s3:
    bucket: my-backup-bucket
    region: us-east-1
    # For S3-compatible services (MinIO, etc.):
    # endpoint: https://minio.example.com
```

Credentials can be provided via:
- AWS credentials file (`~/.aws/credentials`)
- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- IAM role (when running on AWS)
- Config file (`storage.s3.access_key`, `storage.s3.secret_key`)

## Scheduled Backups

Run the scheduler for automated backups:

```bash
# Backup every hour
pgsnap schedule --interval 1h

# Backup every 6 hours
pgsnap schedule --interval 6h
```

For production, use with a process manager:

```bash
# systemd service example
[Unit]
Description=pgsnap backup scheduler

[Service]
ExecStart=/usr/local/bin/pgsnap schedule --interval 1h
Restart=always

[Install]
WantedBy=multi-user.target
```

## Retention Policies

Automatically clean up old backups:

```bash
# Preview what would be deleted
pgsnap prune --dry-run --max-age 30

# Delete backups older than 30 days
pgsnap prune --max-age 30

# Keep last 10 backups
pgsnap prune --max-count 10

# Keep 7 daily, 4 weekly, 12 monthly backups
pgsnap prune --keep-daily 7 --keep-weekly 4 --keep-monthly 12
```

## Development

```bash
# Install dependencies
go mod tidy

# Run tests
make test

# Run linter
make lint

# Build binary
make build
```

## Requirements

- Go 1.23+ (for building)
- PostgreSQL client tools (`pg_dump`, `pg_restore`)

## License

Apache 2.0 - see [LICENSE](LICENSE) for details.
