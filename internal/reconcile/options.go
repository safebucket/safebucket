package reconcile

import (
	"errors"
	"flag"
	"fmt"
	"time"
)

const defaultGrace = 30 * time.Minute

type Options struct {
	Apply bool
	Grace time.Duration
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

	_ = fs.Parse(args)

	if opts.Grace < 0 {
		return Options{}, errors.New("--grace cannot be negative")
	}

	return opts, nil
}
