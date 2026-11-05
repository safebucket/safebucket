package reconcile

import (
	"fmt"
	"os"
	"strconv"

	"github.com/safebucket/safebucket/internal/storage"
)

const (
	sizeUnit      = 1024
	sizePrecision = 1
)

type stats struct {
	Count int64
	Size  int64
}

func (s *stats) add(size int64) {
	s.Count++
	s.Size += size
}

type scanResult struct {
	Total      stats
	Referenced stats
	Orphaned   stats
	TooRecent  stats
	Unknown    stats
	Orphans    []storage.ObjectInfo
}

func printSummary(result *scanResult, opts Options) {
	fmt.Fprintf(os.Stdout, "Scanned %d objects under %s (%s)\n\n",
		result.Total.Count, objectPrefix, formatSize(result.Total.Size))
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
