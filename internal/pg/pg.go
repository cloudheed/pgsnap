// Package pg provides PostgreSQL client utilities.
package pg

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Config holds PostgreSQL connection settings.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// ConnectionString returns a PostgreSQL connection string.
func (c *Config) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

// Env returns environment variables for pg_dump/pg_restore.
func (c *Config) Env() []string {
	return []string{
		fmt.Sprintf("PGHOST=%s", c.Host),
		fmt.Sprintf("PGPORT=%d", c.Port),
		fmt.Sprintf("PGUSER=%s", c.User),
		fmt.Sprintf("PGPASSWORD=%s", c.Password),
		fmt.Sprintf("PGDATABASE=%s", c.Database),
		fmt.Sprintf("PGSSLMODE=%s", c.SSLMode),
	}
}

// DumpOptions configures pg_dump behavior.
type DumpOptions struct {
	Format          string   // custom, plain, directory, tar
	Compress        int      // compression level 0-9
	Jobs            int      // parallel jobs
	ExcludeTables   []string // tables to exclude
	IncludeTables   []string // tables to include (if set, only these)
	SchemaOnly      bool     // dump schema only
	DataOnly        bool     // dump data only
	NoOwner         bool     // skip ownership
	NoPrivileges    bool     // skip privileges
	CleanFirst      bool     // add DROP statements
}

// DefaultDumpOptions returns sensible defaults.
func DefaultDumpOptions() DumpOptions {
	return DumpOptions{
		Format:   "custom",
		Compress: 6,
		Jobs:     4,
	}
}

// Dump runs pg_dump and writes output to the provided writer.
func Dump(ctx context.Context, cfg *Config, opts DumpOptions, w io.Writer) error {
	args := buildDumpArgs(cfg, opts)

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(cmd.Environ(), cfg.Env()...)
	cmd.Stdout = w

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w: %s", err, stderr.String())
	}

	return nil
}

func buildDumpArgs(cfg *Config, opts DumpOptions) []string {
	args := []string{
		"--format=" + opts.Format,
		"--dbname=" + cfg.Database,
	}

	if opts.Compress > 0 {
		args = append(args, fmt.Sprintf("--compress=%d", opts.Compress))
	}

	if opts.Jobs > 1 && opts.Format == "directory" {
		args = append(args, fmt.Sprintf("--jobs=%d", opts.Jobs))
	}

	if opts.SchemaOnly {
		args = append(args, "--schema-only")
	}

	if opts.DataOnly {
		args = append(args, "--data-only")
	}

	if opts.NoOwner {
		args = append(args, "--no-owner")
	}

	if opts.NoPrivileges {
		args = append(args, "--no-privileges")
	}

	if opts.CleanFirst {
		args = append(args, "--clean")
	}

	for _, t := range opts.ExcludeTables {
		args = append(args, "--exclude-table="+t)
	}

	for _, t := range opts.IncludeTables {
		args = append(args, "--table="+t)
	}

	return args
}

// RestoreOptions configures pg_restore behavior.
type RestoreOptions struct {
	Jobs         int    // parallel jobs
	NoOwner      bool   // skip ownership
	NoPrivileges bool   // skip privileges
	CleanFirst   bool   // drop objects before creating
	CreateDB     bool   // create database before restore
	TargetDB     string // target database (if different)
}

// DefaultRestoreOptions returns sensible defaults.
func DefaultRestoreOptions() RestoreOptions {
	return RestoreOptions{
		Jobs: 4,
	}
}

// Restore runs pg_restore from the provided reader.
func Restore(ctx context.Context, cfg *Config, opts RestoreOptions, r io.Reader) error {
	args := buildRestoreArgs(cfg, opts)

	cmd := exec.CommandContext(ctx, "pg_restore", args...)
	cmd.Env = append(cmd.Environ(), cfg.Env()...)
	cmd.Stdin = r

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore failed: %w: %s", err, stderr.String())
	}

	return nil
}

func buildRestoreArgs(cfg *Config, opts RestoreOptions) []string {
	targetDB := cfg.Database
	if opts.TargetDB != "" {
		targetDB = opts.TargetDB
	}

	args := []string{
		"--dbname=" + targetDB,
	}

	if opts.Jobs > 1 {
		args = append(args, fmt.Sprintf("--jobs=%d", opts.Jobs))
	}

	if opts.NoOwner {
		args = append(args, "--no-owner")
	}

	if opts.NoPrivileges {
		args = append(args, "--no-privileges")
	}

	if opts.CleanFirst {
		args = append(args, "--clean")
	}

	if opts.CreateDB {
		args = append(args, "--create")
	}

	return args
}

// CheckTools verifies pg_dump and pg_restore are available.
func CheckTools() error {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		return fmt.Errorf("pg_dump not found: %w", err)
	}
	if _, err := exec.LookPath("pg_restore"); err != nil {
		return fmt.Errorf("pg_restore not found: %w", err)
	}
	return nil
}

// Version returns the PostgreSQL client version.
func Version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "pg_dump", "--version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
