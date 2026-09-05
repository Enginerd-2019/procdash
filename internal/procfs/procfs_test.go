package procfs

import (
	"errors"
	"io/fs"
	"testing"
)

func TestPathBuildsUnderInjectedRoot(t *testing.T) {
	got := NewFS("testdata/basic").Proc(600314).path("status")
	want := "testdata/basic/600314/status"
	if got != want {
		t.Errorf("path() = %q, want %q", got, want)
	}
}

func TestAbsentProcessIsErrGone(t *testing.T) {
	_, err := NewFS("testdata/basic").Proc(999999).Status()
	if !errors.Is(err, ErrGone) {
		t.Fatalf("want ErrGone, got %v", err)
	}
	if errors.Is(err, fs.ErrPermission) {
		t.Fatalf("a vanished process must not classify as permission denied: %v", err)
	}
}

func TestAbsentRootIsErrGone(t *testing.T) {
	// A missing fixture tree looks identical to a missing process: the
	// directory isn't there. Callers still get the ErrGone contract.
	_, err := NewFS("testdata/does-not-exist").Proc(1).Status()
	if !errors.Is(err, ErrGone) {
		t.Fatalf("want ErrGone, got %v", err)
	}
}
