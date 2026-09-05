// Package procfs parses the Linux /proc filesystem into typed values.
//
// The package is the project's foundation layer: it reads kernel text
// and returns structs — it never logs, prints, or formats. All access
// goes through an FS value whose root is injectable, so tests run
// against committed fixture trees instead of a live kernel (see
// testdata/).
package procfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// DefaultRoot is the standard procfs mount point.
const DefaultRoot = "/proc"

// ErrGone reports that a process exited (or never existed): its /proc
// directory or files are absent. For an inspector this is a normal
// outcome — callers skip or degrade, they don't fail. Test with
// errors.Is(err, ErrGone).
var ErrGone = errors.New("process gone")

// FS is a procfs mount point. Construct with Default or NewFS; the
// zero value points nowhere useful.
type FS struct {
	root string
}

// Default returns an FS reading the live kernel at /proc.
func Default() FS {
	return NewFS(DefaultRoot)
}

// NewFS returns an FS rooted at an arbitrary directory — the test
// seam: point it at a fixture tree that mimics /proc's layout.
func NewFS(root string) FS {
	return FS{root: root}
}

// Proc returns a handle for one process's directory. It performs no
// I/O and cannot fail; existence is checked when a method reads.
func (f FS) Proc(pid int) Proc {
	return Proc{PID: pid, fs: f}
}

// Proc is one process's /proc/<pid> directory. Methods read at call
// time — each call is a fresh, independent observation — and are safe
// for concurrent use.
type Proc struct {
	PID int
	fs  FS
}

// path returns the path of one file inside the process directory.
func (p Proc) path(name string) string {
	return filepath.Join(p.fs.root, strconv.Itoa(p.PID), name)
}

// readFile reads one whole file belonging to the process and
// classifies the failure: absent process -> ErrGone; permission denied
// passes through untouched (errors.Is(err, fs.ErrPermission)); other
// errors are unexpected and wrapped as-is. Error classification lives
// here and nowhere else.
func (p Proc) readFile(name string) ([]byte, error) {
	path := p.path(name)
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ESRCH):
		// ENOENT on open, or ESRCH from a read that raced the exit:
		// either way the process is gone — normal, not an error state.
		return nil, fmt.Errorf("read %s: %w", path, ErrGone)
	default:
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
}
