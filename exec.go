package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func run(timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readInt(path string) (int64, bool) {
	s := readFile(path)
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.Fields(s)[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func topProcesses(limit int) []Proc {
	out := run(3*time.Second, "ps", "-axo", "pid=,pcpu=,rss=,comm=")
	var rows []Proc
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		cpu, err2 := strconv.ParseFloat(fields[1], 64)
		rss, err3 := strconv.ParseInt(fields[2], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		comm := strings.Join(fields[3:], " ")
		rows = append(rows, Proc{
			PID:    pid,
			CPUPct: cpu,
			RSSMB:  roundN(float64(rss)/1024.0, 1),
			Comm:   comm,
		})
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].CPUPct > rows[i].CPUPct {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

func pipe(timeout time.Duration, stdin []byte, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	return cmd.Output()
}
