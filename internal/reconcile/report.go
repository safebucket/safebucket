package reconcile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	sizeUnit       = 1024
	sizePrecision  = 1
	reportDirMode  = 0o750
	reportFileMode = 0o600
)

type groupStats struct {
	Count int64 `json:"count"`
	Size  int64 `json:"size"`
}

type objectRecord struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

type deleteFailure struct {
	Key   string `json:"key"`
	Error string `json:"error"`
}

type skippedObject struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

type report struct {
	GeneratedAt      time.Time       `json:"generated_at"`
	Prefix           string          `json:"prefix"`
	GracePeriod      string          `json:"grace_period"`
	Applied          bool            `json:"applied"`
	Total            groupStats      `json:"total"`
	Referenced       groupStats      `json:"referenced"`
	Orphaned         groupStats      `json:"orphaned"`
	TooRecent        groupStats      `json:"too_recent"`
	Unknown          groupStats      `json:"unknown"`
	OrphanedObjects  []objectRecord  `json:"orphaned_objects"`
	UnknownKeys      []string        `json:"unknown_keys"`
	MissingBucketIDs []string        `json:"missing_bucket_ids"`
	DeletedCount     int64           `json:"deleted_count"`
	DeletedSize      int64           `json:"deleted_size"`
	Failed           []deleteFailure `json:"failed"`
	Skipped          []skippedObject `json:"skipped"`
}

func newReport(prefix string, opts Options) *report {
	return &report{
		GeneratedAt:      time.Now().UTC(),
		Prefix:           prefix,
		GracePeriod:      opts.Grace.String(),
		OrphanedObjects:  make([]objectRecord, 0),
		UnknownKeys:      make([]string, 0),
		MissingBucketIDs: make([]string, 0),
		Failed:           make([]deleteFailure, 0),
		Skipped:          make([]skippedObject, 0),
	}
}

func writeReport(path string, result *report) error {
	if err := os.MkdirAll(filepath.Dir(path), reportDirMode); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	content, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	content = append(content, '\n')

	if writeErr := os.WriteFile(path, content, reportFileMode); writeErr != nil {
		return fmt.Errorf("write report: %w", writeErr)
	}

	return nil
}

func printSummary(result *report, opts Options) {
	fmt.Fprintf(os.Stdout, "Scanned %d objects under %s (%s)\n\n",
		result.Total.Count, result.Prefix, formatSize(result.Total.Size))
	fmt.Fprintf(
		os.Stdout,
		"  %-12s %8d  %s\n",
		"referenced",
		result.Referenced.Count,
		formatSize(result.Referenced.Size),
	)
	fmt.Fprintf(os.Stdout, "  %-12s %8d  %s\n", "orphaned", result.Orphaned.Count, formatSize(result.Orphaned.Size))
	fmt.Fprintf(os.Stdout, "  %-12s %8d  skipped (< %s old)\n", "too recent", result.TooRecent.Count, opts.Grace)
	fmt.Fprintf(os.Stdout, "  %-12s %8d  skipped\n", "unknown", result.Unknown.Count)
	fmt.Fprintln(os.Stdout)

	if len(result.MissingBucketIDs) > 0 {
		fmt.Fprintf(os.Stdout, "%d bucket(s) no longer exist in the database: %s\n\n",
			len(result.MissingBucketIDs), strings.Join(result.MissingBucketIDs, ", "))
	}
}

func printResult(result *report, opts Options) {
	if !opts.Apply {
		fmt.Fprintf(os.Stdout, "Nothing was deleted. Re-run with --apply to delete %d orphaned objects (%s).\n",
			result.Orphaned.Count, formatSize(result.Orphaned.Size))
		return
	}

	fmt.Fprintf(os.Stdout, "Deleted %d orphaned objects (%s), %d failed",
		result.DeletedCount, formatSize(result.DeletedSize), len(result.Failed))
	if len(result.Skipped) > 0 {
		fmt.Fprintf(os.Stdout, ", %d skipped", len(result.Skipped))
	}
	fmt.Fprintln(os.Stdout, ".")
}

func formatSize(size int64) string {
	if size < sizeUnit {
		return strconv.FormatInt(size, 10) + " B"
	}

	value := float64(size)
	for _, unit := range []string{"KB", "MB", "GB", "TB", "PB", "EB"} {
		value /= sizeUnit
		if value < sizeUnit {
			return fmt.Sprintf("%.*f %s", sizePrecision, value, unit)
		}
	}

	return fmt.Sprintf("%.*f EB", sizePrecision, value)
}
