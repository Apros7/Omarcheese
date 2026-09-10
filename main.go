package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Omarchesse measures laptop watts on Mac and Omarchy.

Run the same binary-style command on each laptop, unplugged, brightness 40%%.
It waits a few minutes, then writes a JSON file next to you. Bring both
files back here and compare.

  omarchesse                 3 min idle desktop (the first experiment)
  omarchesse coding          5 min coding session
  omarchesse browse
  omarchesse video
  omarchesse compare a b     Mac vs Omarchy gap
  omarchesse compare         newest mac + omarchy JSON in this folder
  omarchesse snapshot        one-shot watt reading
  omarchesse audit           read-only Omarchy waste checklist

  --duration 3m    override length (also 180 or 180s)
  --allow-ac       record while plugged in (not a battery baseline)
  --note text      stored in the JSON

`)
}

func main() {
	fs := flag.NewFlagSet("omarchesse", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = usage
	durationFlag := fs.String("duration", "", "override recording length")
	settleFlag := fs.String("settle", "15s", "settle time before sampling")
	intervalFlag := fs.String("interval", "2s", "sample interval")
	note := fs.String("note", "", "note stored in the JSON")
	allowAC := fs.Bool("allow-ac", false, "allow recording on AC")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}
	args := fs.Args()
	cmd := "idle-desktop"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "help", "-h", "--help":
		usage()
		return
	case "version", "--version":
		fmt.Println(version)
		return
	case "snapshot":
		printSnapshot()
		return
	case "audit":
		runAudit()
		return
	case "compare":
		var a, b string
		var err error
		switch len(args) {
		case 1:
			cwd, _ := os.Getwd()
			a, b, err = newestPair([]string{cwd, "results"})
		case 3:
			a, b = args[1], args[2]
		default:
			usage()
			os.Exit(2)
		}
		if err != nil {
			fatal(err)
		}
		runCompare(a, b)
		return
	}

	name := aliases(cmd)
	sc, ok := scenarios[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	duration := sc.duration
	if *durationFlag != "" {
		d, err := parseDuration(*durationFlag)
		if err != nil {
			fatal(err)
		}
		duration = d
	}
	settle, err := parseDuration(*settleFlag)
	if err != nil {
		fatal(err)
	}
	interval, err := parseDuration(*intervalFlag)
	if err != nil {
		fatal(err)
	}
	printProtocol(sc)
	if _, err := runRecord(sc, duration, settle, interval, *note, *allowAC); err != nil {
		fatal(err)
	}
}

func runCompare(pathA, pathB string) {
	a, err := loadRecord(pathA)
	if err != nil {
		fatal(err)
	}
	b, err := loadRecord(pathB)
	if err != nil {
		fatal(err)
	}
	gap := compareRecords(a, b)
	fmt.Println(formatGap(gap))
	out, _ := json.MarshalIndent(map[string]any{
		"scenario":   gap.Scenario,
		"gap_w":      gap.GapW,
		"gap_pct":    gap.GapPct,
		"north_star": gap.NorthStar,
	}, "", "  ")
	fmt.Println(string(out))
}

func printSnapshot() {
	r := snapshot(&Sampler{})
	osName, pretty := hostOS()
	fmt.Println(r.machine.Product)
	fmt.Println(pretty, "("+osName+")")
	if r.watts == nil {
		fmt.Println("watts: unavailable")
	} else {
		fmt.Printf("watts: %.2f W\n", *r.watts)
	}
	if r.cpuPkgW != nil {
		fmt.Printf("cpu package (RAPL): %.2f W\n", *r.cpuPkgW)
	}
	fmt.Printf("on battery: %v\n", r.onBattery)
	if r.battery.Installed != nil && !*r.battery.Installed {
		fmt.Println("battery: none (this machine cannot be a MacBook baseline)")
	} else if r.battery.Percent != nil {
		now := "-"
		full := "-"
		if r.battery.NowWh != nil {
			now = fmt.Sprintf("%.1f", *r.battery.NowWh)
		}
		if r.battery.FullWh != nil {
			full = fmt.Sprintf("%.1f", *r.battery.FullWh)
		}
		fmt.Printf("battery: %.0f%%  (%s / %s Wh)\n", *r.battery.Percent, now, full)
	}
	fmt.Printf("power profile: %s\n", dash(r.context.PowerProfile))
	if r.context.Brightness != nil {
		fmt.Printf("brightness: %.0f%%\n", *r.context.Brightness*100)
	}
	if r.context.RefreshHz != nil {
		fmt.Printf("refresh: %.0f Hz\n", *r.context.RefreshHz)
	}
	procs := topProcesses(5)
	if len(procs) > 0 {
		fmt.Println("top CPU:")
		for _, p := range procs {
			fmt.Printf("  %5.1f%%  %s\n", p.CPUPct, p.Comm)
		}
	}
}

func printProtocol(sc scenario) {
	fmt.Println(sc.title)
	fmt.Println(sc.why)
	fmt.Println()
	for i, step := range sc.steps {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
		return time.Duration(n) * time.Second, nil
	}
	return 0, fmt.Errorf("invalid duration %q (try 3m, 180s, or 180)", s)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
