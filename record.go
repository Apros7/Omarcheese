package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func runRecord(sc scenario, duration, settle, interval time.Duration, note string, allowAC bool) (string, error) {
	s := &Sampler{}
	first := snapshot(s)
	if !first.onBattery && !allowAC {
		return "", fmt.Errorf("not on battery — unplug the charger, wait a few seconds, then run this again")
	}
	if first.watts == nil && !allowAC {
		return "", fmt.Errorf("could not read system watts from the battery sensor")
	}

	fmt.Println()
	fmt.Printf("Recording %s for %s on %s\n", sc.title, prettyDur(duration), first.machine.Product)
	fmt.Println("Unplug. Brightness 40%. Don't touch the machine.")
	if first.watts != nil {
		fmt.Printf("Right now: %.2f W\n", *first.watts)
	}
	fmt.Println()

	if settle > 0 {
		deadline := time.Now().Add(settle)
		for time.Now().Before(deadline) {
			left := time.Until(deadline).Round(time.Second)
			fmt.Printf("\rsettling… %s left   ", left)
			time.Sleep(200 * time.Millisecond)
		}
		fmt.Printf("\r%-40s\r", "")
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	started := time.Now().UTC()
	deadline := time.Now().Add(duration)
	var samples []Sample
	partial := false

loop:
	for {
		now := time.Now()
		r := snapshot(s)
		samples = append(samples, Sample{
			T:       float64(now.UnixMilli()) / 1000.0,
			Watts:   r.watts,
			CPUPkgW: r.cpuPkgW,
			Percent: r.battery.Percent,
		})
		mean := summarize(wattsOf(samples)).Mean
		wtxt := "—"
		if r.watts != nil {
			wtxt = fmt.Sprintf("%.2f W", *r.watts)
		}
		mtxt := ""
		if mean != nil {
			mtxt = fmt.Sprintf("  mean %.2f W", *mean)
		}
		fmt.Printf("\r%s  %s%s   ", clock(time.Since(started)), wtxt, mtxt)

		if !now.Before(deadline) {
			break
		}
		select {
		case <-interrupt:
			partial = true
			fmt.Print("\ninterrupted — saving what we have")
			break loop
		case <-time.After(interval):
		}
	}
	fmt.Println()

	last := snapshot(s)
	osName, pretty := hostOS()
	watts := summarize(wattsOf(samples))
	pkg := summarize(pkgOf(samples))
	full := last.battery.FullWh
	if full == nil {
		full = first.battery.FullWh
	}

	doc := Record{
		Schema:         schema,
		ToolVersion:    version,
		Scenario:       sc.name,
		StartedAt:      started.Format(time.RFC3339),
		DurationS:      duration.Seconds(),
		IntervalS:      interval.Seconds(),
		SettleS:        settle.Seconds(),
		Note:           note,
		Partial:        partial,
		OS:             osName,
		OSPretty:       pretty,
		Hostname:       hostname(),
		User:           os.Getenv("USER"),
		Machine:        first.machine,
		Context:        first.context,
		Battery:        last.battery,
		OnBattery:      first.onBattery,
		WattSource:     first.wattSource,
		Samples:        samples,
		Summary:        Summary{Watts: watts, CPUPkgW: pkg, HoursAtFullCharge: hoursFromWh(full, watts.Mean), HoursNormalized60Wh: hoursFromWh(fptr(60, 2), watts.Mean)},
		ProcessesAtEnd: topProcesses(8),
	}

	path, err := saveRecord(doc)
	if err != nil {
		return "", err
	}
	if watts.Mean != nil {
		fmt.Printf("mean %s W", fmtFloat(*watts.Mean))
		if watts.P50 != nil && watts.P95 != nil {
			fmt.Printf("  (p50 %s  p95 %s)", fmtFloat(*watts.P50), fmtFloat(*watts.P95))
		}
		fmt.Println()
		if doc.Summary.HoursNormalized60Wh != nil {
			fmt.Printf("≈ %s hours on a 60 Wh pack at this drain\n", fmtFloat(*doc.Summary.HoursNormalized60Wh))
		}
	}
	fmt.Printf("wrote %s\n", path)
	fmt.Println()
	fmt.Println("Copy this file to the Studio, then:")
	fmt.Println("  omarchesse compare <mac.json> <omarchy.json>")
	return path, nil
}

func wattsOf(samples []Sample) []*float64 {
	out := make([]*float64, len(samples))
	for i := range samples {
		out[i] = samples[i].Watts
	}
	return out
}

func pkgOf(samples []Sample) []*float64 {
	out := make([]*float64, len(samples))
	for i := range samples {
		out[i] = samples[i].CPUPkgW
	}
	return out
}

func saveRecord(doc Record) (string, error) {
	stamp := time.Now().Format("20060102-150405")
	host := sanitize(doc.Hostname)
	name := fmt.Sprintf("omarchesse-%s-%s-%s-%s.json", sideName(), doc.Scenario, host, stamp)
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := filepath.Join(cwd, name)
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, append(raw, '\n'), 0644)
}

func sanitize(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	out := re.ReplaceAllString(s, "-")
	if out == "" {
		return "host"
	}
	return out
}

func prettyDur(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d min %ds", m, s)
}

func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

func fmtFloat(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	return strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
}
