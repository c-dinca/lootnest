package plat

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// DiskFree is the space free to this user on path's file system.
func DiskFree(path string) int64 {
	free, _ := DiskSpace(path)
	return free
}

// DiskSpace is the space free to this user and the file system's size.
func DiskSpace(path string) (free, total int64) {
	var s syscall.Statfs_t
	for p := path; p != ""; p = filepath.Dir(p) {
		if syscall.Statfs(p, &s) == nil {
			return int64(s.Bavail) * int64(s.Bsize), int64(s.Blocks) * int64(s.Bsize)
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return 0, 0
}

// CPUName is the processor's name from /proc/cpuinfo.
func CPUName() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), ":"); ok && strings.TrimSpace(k) == "model name" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// TotalRAM is the memory size in bytes, from /proc/meminfo.
func TotalRAM() int64 {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "MemTotal:" {
			kb, _ := strconv.ParseInt(f[1], 10, 64)
			return kb * 1024
		}
	}
	return 0
}

// ProcessNames are the running programs' names, lower case without .exe,
// as Windows' list gives them: a native program by its file's name, and a
// Windows one run through Wine or Proton by the .exe it runs.
func ProcessNames() (map[string]bool, error) {
	return processNamesIn("/proc")
}

func processNamesIn(proc string) (map[string]bool, error) {
	ents, err := os.ReadDir(proc)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, e := range ents {
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		dir := filepath.Join(proc, e.Name())
		if b, err := os.ReadFile(filepath.Join(dir, "comm")); err == nil {
			if n := strings.TrimSpace(string(b)); n != "" {
				out[strings.ToLower(strings.TrimSuffix(n, ".exe"))] = true
			}
		}
		b, err := os.ReadFile(filepath.Join(dir, "cmdline"))
		if err != nil || len(b) == 0 {
			continue
		}
		args := strings.Split(strings.TrimRight(string(b), "\x00"), "\x00")
		for _, a := range args[:min(len(args), 3)] {
			// Wine's own command line names the Windows program (C:\...\x.exe).
			base := strings.ToLower(a[strings.LastIndexAny(a, `/\`)+1:])
			if strings.HasSuffix(base, ".exe") {
				out[strings.TrimSuffix(base, ".exe")] = true
			}
		}
		if base := filepath.Base(args[0]); base != "" && base != "." {
			out[strings.ToLower(base)] = true
		}
	}
	return out, nil
}
