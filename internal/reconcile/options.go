package reconcile

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultGrace          = 24 * time.Hour
	defaultReportDir      = "data"
	reportTimestampLayout = "20060102-150405"
)

type Options struct {
	Apply  bool
	Force  bool
	Grace  time.Duration
	Bucket string
	Report string
}

func newFlagSet(opts *Options) *flag.FlagSet {
	fs := flag.NewFlagSet("reconcile", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.BoolVar(&opts.Apply, "apply", false, "delete orphaned objects")
	fs.BoolVar(&opts.Force, "force", false, "allow --apply when the orphan ratio safety check trips")
	fs.DurationVar(&opts.Grace, "grace", defaultGrace, "skip objects modified more recently than this")
	fs.StringVar(&opts.Bucket, "bucket", "", "limit the scan to one bucket ID")
	fs.StringVar(&opts.Report, "report", "", "JSON report path")

	return fs
}

func ParseOptions(args []string) (Options, error) {
	opts := Options{}
	fs := newFlagSet(&opts)

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}
	if fs.NArg() > 0 {
		return Options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if opts.Grace < 0 {
		return Options{}, errors.New("--grace cannot be negative")
	}
	if opts.Bucket != "" {
		bucketID, err := uuid.Parse(opts.Bucket)
		if err != nil {
			return Options{}, fmt.Errorf("invalid --bucket %q: %w", opts.Bucket, err)
		}
		opts.Bucket = bucketID.String()
	}
	if opts.Report == "" {
		opts.Report = filepath.Join(
			defaultReportDir,
			"reconcile-"+time.Now().Format(reportTimestampLayout)+".json",
		)
	}

	return opts, nil
}

func Usage() string {
	opts := Options{}
	fs := newFlagSet(&opts)

	var builder strings.Builder
	builder.WriteString("Usage: safebucket reconcile [flags]\n\nFlags:\n")
	fs.SetOutput(&builder)
	fs.PrintDefaults()

	return builder.String()
}

func (o Options) prefix() string {
	if o.Bucket == "" {
		return objectPrefix
	}
	return objectPrefix + o.Bucket + "/"
}
