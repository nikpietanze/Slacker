// Package bridge carries the notification count from the companion into the
// WoW addon.
//
// WoW addons run in a sandbox with no file, socket or process access. The one
// thing an addon can learn about a file on disk is the native pixel size of a
// texture (Texture:GetSize() on an unsized texture). The companion therefore
// encodes the count into the width and height of a tiny, blank TGA file that
// lives inside the addon folder, and the addon periodically re-loads it.
//
// Encoding: every value ("code") maps to a pair of exponents (a, b) and the
// texture is 2^(a+1) x 2^(b+1) pixels. Pairs are enumerated diagonal by
// diagonal (a+b = 0, 1, 2, ...) so small counts produce tiny files.
//
//	code 0         -> companion offline / no data (2x2, the shipped placeholder)
//	code 1..100    -> count 0..99
//	code 101       -> 100 or more (shown as "99+")
//
// The same table is built in Bridge.lua at the repo root; keep them in sync.
package bridge

const (
	// MaxExp bounds each axis to 2^(MaxExp+1) = 2048 px.
	MaxExp = 10
	// MaxCount is the largest exact count; anything above is "MaxCount+1"
	// which the addon renders as "99+".
	MaxCount = 99
	// CodeOffline means the companion is not running.
	CodeOffline = 0
	numCodes    = MaxCount + 3 // offline + 0..99 + 100+
)

type pair struct{ A, B int }

var table = buildTable()

func buildTable() []pair {
	t := make([]pair, 0, numCodes)
	for s := 0; len(t) < numCodes; s++ {
		for a := 0; a <= s && len(t) < numCodes; a++ {
			b := s - a
			if a > MaxExp || b > MaxExp {
				continue
			}
			t = append(t, pair{a, b})
		}
	}
	return t
}

// CodeForCount maps a notification count to a bridge code.
func CodeForCount(count int) int {
	if count < 0 {
		count = 0
	}
	if count > MaxCount+1 {
		count = MaxCount + 1
	}
	return count + 1
}

// Dimensions returns the texture width and height for a code.
func Dimensions(code int) (w, h int) {
	if code < 0 || code >= len(table) {
		code = CodeOffline
	}
	p := table[code]
	return 1 << (p.A + 1), 1 << (p.B + 1)
}

// Decode is the inverse of Dimensions; ok is false for sizes not in the table.
func Decode(w, h int) (code int, ok bool) {
	for i, p := range table {
		if 1<<(p.A+1) == w && 1<<(p.B+1) == h {
			return i, true
		}
	}
	return 0, false
}
