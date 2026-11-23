package main

import (
	"context"
	"log"
	"sync"
	"time"

	"slacker/companion/internal/bridge"
	"slacker/companion/internal/source"
)

// watcher mirrors the Slack notification count into every Slacker addon
// folder it can find. It keeps running when no addon is installed yet and
// picks one up as soon as it appears.
type watcher struct {
	cfg config
	log *log.Logger
	src source.Source

	mu       sync.Mutex
	count    int            // -1 until the first reading
	dirs     map[string]int // addon dir -> code last written there
	srcErr   string
	scanned  time.Time
	onChange func()
}

func newWatcher(cfg config, logger *log.Logger) *watcher {
	return &watcher{
		cfg:   cfg,
		log:   logger,
		src:   source.New(source.Options{Interval: cfg.interval, Log: logger, Verbose: cfg.verbose}),
		count: -1,
		dirs:  map[string]int{},
	}
}

// status is a snapshot for the tray menu.
type status struct {
	Count  int // -1 unknown
	Addons int // addon folders being written to
	Err    string
}

func (w *watcher) Status() status {
	w.mu.Lock()
	defer w.mu.Unlock()
	return status{Count: w.count, Addons: len(w.dirs), Err: w.srcErr}
}

func (w *watcher) changed() {
	if w.onChange != nil {
		w.onChange()
	}
}

// rescan looks for addon folders: every 10s while none are known, then every 30s.
func (w *watcher) rescan(force bool) {
	every := 30 * time.Second
	if len(w.dirs) == 0 {
		every = 10 * time.Second
	}
	if !force && time.Since(w.scanned) < every {
		return
	}
	w.scanned = time.Now()
	next := map[string]int{}
	for _, d := range discover(w.cfg) {
		if c, ok := w.dirs[d]; ok {
			next[d] = c
		} else {
			next[d] = -1
			w.log.Printf("bridging to %s", d)
		}
	}
	if len(next) == 0 && len(w.dirs) > 0 {
		w.log.Printf("no Slacker addon folder found any more")
	}
	w.dirs = next
}

// publish writes code into every known addon folder that doesn't have it yet.
func (w *watcher) publish(code int) {
	for d, last := range w.dirs {
		if last == code {
			continue
		}
		if err := bridge.WriteCode(d, code); err != nil {
			w.log.Printf("writing %s: %v", d, err)
			continue
		}
		w.dirs[d] = code
	}
}

func (w *watcher) Run(ctx context.Context) error {
	w.log.Printf("slacker-companion %s watching %s", version, w.src.Name())
	counts := make(chan int, 16)
	errc := make(chan error, 1)
	go func() {
		errc <- w.src.Run(ctx, func(n int) {
			select {
			case counts <- n:
			default: // a newer value will follow
			}
		})
	}()

	w.mu.Lock()
	w.rescan(true)
	w.publish(bridge.CodeForCount(0))
	w.mu.Unlock()
	w.changed()

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case n := <-counts:
			w.mu.Lock()
			if n != w.count {
				w.log.Printf("slack notifications: %d", n)
			}
			changed := n != w.count
			w.count = n
			w.publish(bridge.CodeForCount(n))
			w.mu.Unlock()
			if changed {
				w.changed()
			}
		case <-tick.C:
			w.mu.Lock()
			before := len(w.dirs)
			w.rescan(false)
			if w.count >= 0 {
				w.publish(bridge.CodeForCount(w.count))
			}
			after := len(w.dirs)
			w.mu.Unlock()
			if before != after {
				w.changed()
			}
		case err := <-errc:
			w.mu.Lock()
			w.publish(bridge.CodeOffline)
			if err != nil {
				w.srcErr = err.Error()
			}
			w.mu.Unlock()
			w.changed()
			if err != nil {
				w.log.Printf("notification source stopped: %v", err)
			}
			<-ctx.Done() // keep the tray up so the error is visible
			return err
		case <-ctx.Done():
			w.log.Printf("stopping")
			w.mu.Lock()
			w.rescan(true)
			w.publish(bridge.CodeOffline)
			w.mu.Unlock()
			return nil
		}
	}
}
