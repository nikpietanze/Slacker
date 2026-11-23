package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// Windows keeps every toast shown in the notification center (Action Center)
// in a per-user SQLite database. We read it, read-only, and count the toasts
// that belong to Slack. When the user opens/focuses Slack we treat everything
// up to that point as seen, mirroring how Slack's own unread state clears.
//
// Requires Slack's "Deliver notifications via" setting to use Windows
// notifications (the default), not Slack's built-in notifications.

const slackToastQuery = `SELECT n.Id FROM Notification n
JOIN NotificationHandler h ON h.RecordId = n.HandlerId
WHERE lower(n.Type) = 'toast' AND lower(h.PrimaryId) LIKE '%slack%'`

type windowsSource struct {
	opts   Options
	dbPath string
	// acked holds toast ids that existed while Slack was focused. A set, not
	// a high-water mark: SQLite can reuse the id of a deleted row.
	acked map[int64]bool
}

// New returns the platform source.
func New(opts Options) Source {
	return &windowsSource{
		opts:   opts,
		acked:  map[int64]bool{},
		dbPath: filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Notifications", "wpndatabase.db"),
	}
}

func (s *windowsSource) Name() string { return "Windows notification center" }

func (s *windowsSource) Run(ctx context.Context, report func(int)) error {
	if _, err := os.Stat(s.dbPath); err != nil {
		return fmt.Errorf("notification database not found: %w", err)
	}
	var lastSig string
	var ids []int64
	var lastQuery time.Time
	var lastErr string
	ticker(ctx, s.opts.Interval, func() {
		// Only hit SQLite when the database files changed (or every 30s).
		sig := fileSig(s.dbPath) + fileSig(s.dbPath+"-wal")
		if sig != lastSig || time.Since(lastQuery) > 30*time.Second {
			got, err := s.slackToastIDs()
			if err != nil {
				if err.Error() != lastErr {
					s.opts.Log.Printf("reading notification database: %v", err)
					lastErr = err.Error()
				}
				return
			}
			lastErr = ""
			ids, lastSig, lastQuery = got, sig, time.Now()
			s.pruneAcked(ids)
		}
		if slackFocused() {
			for _, id := range ids {
				s.acked[id] = true
			}
		}
		report(s.unacked(ids))
	})
	return nil
}

// pruneAcked forgets acknowledged ids that are gone, so a reused id counts.
func (s *windowsSource) pruneAcked(ids []int64) {
	present := make(map[int64]bool, len(ids))
	for _, id := range ids {
		present[id] = true
	}
	for id := range s.acked {
		if !present[id] {
			delete(s.acked, id)
		}
	}
}

func (s *windowsSource) unacked(ids []int64) int {
	n := 0
	for _, id := range ids {
		if !s.acked[id] {
			n++
		}
	}
	return n
}

func (s *windowsSource) slackToastIDs() ([]int64, error) {
	rows, err := s.query(slackToastQuery)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if id, err := strconv.ParseInt(r[0], 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// query opens the live database read-only. If Windows has it locked, it
// falls back to querying a private copy in the temp folder.
func (s *windowsSource) query(sql string) ([][]string, error) {
	d, err := openDB(s.dbPath, sqliteOpenReadOnly)
	if err == nil {
		rows, qerr := d.query(sql)
		d.Close()
		if qerr == nil {
			return rows, nil
		}
		err = qerr
	}
	tmp := filepath.Join(os.TempDir(), "slacker-wpn")
	os.MkdirAll(tmp, 0o700)
	os.Remove(filepath.Join(tmp, "wpn.db-shm"))
	os.Remove(filepath.Join(tmp, "wpn.db-wal"))
	for _, suffix := range []string{"", "-wal"} {
		if cerr := copyFile(s.dbPath+suffix, filepath.Join(tmp, "wpn.db"+suffix)); cerr != nil && suffix == "" {
			return nil, fmt.Errorf("%v; copy fallback: %w", err, cerr)
		}
	}
	// The copy is ours, so open it read-write: SQLite needs that to rebuild
	// the WAL index for a copied database.
	d, cerr := openDB(filepath.Join(tmp, "wpn.db"), sqliteOpenReadWrite)
	if cerr != nil {
		return nil, fmt.Errorf("%v; copy fallback: %w", err, cerr)
	}
	defer d.Close()
	return d.query(sql)
}

func (s *windowsSource) Diagnose(ctx context.Context) string {
	var b strings.Builder
	fmt.Fprintf(&b, "database: %s\n", s.dbPath)
	if _, err := os.Stat(s.dbPath); err != nil {
		fmt.Fprintf(&b, "  NOT FOUND: %v\n", err)
		return b.String()
	}
	rows, err := s.query(`SELECT h.PrimaryId, count(n.Id) FROM NotificationHandler h
LEFT JOIN Notification n ON n.HandlerId = h.RecordId AND lower(n.Type) = 'toast'
WHERE lower(h.PrimaryId) LIKE '%slack%' GROUP BY h.RecordId`)
	if err != nil {
		fmt.Fprintf(&b, "  query failed: %v\n", err)
		return b.String()
	}
	if len(rows) == 0 {
		b.WriteString("  no Slack entry yet. Make sure Slack delivers notifications via Windows\n" +
			"  (Slack > Settings > Notifications) and that one has been shown at least once.\n")
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "  app %q: %s toast(s) in notification center\n", r[0], r[1])
	}
	fmt.Fprintf(&b, "Slack focused right now: %v\n", slackFocused())
	return b.String()
}

func fileSig(p string) string {
	st, err := os.Stat(p)
	if err != nil {
		return "-"
	}
	return fmt.Sprintf("%d/%d;", st.ModTime().UnixNano(), st.Size())
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// slackFocused reports whether the foreground window belongs to slack.exe.
func slackFocused() bool {
	hwnd := windows.GetForegroundWindow()
	if hwnd == 0 {
		return false
	}
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil || pid == 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_PATH)
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return false
	}
	return strings.EqualFold(filepath.Base(windows.UTF16ToString(buf[:n])), "slack.exe")
}
