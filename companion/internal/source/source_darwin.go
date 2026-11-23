package source

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// On macOS Slack puts its count of notification-worthy unreads (DMs,
// mentions, keywords, channels you asked to be notified about) on its Dock
// icon. lsappinfo reads that badge label without any special permission, and
// Slack clears it itself as you read, so no focus heuristics are needed.
// A bare "•" means unread messages that would not notify, which we count as 0.

type darwinSource struct{ opts Options }

// New returns the platform source.
func New(opts Options) Source { return &darwinSource{opts} }

func (s *darwinSource) Name() string { return "macOS Dock badge" }

var labelRe = regexp.MustCompile(`"label"="([^"]*)"`)

func (s *darwinSource) badge(ctx context.Context) (label string, err error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/lsappinfo", "info", "-only", "StatusLabel", "Slack").Output()
	if err != nil {
		return "", err
	}
	if m := labelRe.FindSubmatch(out); m != nil {
		return string(m[1]), nil
	}
	return "", nil // Slack not running or no badge
}

func parseBadge(label string) int {
	label = strings.TrimSpace(strings.TrimSuffix(label, "+"))
	n, err := strconv.Atoi(label)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func (s *darwinSource) Run(ctx context.Context, report func(int)) error {
	var lastErr string
	ticker(ctx, s.opts.Interval, func() {
		label, err := s.badge(ctx)
		if err != nil {
			if ctx.Err() == nil && err.Error() != lastErr {
				s.opts.Log.Printf("lsappinfo: %v", err)
				lastErr = err.Error()
			}
			return
		}
		lastErr = ""
		report(parseBadge(label))
	})
	return nil
}

func (s *darwinSource) Diagnose(ctx context.Context) string {
	label, err := s.badge(ctx)
	if err != nil {
		return fmt.Sprintf("lsappinfo failed: %v\n", err)
	}
	if label == "" {
		return "Slack Dock badge: none (Slack not running, or nothing unread)\n"
	}
	return fmt.Sprintf("Slack Dock badge: %q -> count %d\n", label, parseBadge(label))
}
