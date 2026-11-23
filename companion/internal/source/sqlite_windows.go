package source

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Minimal read-only binding to winsqlite3.dll, which ships with Windows 10 and
// 11. Using the system DLL keeps the companion free of cgo and bundled SQLite.

var (
	sqlite       = syscall.NewLazyDLL("winsqlite3.dll")
	pOpen        = sqlite.NewProc("sqlite3_open_v2")
	pClose       = sqlite.NewProc("sqlite3_close_v2")
	pBusy        = sqlite.NewProc("sqlite3_busy_timeout")
	pPrepare     = sqlite.NewProc("sqlite3_prepare_v2")
	pStep        = sqlite.NewProc("sqlite3_step")
	pFinalize    = sqlite.NewProc("sqlite3_finalize")
	pColumnText  = sqlite.NewProc("sqlite3_column_text")
	pColumnCount = sqlite.NewProc("sqlite3_column_count")
	pErrmsg      = sqlite.NewProc("sqlite3_errmsg")
	errNoSQLite  = errors.New("winsqlite3.dll not available (Windows 10 or newer is required)")
)

const (
	sqliteOK            = 0
	sqliteRow           = 100
	sqliteDone          = 101
	sqliteOpenReadOnly  = 0x1
	sqliteOpenReadWrite = 0x2
)

type db struct{ h uintptr }

func openDB(path string, flags uintptr) (*db, error) {
	if err := sqlite.Load(); err != nil {
		return nil, errNoSQLite
	}
	p, err := syscall.BytePtrFromString(path)
	if err != nil {
		return nil, err
	}
	var h uintptr
	rc, _, _ := pOpen.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&h)), flags, 0)
	if rc != sqliteOK {
		msg := "unknown error"
		if h != 0 {
			msg = cstr(callPtr(pErrmsg, h))
			pClose.Call(h)
		}
		return nil, fmt.Errorf("open %s: %s (code %d)", path, msg, rc)
	}
	pBusy.Call(h, 750)
	return &db{h}, nil
}

func (d *db) Close() { pClose.Call(d.h) }

// query runs sql and returns every row as strings.
func (d *db) query(sql string) ([][]string, error) {
	q, err := syscall.BytePtrFromString(sql)
	if err != nil {
		return nil, err
	}
	var stmt uintptr
	rc, _, _ := pPrepare.Call(d.h, uintptr(unsafe.Pointer(q)), ^uintptr(0), uintptr(unsafe.Pointer(&stmt)), 0)
	if rc != sqliteOK {
		return nil, fmt.Errorf("prepare: %s", cstr(callPtr(pErrmsg, d.h)))
	}
	defer pFinalize.Call(stmt)
	cols, _, _ := pColumnCount.Call(stmt)
	var rows [][]string
	for {
		rc, _, _ := pStep.Call(stmt)
		switch rc {
		case sqliteRow:
			row := make([]string, cols)
			for i := range row {
				row[i] = cstr(callPtr(pColumnText, stmt, uintptr(i)))
			}
			rows = append(rows, row)
		case sqliteDone:
			return rows, nil
		default:
			return rows, fmt.Errorf("step: %s (code %d)", cstr(callPtr(pErrmsg, d.h)), rc)
		}
	}
}

func callPtr(p *syscall.LazyProc, args ...uintptr) uintptr {
	r, _, _ := p.Call(args...)
	return r
}

func cstr(p uintptr) string {
	if p == 0 {
		return ""
	}
	return windows.BytePtrToString((*byte)(unsafe.Pointer(p)))
}
