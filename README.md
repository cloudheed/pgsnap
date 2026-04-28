# pgsnap

A fast, reliable PostgreSQL backup and restore tool.

## Features

- Multiple storage backends (local filesystem, Cloudheed, S3, GCS, Azure)
- Compression and encryption support
- Incremental backups
- Backup verification
- Simple CLI interface

## Installation

```bash
go install github.com/cloudheed/pgsnap/cmd/pgsnap@latest
```

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
# Edit pgsnap.yaml with your settings
```

2. Create a backup:

```bash
pgsnap backup
```

3. List available backups:

```bash
pgsnap list
```

4. Restore a backup:

```bash
pgsnap restore <backup-id>
```

## Configuration

pgsnap looks for configuration in the following locations:

1. Path specified by `--config` flag
2. `./pgsnap.yaml`
3. `~/.pgsnap.yaml`
4. `/etc/pgsnap/pgsnap.yaml`

Environment variables with `PGSNAP_` prefix override file settings:

```bash
export PGSNAP_POSTGRES_HOST=localhost
export PGSNAP_POSTGRES_PASSWORD=secret
```

See [pgsnap.example.yaml](pgsnap.example.yaml) for all available options.

## Commands

| Command   | Description                    |
|-----------|--------------------------------|
| `backup`  | Create a database backup       |
| `restore` | Restore a database backup      |
| `list`    | List available backups         |
| `verify`  | Verify backup integrity        |

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

## License

Apache 2.0 - see [LICENSE](LICENSE) for details.
