package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Status is the parsed identity/state/memory view of
// /proc/<pid>/status (proc_pid_status(5)): "Key:\tvalue" lines, keyed
// by name, tolerant of unknown keys. Field names carry the kernel's
// units verbatim; conversion happens at boundaries, never here. The
// struct is deliberately comparable (arrays, no slices) so tests can
// assert with ==.
type Status struct {
	Name      string // comm, ≤15 chars; may differ from argv[0]
	State     byte   // 'R','S','D','Z','T','t','I','X'
	StateDesc string // the kernel's gloss, e.g. "sleeping"

	PID  int
	PPID int
	TGID int

	UID [4]uint32 // real, effective, saved, filesystem
	GID [4]uint32

	Threads int
	FDSize  int // fd-table slots allocated — NOT the open-fd count

	// Memory. Valid only when HasVM: kernel threads and zombies have
	// no userspace address space and omit the whole Vm* block.
	HasVM     bool
	VmPeakKiB uint64 // peak virtual size (the file's "kB" means KiB)
	VmSizeKiB uint64 // current virtual size
	VmHWMKiB  uint64 // peak RSS (high-water mark)
	VmRSSKiB  uint64 // current resident set
	VmSwapKiB uint64
}

// Status reads and parses /proc/<pid>/status.
func (p Proc) Status() (Status, error) {
	data, err := p.readFile("status")
	if err != nil {
		return Status{}, err
	}
	st, err := parseStatus(data)
	if err != nil {
		return Status{}, fmt.Errorf("parse %s: %w", p.path("status"), err)
	}
	return st, nil
}

// parseStatus is pure: bytes in, Status out. It requires the Name,
// State, and Pid keys — an empty or unrelated file must not parse
// into a zero-value success — and ignores keys it does not know,
// because kernels add fields over time.
func parseStatus(data []byte) (Status, error) {
	var st Status
	var sawName, sawState, sawPID bool

	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue // no colon: blank or foreign line, not ours to police
		}
		val = strings.TrimSpace(val)

		var err error
		switch key {
		case "Name":
			st.Name = val
			sawName = true
		case "State":
			if val == "" {
				err = fmt.Errorf("empty value")
				break
			}
			st.State = val[0]
			if o, c := strings.IndexByte(val, '('), strings.LastIndexByte(val, ')'); o >= 0 && c > o {
				st.StateDesc = val[o+1 : c]
			}
			sawState = true
		case "Tgid":
			st.TGID, err = strconv.Atoi(val)
		case "Pid":
			st.PID, err = strconv.Atoi(val)
			sawPID = true
		case "PPid":
			st.PPID, err = strconv.Atoi(val)
		case "Uid":
			st.UID, err = parseIDs(val)
		case "Gid":
			st.GID, err = parseIDs(val)
		case "Threads":
			st.Threads, err = strconv.Atoi(val)
		case "FDSize":
			st.FDSize, err = strconv.Atoi(val)
		case "VmPeak":
			st.VmPeakKiB, err = parseKiB(val)
		case "VmSize":
			st.VmSizeKiB, err = parseKiB(val)
		case "VmHWM":
			st.VmHWMKiB, err = parseKiB(val)
		case "VmRSS":
			st.VmRSSKiB, err = parseKiB(val)
			st.HasVM = true
		case "VmSwap":
			st.VmSwapKiB, err = parseKiB(val)
		}
		if err != nil {
			return Status{}, fmt.Errorf("%s line %q: %w", key, line, err)
		}
	}
	if err := sc.Err(); err != nil {
		return Status{}, err
	}
	if !sawName || !sawState || !sawPID {
		return Status{}, fmt.Errorf("required keys missing (need Name, State, Pid): not a status file")
	}
	return st, nil
}

// parseIDs parses the four whitespace-separated values of a Uid: or
// Gid: line — real, effective, saved, filesystem.
func parseIDs(val string) ([4]uint32, error) {
	var ids [4]uint32
	f := strings.Fields(val)
	if len(f) != 4 {
		return ids, fmt.Errorf("want 4 ids, got %d", len(f))
	}
	for i, s := range f {
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return ids, err
		}
		ids[i] = uint32(n)
	}
	return ids, nil
}

// parseKiB parses a memory value like "831896 kB". The kernel's "kB"
// suffix means KiB (the value is shifted left 10 bits, not divided by
// 1000); we record the number verbatim and leave the suffix — and all
// unit conversion — to boundaries.
func parseKiB(val string) (uint64, error) {
	f := strings.Fields(val)
	if len(f) == 0 {
		return 0, fmt.Errorf("empty value")
	}
	return strconv.ParseUint(f[0], 10, 64)
}
