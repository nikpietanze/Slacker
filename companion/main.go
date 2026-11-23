// Command slacker-companion mirrors the number of Slack desktop
// notifications into the Slacker World of Warcraft addon.
//
// It never contacts Slack or any server: it reads what Slack already shows
// through the operating system and writes a tiny file into the addon folder.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"slacker/companion/internal/autostart"
	"slacker/companion/internal/bridge"
	"slacker/companion/internal/source"
)

var version = "dev"

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

type config struct {
	addonDirs multiFlag
	wowDirs   multiFlag
	interval  time.Duration
	verbose   bool
	logFile   string
}

func usage() {
	fmt.Fprintf(os.Stderr, `slacker-companion %s

Mirrors your Slack desktop notification count into the Slacker WoW addon.
Local only: no Slack login, no network.

Usage:
  slacker-companion [run] [flags]     watch notifications (default)
  slacker-companion doctor [flags]    check setup and show what is detected
  slacker-companion demo [flags]      cycle fake counts to test the in-game badge
  slacker-companion set N|off [flags] write one value and exit
  slacker-companion autostart on|off [flags]
                                          start (or stop starting) at login
  slacker-companion version

Flags:
`, version)
	newFlagSet("").PrintDefaults()
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = usage
	return fs
}

func parseFlags(args []string, cfg *config) ([]string, error) {
	fs := newFlagSet("slacker-companion")
	fs.Var(&cfg.addonDirs, "addon-dir", "path to the installed Slacker addon folder (repeatable)")
	fs.Var(&cfg.wowDirs, "wow-dir", `World of Warcraft folder, the one containing _forever_ (repeatable)`)
	def := time.Second
	if runtime.GOOS == "darwin" {
		def = 2 * time.Second
	}
	fs.DurationVar(&cfg.interval, "interval", def, "how often to check for notifications")
	fs.BoolVar(&cfg.verbose, "v", false, "verbose logging")
	fs.StringVar(&cfg.logFile, "log", "", "write the log to this file")
	// Allow flags before and after positional arguments ("set 3 --wow-dir X").
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		positional, args = append(positional, args[0]), args[1:]
	}
	if cfg.interval < 250*time.Millisecond {
		cfg.interval = 250 * time.Millisecond
	}
	return positional, nil
}

func main() {
	args := os.Args[1:]
	cmd := "run"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		usage()
		return
	}
	var cfg config
	rest, err := parseFlags(args, &cfg)
	if err == flag.ErrHelp {
		return
	}
	if err != nil {
		os.Exit(2)
	}
	logger := setupLog(cfg)

	switch cmd {
	case "run":
		err = run(cfg, logger, len(os.Args) == 1)
	case "doctor":
		err = doctor(cfg)
	case "demo":
		err = demo(cfg)
	case "set":
		err = set(cfg, rest)
	case "autostart":
		err = setAutostart(cfg, rest)
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		if !hasConsole() {
			logger.Printf("error: %v", err) // the log is the only trace when started at login
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		showError(err)
		os.Exit(1)
	}
}

func setupLog(cfg config) *log.Logger {
	var w io.Writer = os.Stderr
	path := cfg.logFile
	if path == "" && !hasConsole() {
		path = defaultLogPath()
	}
	if path != "" {
		os.MkdirAll(filepath.Dir(path), 0o755)
		if st, err := os.Stat(path); err == nil && st.Size() > 1<<20 {
			os.Rename(path, path+".old") // keep the log small
		}
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			w = f
			if hasConsole() {
				w = io.MultiWriter(os.Stderr, f)
			}
		}
	}
	return log.New(w, "", log.LstdFlags)
}

func defaultLogPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "Slacker", "companion.log")
}

func discover(cfg config) []string {
	return bridge.Discover(cfg.addonDirs, cfg.wowDirs)
}

const noAddonHelp = `no Slacker addon folder found.
Install the addon into <WoW>/_forever_/Interface/AddOns/Slacker, or pass
--wow-dir "<path to World of Warcraft>" or --addon-dir "<path to the Slacker folder>"`

// publisher writes the current code into every addon folder it knows about,
// picking up newly installed copies as it goes.
type publisher struct {
	cfg     config
	log     *log.Logger
	dirs    map[string]int // dir -> code last written
	scanned time.Time
}

func (p *publisher) publish(code int) {
	if time.Since(p.scanned) > 30*time.Second {
		p.scanned = time.Now()
		found := discover(p.cfg)
		next := map[string]int{}
		for _, d := range found {
			if c, ok := p.dirs[d]; ok {
				next[d] = c
			} else {
				next[d] = -1
				p.log.Printf("bridging to %s", d)
			}
		}
		if len(found) == 0 && len(p.dirs) > 0 {
			p.log.Printf("lost all addon folders; %s", noAddonHelp)
		}
		p.dirs = next
	}
	for d, last := range p.dirs {
		if last == code {
			continue
		}
		if err := bridge.WriteCode(d, code); err != nil {
			p.log.Printf("writing %s: %v", d, err)
			continue
		}
		p.dirs[d] = code
	}
}

func run(cfg config, logger *log.Logger, interactiveLaunch bool) error {
	release, ok := singleInstance()
	if !ok {
		return fmt.Errorf("the companion is already running")
	}
	defer release()

	dirs := discover(cfg)
	if len(dirs) == 0 {
		return fmt.Errorf("%s", noAddonHelp)
	}
	src := source.New(source.Options{Interval: cfg.interval, Log: logger, Verbose: cfg.verbose})
	logger.Printf("slacker-companion %s watching %s", version, src.Name())
	for _, d := range dirs {
		logger.Printf("bridging to %s", d)
	}
	if interactiveLaunch {
		notifyStarted(len(dirs))
	}

	pub := &publisher{cfg: cfg, log: logger, dirs: map[string]int{}, scanned: time.Now()}
	for _, d := range dirs {
		pub.dirs[d] = -1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	counts := make(chan int, 16)
	errc := make(chan error, 1)
	go func() {
		errc <- src.Run(ctx, func(n int) {
			select {
			case counts <- n:
			default: // a newer value will follow
			}
		})
	}()

	last := -1
	pub.publish(bridge.CodeForCount(0))
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case n := <-counts:
			if n != last {
				logger.Printf("slack notifications: %d", n)
				last = n
			}
			pub.publish(bridge.CodeForCount(n))
		case <-heartbeat.C:
			// Rescan for new addon installs and rewrite if something reset the file.
			if last >= 0 {
				pub.publish(bridge.CodeForCount(last))
			}
		case err := <-errc:
			pub.publish(bridge.CodeOffline)
			if err != nil {
				return err
			}
			return nil
		case <-ctx.Done():
			logger.Printf("stopping")
			pub.scanned = time.Now()
			pub.publish(bridge.CodeOffline)
			return nil
		}
	}
}

func doctor(cfg config) error {
	fmt.Printf("slacker-companion %s on %s/%s\n\n", version, runtime.GOOS, runtime.GOARCH)
	src := source.New(source.Options{Interval: cfg.interval, Log: log.New(io.Discard, "", 0)})
	fmt.Printf("Notification source: %s\n", src.Name())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fmt.Print(indent(src.Diagnose(ctx)))

	fmt.Println("\nAddon folders:")
	dirs := discover(cfg)
	if len(dirs) == 0 {
		fmt.Println(indent(noAddonHelp))
	}
	for _, d := range dirs {
		code, err := bridge.ReadCode(d)
		state := describe(code)
		if err != nil {
			state = err.Error()
		}
		fmt.Printf("  %s\n    bridge file says: %s\n", d, state)
	}
	fmt.Printf("\nLog file: %s\n", defaultLogPath())
	return nil
}

func describe(code int) string {
	switch {
	case code == bridge.CodeOffline:
		return "companion offline"
	case code-1 > bridge.MaxCount:
		return "99+ notifications"
	default:
		return fmt.Sprintf("%d notification(s)", code-1)
	}
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return "  " + strings.Join(lines, "\n  ") + "\n"
}

func writeAll(cfg config, code int) error {
	dirs := discover(cfg)
	if len(dirs) == 0 {
		return fmt.Errorf("%s", noAddonHelp)
	}
	for _, d := range dirs {
		if err := bridge.WriteCode(d, code); err != nil {
			return err
		}
	}
	return nil
}

func set(cfg config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: set N|off")
	}
	code := bridge.CodeOffline
	if args[0] != "off" {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 0 {
			return fmt.Errorf("not a count: %q", args[0])
		}
		code = bridge.CodeForCount(n)
	}
	if err := writeAll(cfg, code); err != nil {
		return err
	}
	fmt.Printf("wrote: %s\n", describe(code))
	return nil
}

func demo(cfg config) error {
	if _, ok := singleInstance(); !ok {
		return fmt.Errorf("stop the running companion before using demo")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	fmt.Println("Cycling fake counts; watch the Slacker icon in game. Ctrl+C to stop.")
	steps := []int{0, 1, 2, 3, 7, 12, 42, 99, 150, 0}
	for _, n := range steps {
		if err := writeAll(cfg, bridge.CodeForCount(n)); err != nil {
			return err
		}
		fmt.Printf("  now showing %s\n", describe(bridge.CodeForCount(n)))
		select {
		case <-ctx.Done():
			return writeAll(cfg, bridge.CodeOffline)
		case <-time.After(4 * time.Second):
		}
	}
	fmt.Println("Done. If the badge never changed, see docs/TROUBLESHOOTING.md.")
	return writeAll(cfg, bridge.CodeOffline)
}

func setAutostart(cfg config, args []string) error {
	if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
		return fmt.Errorf("usage: autostart on|off")
	}
	if args[0] == "off" {
		if err := autostart.Disable(); err != nil {
			return err
		}
		fmt.Println("autostart disabled")
		return nil
	}
	if len(discover(cfg)) == 0 {
		return fmt.Errorf("%s", noAddonHelp)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	runArgs := []string{"run"}
	for _, d := range cfg.addonDirs {
		abs, _ := filepath.Abs(d)
		runArgs = append(runArgs, "--addon-dir", abs)
	}
	for _, d := range cfg.wowDirs {
		abs, _ := filepath.Abs(d)
		runArgs = append(runArgs, "--wow-dir", abs)
	}
	where, err := autostart.Enable(exe, runArgs)
	if err != nil {
		return err
	}
	fmt.Printf("autostart enabled: %s\n", where)
	return nil
}
