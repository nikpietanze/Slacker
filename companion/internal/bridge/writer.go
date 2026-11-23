package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AddonName is the addon folder the companion writes into.
const AddonName = "Slacker"

// BridgeDir is the folder inside the addon that holds the slot files. The
// addon loads them as Interface\AddOns\Slacker\Bridge\sNNNt.tga / sNNNo.tga.
const BridgeDir = "Bridge"

// digitImages caches the encoded image for each digit.
var digitImages = func() [numDigits][]byte {
	var imgs [numDigits][]byte
	for d := range imgs {
		w, h := DigitDims(d)
		imgs[d] = BlankTGA(w, h)
	}
	return imgs
}()

// WriteCode writes code into every slot of one addon directory.
func WriteCode(addonDir string, code int) error {
	dir := filepath.Join(addonDir, BridgeDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tens, ones := Digits(code)
	for slot := 1; slot <= Slots; slot++ {
		tf, of := SlotFiles(slot)
		// Ones first: a reader that catches a slot half-written sees the new
		// ones digit with the old tens, and the addon waits for two
		// matching reads before believing a value.
		if err := writeFile(filepath.Join(dir, of), digitImages[ones]); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, tf), digitImages[tens]); err != nil {
			return err
		}
	}
	return nil
}

// writeFile replaces dst atomically (temp file + rename) so WoW never reads
// a half-written image. On Windows the rename can briefly fail while the game
// has the file open, so it retries before falling back to a direct write.
func writeFile(dst string, data []byte) error {
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	var err error
	for i := 0; i < 10; i++ {
		if err = os.Rename(tmp, dst); err == nil {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	os.Remove(tmp)
	if werr := os.WriteFile(dst, data, 0o644); werr != nil {
		return fmt.Errorf("rename: %v; direct write: %w", err, werr)
	}
	return nil
}

func readDims(path string) (w, h int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	hdr := make([]byte, 18)
	if _, err := f.Read(hdr); err != nil {
		return 0, 0, err
	}
	return int(hdr[12]) | int(hdr[13])<<8, int(hdr[14]) | int(hdr[15])<<8, nil
}

// ReadCode reports the code stored in slot 1 of an addon directory.
func ReadCode(addonDir string) (int, error) {
	tf, of := SlotFiles(1)
	var digits [2]int
	for i, name := range []string{tf, of} {
		w, h, err := readDims(filepath.Join(addonDir, BridgeDir, name))
		if err != nil {
			return 0, err
		}
		d, ok := DigitFromDims(w, h)
		if !ok {
			return 0, fmt.Errorf("unexpected image size %dx%d in %s", w, h, name)
		}
		digits[i] = d
	}
	code, ok := CodeFromDigits(digits[0], digits[1])
	if !ok {
		return 0, fmt.Errorf("unexpected digits %d,%d", digits[0], digits[1])
	}
	return code, nil
}
