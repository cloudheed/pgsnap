package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/cloudheed/pgsnap/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Long: `List all available backups in the configured storage backend.

The output includes backup ID, size, and compression/encryption status.`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create storage backend
	backend, err := createBackend(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	// List all objects
	objects, err := backend.List(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(objects) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Filter to only backup files
	var backups []storage.ObjectInfo
	for _, obj := range objects {
		if strings.Contains(obj.Key, ".dump") {
			backups = append(backups, obj)
		}
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSIZE\tCOMPRESSED\tENCRYPTED\tDATE")
	fmt.Fprintln(w, "--\t----\t----------\t---------\t----")

	for _, b := range backups {
		id := extractID(b.Key)
		compressed := strings.Contains(b.Key, ".gz")
		encrypted := strings.HasSuffix(b.Key, ".enc")

		fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%s\n",
			id,
			formatSize(b.Size),
			compressed,
			encrypted,
			b.LastModified.Format("2006-01-02 15:04:05"),
		)
	}

	return w.Flush()
}

func extractID(key string) string {
	// Remove extensions to get ID
	id := key
	for _, ext := range []string{".enc", ".gz", ".dump"} {
		id = strings.TrimSuffix(id, ext)
	}
	return id
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	// List-specific flags can be added here
}
