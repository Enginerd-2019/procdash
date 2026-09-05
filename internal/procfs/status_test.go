package procfs

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"
)

// Fixture-driven: the full FS → Proc → Status path against committed
// trees. Every field is pinned to the captured file's known values —
// a test that would pass against an empty implementation is not a
// test.
func TestStatusFixtures(t *testing.T) {
	tests := []struct {
		name string
		root string
		pid  int
		want Status
	}{
		{
			name: "live-captured shell process",
			root: "testdata/basic",
			pid:  600314,
			want: Status{
				Name: "sh", State: 'S', StateDesc: "sleeping",
				PID: 600314, PPID: 600306, TGID: 600314,
				UID:     [4]uint32{1000, 1000, 1000, 1000},
				GID:     [4]uint32{1000, 1000, 1000, 1000},
				Threads: 1, FDSize: 64,
				HasVM:     true,
				VmPeakKiB: 2812, VmSizeKiB: 2812,
				VmHWMKiB: 1848, VmRSSKiB: 1848, VmSwapKiB: 0,
			},
		},
		{
			name: "kernel thread has no Vm block",
			root: "testdata/kthread",
			pid:  2,
			want: Status{
				Name: "kthreadd", State: 'S', StateDesc: "sleeping",
				PID: 2, PPID: 0, TGID: 2,
				UID:     [4]uint32{0, 0, 0, 0},
				GID:     [4]uint32{0, 0, 0, 0},
				Threads: 1, FDSize: 64,
				HasVM: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFS(tt.root).Proc(tt.pid).Status()
			if err != nil {
				t.Fatalf("Status(): %v", err)
			}
			if got != tt.want {
				t.Errorf("Status()\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

// Parse-level: parseStatus is pure over bytes, so edge cases need no
// fixture files — hand-crafted inputs go straight in.
func TestParseStatusErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string // substring the error must contain
	}{
		{"empty file", "", "required keys missing"},
		{"unrelated file", "MemTotal:       16299 kB\n", "required keys missing"},
		{"uid with wrong count",
			"Name:\tx\nState:\tR (running)\nPid:\t1\nUid:\t1000\t1000\n", "want 4 ids"},
		{"uid with five fields",
			"Name:\tx\nState:\tR (running)\nPid:\t1\nUid:\t1 2 3 4 5\n", "want 4 ids"},
		{"uid with non-numeric id",
			"Name:\tx\nState:\tR (running)\nPid:\t1\nUid:\t1000 1000 x 1000\n", "invalid syntax"},
		{"non-numeric vm value",
			"Name:\tx\nState:\tR (running)\nPid:\t1\nVmRSS:\tlots kB\n", "VmRSS"},
		{"empty vm value",
			"Name:\tx\nState:\tR (running)\nPid:\t1\nVmRSS:\n", "empty value"},
		{"empty state",
			"Name:\tx\nState:\t\nPid:\t1\n", "State"},
		{"non-numeric pid",
			"Name:\tx\nState:\tR (running)\nPid:\tseven\n", "Pid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseStatus([]byte(tt.input))
			if err == nil {
				t.Fatalf("parseStatus(%q): want error, got nil", tt.input)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}

// A scanner can fail, not just end: a single line longer than
// bufio.Scanner's token limit aborts the scan, and parseStatus must
// surface that error rather than return a partial success — the
// sc.Err() check after the loop.
func TestParseStatusScannerError(t *testing.T) {
	input := append([]byte("Name:\tx\n"), bytes.Repeat([]byte("a"), bufio.MaxScanTokenSize+1)...)
	_, err := parseStatus(input)
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Fatalf("parseStatus() error = %v, want bufio.ErrTooLong", err)
	}
}

// Tolerance: unknown keys and colon-free lines are ignored — kernels
// add fields over time and a parser that panics on novelty is a
// maintenance bomb.
func TestParseStatusToleratesUnknownKeys(t *testing.T) {
	input := "Name:\tweird\n" +
		"SomeFutureKey:\t42\n" +
		"a line with no separator at all\n" +
		"State:\tZ (zombie)\n" +
		"Pid:\t7\n"
	got, err := parseStatus([]byte(input))
	if err != nil {
		t.Fatalf("parseStatus(): %v", err)
	}
	want := Status{Name: "weird", State: 'Z', StateDesc: "zombie", PID: 7}
	if got != want {
		t.Errorf("parseStatus()\n got: %+v\nwant: %+v", got, want)
	}
}
