// Package source watches the operating system for Slack desktop notifications
// and reports the current count. Nothing here talks to Slack or the network.
package source

import (
	"context"
	"log"
	"time"
)

// Options configures a Source.
type Options struct {
	Interval time.Duration // how often to poll, where polling is needed
	Log      *log.Logger
	Verbose  bool
}

// Source reports Slack's current notification count via report. It calls
// report whenever the count may have changed (duplicates are fine) and
// returns when ctx is cancelled or on a fatal error.
type Source interface {
	Name() string
	Run(ctx context.Context, report func(count int)) error
	// Diagnose prints what the source can see, for the doctor command.
	Diagnose(ctx context.Context) string
}

func (o Options) debugf(format string, args ...any) {
	if o.Verbose && o.Log != nil {
		o.Log.Printf(format, args...)
	}
}

func ticker(ctx context.Context, d time.Duration, fn func()) {
	fn()
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn()
		}
	}
}
