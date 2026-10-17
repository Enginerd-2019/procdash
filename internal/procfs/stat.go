package procfs

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// userHZ is USER_HZ, the tick rate the kernel uses for the cumulative
// CPU counters in /proc/<pid>/stat. It is a fixed userspace ABI (100
// on every mainstream architecture) — distinct from the kernel's
// internal CONFIG_HZ — and Go has no sysconf(3), so it is a documented
// constant; the runbook's `getconf CLK_TCK` check guards it.
const userHZ = 100

// maxField is the highest man-page field number this parser reads
// (rss, field 24). A file with fewer fields is truncated.
const maxField = 24

// Stat is the parsed subset of /proc/<pid>/stat (proc_pid_stat(5)):
// one line, space-separated, position-defined. Field names carry the
// kernel's units verbatim; conversions live in the methods below.
type Stat struct {
	PID   int
	Comm  string // field 2, parens stripped; process-controlled — may contain spaces and ')'
	State byte   // field 3: 'R','S','D','Z','T','t','I','X'
	PPID  int    // field 4

	UTimeTicks uint64 // field 14 — cumulative user-mode CPU, USER_HZ ticks
	STimeTicks uint64 // field 15 — cumulative kernel-mode CPU, USER_HZ ticks

	NumThreads     int64  // field 20 — cross-check against Status.Threads
	StartTimeTicks uint64 // field 22 — ticks after boot: age anchor, and identity when PIDs recycle
	VSizeBytes     uint64 // field 23 — virtual size, honestly in bytes
	RSSPages       int64  // field 24 — resident pages; RSSBytes() converts
}

// Stat reads and parses /proc/<pid>/stat.
func (p Proc) Stat() (Stat, error) {
	data, err := p.readFile("stat")
	if err != nil {
		return Stat{}, err
	}
	st, err := parseStat(data)
	if err != nil {
		return Stat{}, fmt.Errorf("parse %s: %w", p.path("stat"), err)
	}
	return st, nil
}

// parseStat is pure: bytes in, Stat out. comm (field 2) is the text
// between the FIRST '(' and the LAST ')' — comm is process-controlled
// and may itself contain spaces and parentheses, so splitting the
// whole line on whitespace mis-indexes every later field. Everything
// after the last ')' is kernel-generated and safely positional.
func parseStat(data []byte) (Stat, error) {
	open := bytes.IndexByte(data, '(')
	close := bytes.LastIndexByte(data, ')')
	if open < 0 || close < 0 || close < open {
		return Stat{}, fmt.Errorf("no parenthesized comm field")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data[:open])))
	if err != nil {
		return Stat{}, fmt.Errorf("pid (field 1): %w", err)
	}

	// rest[0] is man-page field 3 (state), so field N lives at index
	// N−3. That arithmetic exists only inside uintField/int64Field;
	// nothing else indexes rest.
	rest := strings.Fields(string(data[close+1:]))
	if len(rest) < maxField-2 {
		return Stat{}, fmt.Errorf("truncated: %d fields after comm, need %d", len(rest), maxField-2)
	}
	if len(rest[0]) != 1 {
		return Stat{}, fmt.Errorf("state (field 3) %q: want a single letter", rest[0])
	}

	st := Stat{PID: pid, Comm: string(data[open+1 : close]), State: rest[0][0]}

	ppid, err := int64Field(rest, 4, "ppid")
	if err != nil {
		return Stat{}, err
	}
	st.PPID = int(ppid)

	if st.UTimeTicks, err = uintField(rest, 14, "utime"); err != nil {
		return Stat{}, err
	}
	if st.STimeTicks, err = uintField(rest, 15, "stime"); err != nil {
		return Stat{}, err
	}
	if st.NumThreads, err = int64Field(rest, 20, "num_threads"); err != nil {
		return Stat{}, err
	}
	if st.StartTimeTicks, err = uintField(rest, 22, "starttime"); err != nil {
		return Stat{}, err
	}
	if st.VSizeBytes, err = uintField(rest, 23, "vsize"); err != nil {
		return Stat{}, err
	}
	if st.RSSPages, err = int64Field(rest, 24, "rss"); err != nil {
		return Stat{}, err
	}
	return st, nil
}

// uintField parses man-page field n (unsigned) from the post-comm
// fields. The N−3 offset lives here and in int64Field only.
func uintField(rest []string, n int, name string) (uint64, error) {
	v, err := strconv.ParseUint(rest[n-3], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s (field %d) %q: %w", name, n, rest[n-3], err)
	}
	return v, nil
}

// int64Field parses man-page field n (signed — the kernel prints
// these with %ld) from the post-comm fields.
func int64Field(rest []string, n int, name string) (int64, error) {
	v, err := strconv.ParseInt(rest[n-3], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s (field %d) %q: %w", name, n, rest[n-3], err)
	}
	return v, nil
}

// ticksToDuration converts USER_HZ ticks to a duration — the only
// place tick arithmetic lives.
func ticksToDuration(ticks uint64) time.Duration {
	return time.Duration(ticks) * time.Second / userHZ
}

// UserTime is cumulative user-mode CPU time.
func (s Stat) UserTime() time.Duration { return ticksToDuration(s.UTimeTicks) }

// SystemTime is cumulative kernel-mode CPU time.
func (s Stat) SystemTime() time.Duration { return ticksToDuration(s.STimeTicks) }

// CPUTime is total cumulative CPU time (user + system).
func (s Stat) CPUTime() time.Duration { return ticksToDuration(s.UTimeTicks + s.STimeTicks) }

// RSSBytes is the resident set in bytes, using the runtime page size.
func (s Stat) RSSBytes() uint64 { return uint64(s.RSSPages) * uint64(os.Getpagesize()) }
