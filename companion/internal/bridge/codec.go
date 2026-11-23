// Package bridge carries the notification count from the companion into the
// WoW addon.
//
// WoW addons run in a sandbox with no file, socket or process access. The one
// thing an addon can learn about a file on disk is a texture's pixel size
// (Texture:GetSize() on an unsized texture). So the companion writes tiny
// blank images whose sizes encode the count, and the addon loads them.
//
// The client caches a texture until a periodic cleanup frees unused ones
// (every few seconds to a minute or more, observed on 1.60.1), so loading the
// same file again usually returns the old size. A file that hasn't been
// loaded since the last cleanup is always read fresh. The companion therefore
// writes the count into a ring of Slots identical slots, and the addon reads
// the next slot each second: by the time it returns to a slot, a cleanup has
// almost certainly freed it.
//
// Each slot is two images, a tens digit and a ones digit, each holding a
// digit 0..10 (10 is a marker):
//
//	tens 0..9, ones 0..9  -> count 0..99
//	tens 10,   ones 1     -> 100 or more ("99+")
//	tens 10,   ones 10    -> companion offline (the shipped placeholder)
//
// Digit d maps to the d-th exponent pair (a, b), enumerated diagonal by
// diagonal (a+b = 0, 1, 2, ...), and the image is 2^(a+1) x 2^(b+1) pixels,
// so the largest is 64 pixels (256 bytes).
//
// The same table is built in Bridge.lua at the repo root; keep them in sync.
package bridge

import "fmt"

const (
	// Slots is how many slots the addon cycles through, one per second.
	Slots = 128
	// MaxCount is the largest exact count; anything above shows as "99+".
	MaxCount = 99
	// Marker is the digit value used for "99+" and offline.
	Marker = 10
	// CodeOffline means the companion is not running.
	CodeOffline = 0
	numDigits   = Marker + 1
)

// Codes: 0 offline, 1..100 count 0..99, 101 for 100 or more.

// CodeForCount maps a notification count to a code.
func CodeForCount(count int) int {
	if count < 0 {
		count = 0
	}
	if count > MaxCount+1 {
		count = MaxCount + 1
	}
	return count + 1
}

type pair struct{ A, B int }

var digitTable = buildDigitTable()

func buildDigitTable() []pair {
	t := make([]pair, 0, numDigits)
	for s := 0; len(t) < numDigits; s++ {
		for a := 0; a <= s && len(t) < numDigits; a++ {
			t = append(t, pair{a, s - a})
		}
	}
	return t
}

// Digits splits a code into the tens and ones digits written to a slot.
func Digits(code int) (tens, ones int) {
	switch {
	case code <= CodeOffline:
		return Marker, Marker
	case code-1 > MaxCount:
		return Marker, 1
	default:
		return (code - 1) / 10, (code - 1) % 10
	}
}

// CodeFromDigits is the inverse of Digits; ok is false for combinations
// Digits never produces.
func CodeFromDigits(tens, ones int) (code int, ok bool) {
	switch {
	case tens == Marker && ones == Marker:
		return CodeOffline, true
	case tens == Marker && ones == 1:
		return CodeForCount(MaxCount + 1), true
	case tens >= 0 && tens < Marker && ones >= 0 && ones < Marker:
		return CodeForCount(tens*10 + ones), true
	}
	return 0, false
}

// DigitDims returns the image size for a digit 0..10.
func DigitDims(d int) (w, h int) {
	p := digitTable[d]
	return 1 << (p.A + 1), 1 << (p.B + 1)
}

// DigitFromDims is the inverse of DigitDims.
func DigitFromDims(w, h int) (int, bool) {
	for d := range digitTable {
		if dw, dh := DigitDims(d); dw == w && dh == h {
			return d, true
		}
	}
	return 0, false
}

// SlotFiles returns the two file names (tens, ones) of a slot 1..Slots,
// relative to the addon's Bridge folder.
func SlotFiles(slot int) (tens, ones string) {
	return fmt.Sprintf("s%03dt.tga", slot), fmt.Sprintf("s%03do.tga", slot)
}
