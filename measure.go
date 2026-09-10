package main

import (
	"os"
	"runtime"
	"time"
)

func snapshot(s *Sampler) reading {
	r := platformSample(s)
	return r
}

func hostOS() (osName, pretty string) {
	switch runtime.GOOS {
	case "darwin":
		return "darwin", "macOS " + run(2*time.Second, "sw_vers", "-productVersion")
	case "linux":
		return "linux", "Linux " + readFile("/proc/sys/kernel/osrelease")
	default:
		return runtime.GOOS, runtime.GOOS + " " + runtime.GOARCH
	}
}

func sideName() string {
	if runtime.GOOS == "darwin" {
		return "mac"
	}
	return "omarchy"
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}
