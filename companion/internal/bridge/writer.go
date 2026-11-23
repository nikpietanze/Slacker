package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AddonName is the addon folder the companion writes into.
const AddonName = "SlackNotify"

// relPath is the bridge texture relative to the addon folder. The addon loads
// it as Interface\AddOns\SlackNotify\Bridge\count.tga.
var relPath = filepath.Join("Bridge", "count.tga")

// WriteCode writes the bridge texture for code into one addon directory.
// It writes to a temp file and renames it into place so WoW never reads a
// half-written file; on Windows the rename can briefly fail while the game
// has the file open, so it retries before falling back to a direct write.
func WriteCode(addonDir string, code int) error {
	w, h := Dimensions(code)
	data := GrayTGA(w, h)
	dst := filepath.Join(addonDir, relPath)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	var err error
	for i := 0; i < 10; i++ {
		if err = os.Rename(tmp, dst); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	os.Remove(tmp)
	if werr := os.WriteFile(dst, data, 0o644); werr != nil {
		return fmt.Errorf("rename: %v; direct write: %w", err, werr)
	}
	return nil
}

// ReadCode reports the code currently on disk in an addon directory.
func ReadCode(addonDir string) (int, error) {
	f, err := os.Open(filepath.Join(addonDir, relPath))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	hdr := make([]byte, 18)
	if _, err := f.Read(hdr); err != nil {
		return 0, err
	}
	w := int(hdr[12]) | int(hdr[13])<<8
	h := int(hdr[14]) | int(hdr[15])<<8
	code, ok := Decode(w, h)
	if !ok {
		return 0, fmt.Errorf("unexpected texture size %dx%d", w, h)
	}
	return code, nil
}
