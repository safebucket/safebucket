package reconcile

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const defaultGrace = 30 * time.Minute

type Options struct {
	Apply  bool
	Grace  time.Duration
	Bucket string
}

func ParseOptions(args []string) (Options, error) {
	opts := Options{}
	fs := flag.NewFlagSet("reconcile", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), "Usage: safebucket reconcile [flags]\n\nFlags:\n")
		fs.PrintDefaults()
	}

	fs.BoolVar(&opts.Apply, "apply", false, "delete orphaned objects")
	fs.DurationVar(&opts.Grace, "grace", defaultGrace, "skip objects modified more recently than this")
	fs.StringVar(&opts.Bucket, "bucket", "", "limit the scan to one bucket ID")

	_ = fs.Parse(args)

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

	return opts, nil
}

func (o Options) prefix() string {
	if o.Bucket == "" {
		return objectPrefix
	}
	return objectPrefix + o.Bucket + "/"
}
