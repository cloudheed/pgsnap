# TODO

## Implemented

- [x] CLI scaffold (Cobra)
- [x] Configuration (Viper, YAML, env vars)
- [x] Local storage backend
- [x] PostgreSQL dump/restore wrapper (pg_dump, pg_restore)
- [x] Compression (gzip)
- [x] Encryption (AES-256-GCM with PBKDF2)
- [x] Backup command
- [x] Restore command
- [x] List command
- [x] CI/CD pipeline
- [x] Release automation (GoReleaser)

## High Priority

- [ ] **Verify command** - Implement backup integrity verification
  - Checksum validation
  - Test restore to temp database

- [ ] **Manifest files** - Store backup metadata
  - JSON manifest with backup info
  - Database schema version
  - Table list and row counts

- [ ] **Encryption salt storage** - Currently salt is not persisted
  - Store salt alongside encrypted backup
  - Header format: `[salt:16][nonce:12][ciphertext]`

- [ ] **Streaming encryption/compression** - Current implementation loads entire backup into memory
  - Use io.Pipe for streaming
  - Chunk-based encryption for large databases

- [ ] **Progress reporting** - Show backup/restore progress
  - Bytes transferred
  - Estimated time remaining

## Medium Priority

- [ ] **S3 storage backend**
  - AWS SDK integration
  - Multipart uploads for large backups
  - S3-compatible endpoints (MinIO, DigitalOcean Spaces)

- [ ] **GCS storage backend**
  - Google Cloud Storage integration
  - Service account authentication

- [ ] **Azure Blob storage backend**
  - Azure SDK integration
  - Managed identity support

- [ ] **Incremental backups**
  - WAL archiving
  - Point-in-time recovery (PITR)
  - pg_basebackup integration

- [ ] **Retention policies**
  - Auto-delete old backups
  - Keep daily/weekly/monthly
  - Configurable retention rules

- [ ] **Parallel backup/restore**
  - Multiple tables concurrently
  - Directory format support

## Low Priority

- [ ] **Web UI** - Simple dashboard
  - Backup history
  - One-click restore
  - Configuration editor

- [ ] **Notifications**
  - Slack/Discord webhooks
  - Email notifications
  - PagerDuty integration

- [ ] **Metrics/Monitoring**
  - Prometheus metrics endpoint
  - Backup success/failure rates
  - Duration and size tracking

- [ ] **Scheduled backups**
  - Built-in cron scheduler
  - Or documentation for system cron

- [ ] **Multi-database support**
  - Backup all databases on server
  - Selective restore

- [ ] **Schema migrations**
  - Track schema versions
  - Migration compatibility checks

- [ ] **Backup catalog**
  - SQLite or PostgreSQL catalog
  - Search and filter backups
  - Tags and labels

## Technical Debt

- [ ] Add comprehensive tests for pg, backup, restore packages
- [ ] Add integration tests with real PostgreSQL
- [ ] Add benchmarks for compression/encryption
- [ ] Improve error messages and logging
- [ ] Add `--dry-run` flag for backup/restore
- [ ] Add `--verbose` flag for detailed output
- [ ] Document all configuration options
- [ ] Add man pages
