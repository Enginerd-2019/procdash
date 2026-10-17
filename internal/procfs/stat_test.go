package procfs

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// Fixture-driven: the full FS → Proc → Stat path against committed
// captures, every field pinned to values derived from the raw file by
// hand (not by this parser).
func TestStatFixtures(t *testing.T) {
	tests := []struct {
		name string
		root string
		pid  int
		want Stat
	}{
		{
			name: "live-captured shell process",
			root: "testdata/basic",
			pid:  1726633,
			want: Stat{
				PID: 1726633, Comm: "sh", State: 'R', PPID: 1726630,
				UTimeTicks: 0, STimeTicks: 0, NumThreads: 1,
				StartTimeTicks: 43395608, VSizeBytes: 2879488, RSSPages: 437,
			},
		},
		{
			name: "hostile comm with space and parens",
			root: "testdata/hostile-comm",
			pid:  1729364,
			want: Stat{
				PID: 1729364, Comm: "we) (ird", State: 'S', PPID: 1729361,
				UTimeTicks: 0, STimeTicks: 0, NumThreads: 1,
				StartTimeTicks: 43402561, VSizeBytes: 7577600, RSSPages: 911,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFS(tt.root).Proc(tt.pid).Stat()
			if err != nil {
				t.Fatalf("Stat(): %v", err)
			}
			if got != tt.want {
				t.Errorf("Stat()\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

// A malformed file reaches the caller as a loud parse error naming
// the file — the wrap in Stat(). The fixture is hand-crafted (see
// testdata/README.md): no kernel emits it.
func TestStatParseErrorNamesFile(t *testing.T) {
	_, err := NewFS("testdata/truncated").Proc(1).Stat()
	if err == nil {
		t.Fatal("Stat(): want error, got nil")
	}
	for _, must := range []string{"parse", "truncated/1/stat", "truncated:"} {
		if !strings.Contains(err.Error(), must) {
			t.Errorf("error %q does not mention %q", err, must)
		}
	}
}

func TestStatGone(t *testing.T) {
	_, err := NewFS("testdata/basic").Proc(999999).Stat()
	if !errors.Is(err, ErrGone) {
		t.Fatalf("Stat() error = %v, want ErrGone", err)
	}
}

// Parse-level happy paths over crafted lines: 22 post-comm fields,
// values chosen so every extracted field is distinct and checkable.
func TestParseStatInline(t *testing.T) {
	const tail = " S 1 7 7 0 -1 0 0 0 0 0 5 3 0 0 20 0 2 0 100 4096 9\n"
	tests := []struct {
		name  string
		input string
		want  Stat
	}{
		{"plain comm", "7 (x)" + tail, Stat{
			PID: 7, Comm: "x", State: 'S', PPID: 1,
			UTimeTicks: 5, STimeTicks: 3, NumThreads: 2,
			StartTimeTicks: 100, VSizeBytes: 4096, RSSPages: 9,
		}},
		{"comm with parens and spaces inline", "7 (a) (b)" + tail, Stat{
			PID: 7, Comm: "a) (b", State: 'S', PPID: 1,
			UTimeTicks: 5, STimeTicks: 3, NumThreads: 2,
			StartTimeTicks: 100, VSizeBytes: 4096, RSSPages: 9,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStat([]byte(tt.input))
			if err != nil {
				t.Fatalf("parseStat(): %v", err)
			}
			if got != tt.want {
				t.Errorf("parseStat()\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

func TestParseStatErrors(t *testing.T) {
	// good is a valid line; each case mutates one aspect of it.
	const good = "7 (x) S 1 7 7 0 -1 0 0 0 0 0 5 3 0 0 20 0 2 0 100 4096 9"
	tests := []struct {
		name    string
		input   string
		wantErr string // substring the error must contain
	}{
		{"empty input", "", "no parenthesized comm"},
		{"no parens at all", "1234 sh R 1 2 3", "no parenthesized comm"},
		{"reversed parens", "1234 )sh( R 1 2 3", "no parenthesized comm"},
		{"non-numeric pid", strings.Replace(good, "7 (x)", "seven (x)", 1), "pid (field 1)"},
		{"truncated after comm", "7 (x) S 1 7 7 0 -1 0", "truncated"},
		{"multi-char state", strings.Replace(good, "(x) S", "(x) SS", 1), "state (field 3)"},
		{"non-numeric ppid", strings.Replace(good, "S 1 7", "S one 7", 1), "ppid (field 4)"},
		{"non-numeric utime", strings.Replace(good, "0 5 3 0", "0 lots 3 0", 1), "utime (field 14)"},
		{"non-numeric stime", strings.Replace(good, " 5 3 ", " 5 bad ", 1), "stime (field 15)"},
		{"non-numeric num_threads", strings.Replace(good, " 2 0 100", " two 0 100", 1), "num_threads (field 20)"},
		{"non-numeric starttime", strings.Replace(good, "100 4096", "1e2 4096", 1), "starttime (field 22)"},
		{"non-numeric vsize", strings.Replace(good, "4096 9", "4k96 9", 1), "vsize (field 23)"},
		{"non-numeric rss", strings.Replace(good, "4096 9", "4096 nine", 1), "rss (field 24)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseStat([]byte(tt.input))
			if err == nil {
				t.Fatalf("parseStat(%q): want error, got nil", tt.input)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}

// Conversion methods are pure arithmetic over the struct; exact values.
func TestStatConversions(t *testing.T) {
	s := Stat{UTimeTicks: 150, STimeTicks: 50, RSSPages: 3}
	if got, want := s.UserTime(), 1500*time.Millisecond; got != want {
		t.Errorf("UserTime() = %v, want %v", got, want)
	}
	if got, want := s.SystemTime(), 500*time.Millisecond; got != want {
		t.Errorf("SystemTime() = %v, want %v", got, want)
	}
	if got, want := s.CPUTime(), 2*time.Second; got != want {
		t.Errorf("CPUTime() = %v, want %v", got, want)
	}
	if got, want := s.RSSBytes(), uint64(3*os.Getpagesize()); got != want {
		t.Errorf("RSSBytes() = %d, want %d", got, want)
	}
}
