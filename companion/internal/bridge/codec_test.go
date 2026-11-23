package bridge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	if len(table) != numCodes {
		t.Fatalf("table has %d codes, want %d", len(table), numCodes)
	}
	for code := 0; code < numCodes; code++ {
		w, h := Dimensions(code)
		got, ok := Decode(w, h)
		if !ok || got != code {
			t.Fatalf("code %d -> %dx%d -> %d,%v", code, w, h, got, ok)
		}
		if w > 2048 || h > 2048 {
			t.Fatalf("code %d too large: %dx%d", code, w, h)
		}
	}
}

func TestSmallCountsAreTiny(t *testing.T) {
	if w, h := Dimensions(CodeOffline); w != 2 || h != 2 {
		t.Fatalf("offline must match the shipped 2x2 placeholder, got %dx%d", w, h)
	}
	for n := 0; n <= 10; n++ {
		if w, h := Dimensions(CodeForCount(n)); w*h > 256 {
			t.Errorf("count %d uses %dx%d", n, w, h)
		}
	}
	maxArea := 0
	for code := 0; code < numCodes; code++ {
		w, h := Dimensions(code)
		if w*h > maxArea {
			maxArea = w * h
		}
	}
	t.Logf("largest bridge file: %d bytes", 18+maxArea)
}

func TestCodeForCountClamps(t *testing.T) {
	if CodeForCount(-5) != CodeForCount(0) {
		t.Error("negative counts must clamp to 0")
	}
	if CodeForCount(100) != CodeForCount(5000) || CodeForCount(100) != numCodes-1 {
		t.Error("counts above 99 must share the 99+ code")
	}
	if CodeForCount(99) == CodeForCount(100) {
		t.Error("99 must be distinct from 99+")
	}
}

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []int{0, 3, 99, 150} {
		if err := WriteCode(dir, CodeForCount(n)); err != nil {
			t.Fatal(err)
		}
		code, err := ReadCode(dir)
		if err != nil || code != CodeForCount(n) {
			t.Fatalf("count %d: read %d, %v", n, code, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "Bridge", "count.tga.tmp")); !os.IsNotExist(err) {
		t.Error("temp file left behind")
	}
}

// The addon ships Bridge/count.tga; it must decode as "offline".
func TestShippedPlaceholder(t *testing.T) {
	addon := filepath.Join("..", "..", "..") // the addon lives at the repo root
	code, err := ReadCode(addon)
	if err != nil {
		t.Skipf("addon placeholder not found: %v", err)
	}
	if code != CodeOffline {
		t.Fatalf("shipped placeholder decodes to %d, want offline", code)
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
