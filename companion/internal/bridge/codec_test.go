package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDigitsRoundTrip(t *testing.T) {
	for code := 0; code <= CodeForCount(1000); code++ {
		tens, ones := Digits(code)
		got, ok := CodeFromDigits(tens, ones)
		if !ok || got != code {
			t.Fatalf("code %d -> %d,%d -> %d,%v", code, tens, ones, got, ok)
		}
	}
	if _, ok := CodeFromDigits(Marker, 5); ok {
		t.Error("unused marker combination decoded")
	}
}

func TestDigitImagesAreTiny(t *testing.T) {
	seen := map[[2]int]bool{}
	for d := 0; d < numDigits; d++ {
		w, h := DigitDims(d)
		if w*h > 64 {
			t.Errorf("digit %d is %dx%d", d, w, h)
		}
		if seen[[2]int{w, h}] {
			t.Errorf("digit %d reuses size %dx%d", d, w, h)
		}
		seen[[2]int{w, h}] = true
		if got, ok := DigitFromDims(w, h); !ok || got != d {
			t.Errorf("digit %d -> %dx%d -> %d", d, w, h, got)
		}
	}
	if w, h := DigitDims(Marker); w*h*4+18 > 300 {
		t.Errorf("largest digit image too big: %dx%d", w, h)
	}
}

func TestCodeForCountClamps(t *testing.T) {
	if CodeForCount(-5) != CodeForCount(0) {
		t.Error("negative counts must clamp to 0")
	}
	if CodeForCount(100) != CodeForCount(5000) {
		t.Error("counts above 99 must share the 99+ code")
	}
	if CodeForCount(99) == CodeForCount(100) {
		t.Error("99 must be distinct from 99+")
	}
}

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []int{0, 3, 42, 99, 150} {
		if err := WriteCode(dir, CodeForCount(n)); err != nil {
			t.Fatal(err)
		}
		code, err := ReadCode(dir)
		if err != nil || code != CodeForCount(n) {
			t.Fatalf("count %d: read %d, %v", n, code, err)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, BridgeDir))
	if len(entries) != 2*Slots {
		t.Errorf("want %d slot files, found %d", 2*Slots, len(entries))
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".tga" {
			t.Errorf("leftover file %s", e.Name())
		}
	}
}

// The addon ships every slot file as the "offline" placeholder.
func TestShippedPlaceholders(t *testing.T) {
	addon := filepath.Join("..", "..", "..") // the addon lives at the repo root
	if _, err := os.Stat(filepath.Join(addon, AddonName+".toc")); err != nil {
		t.Skip("not running inside the repo")
	}
	for slot := 1; slot <= Slots; slot++ {
		tf, of := SlotFiles(slot)
		for _, name := range []string{tf, of} {
			w, h, err := readDims(filepath.Join(addon, BridgeDir, name))
			if err != nil {
				t.Fatalf("missing placeholder %s: %v", name, err)
			}
			if d, ok := DigitFromDims(w, h); !ok || d != Marker {
				t.Fatalf("%s should be the offline marker, is %dx%d", name, w, h)
			}
		}
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "_forever_", "Interface", "AddOns", AddonName)
	os.MkdirAll(want, 0o755)
	os.MkdirAll(filepath.Join(root, "_retail_", "Interface", "AddOns", "Other"), 0o755)
	got := Discover(nil, []string{root})
	if len(got) != 1 || got[0] != want {
		t.Fatalf("Discover = %v, want [%s]", got, want)
	}
}
