package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type side struct {
	Label    string   `json:"label"`
	OS       string   `json:"os"`
	Hostname string   `json:"hostname"`
	Machine  string   `json:"machine"`
	Watts    *float64 `json:"watts"`
	HoursOwn *float64 `json:"hours_at_full_charge"`
	Hours60  *float64 `json:"hours_normalized_60wh"`
	Context  Context  `json:"context"`
}

type Gap struct {
	Schema    string   `json:"schema"`
	Scenario  string   `json:"scenario"`
	Mac       side     `json:"mac"`
	Omarchy   side     `json:"omarchy"`
	GapW      *float64 `json:"gap_w"`
	GapPct    *float64 `json:"gap_pct"`
	NorthStar string   `json:"north_star"`
	Warnings  []string `json:"warnings"`
}

func loadRecord(path string) (Record, error) {
	var rec Record
	b, err := os.ReadFile(path)
	if err != nil {
		return rec, err
	}
	err = json.Unmarshal(b, &rec)
	return rec, err
}

func asSide(rec Record, label string) side {
	return side{
		Label:    label,
		OS:       rec.OS,
		Hostname: rec.Hostname,
		Machine:  rec.Machine.Product,
		Watts:    rec.Summary.Watts.Mean,
		HoursOwn: rec.Summary.HoursAtFullCharge,
		Hours60:  rec.Summary.HoursNormalized60Wh,
		Context:  rec.Context,
	}
}

func compareRecords(a, b Record) Gap {
	mac, oma := a, b
	if a.OS == "linux" && b.OS == "darwin" {
		mac, oma = b, a
	} else if b.OS == "linux" && a.OS == "darwin" {
		mac, oma = a, b
	} else if a.OS == "linux" {
		mac, oma = b, a
	}
	macW := mac.Summary.Watts.Mean
	omaW := oma.Summary.Watts.Mean
	var gapW, gapPct *float64
	if macW != nil && omaW != nil {
		g := roundN(*omaW-*macW, 3)
		gapW = &g
		if *macW != 0 {
			p := roundN(100.0*g / *macW, 1)
			gapPct = &p
		}
	}
	scenario := oma.Scenario
	if scenario == "" {
		scenario = mac.Scenario
	}
	return Gap{
		Schema:    "omarchesse.gap/v1",
		Scenario:  scenario,
		Mac:       asSide(mac, "Mac"),
		Omarchy:   asSide(oma, "Omarchy"),
		GapW:      gapW,
		GapPct:    gapPct,
		NorthStar: "gap_w → 0",
		Warnings:  mismatch(mac, oma),
	}
}

func mismatch(a, b Record) []string {
	var w []string
	if a.Scenario != "" && b.Scenario != "" && a.Scenario != b.Scenario {
		w = append(w, fmt.Sprintf("scenarios differ: %s vs %s", a.Scenario, b.Scenario))
	}
	if a.Context.Brightness != nil && b.Context.Brightness != nil {
		if abs(*a.Context.Brightness-*b.Context.Brightness) > 0.15 {
			w = append(w, "brightness differs a lot — display power will dominate the gap")
		}
	}
	if !a.OnBattery || !b.OnBattery {
		w = append(w, "one or both recordings were on AC; battery gap is not valid")
	}
	return w
}

func formatGap(g Gap) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Scenario: %s\n", dash(g.Scenario))
	fmt.Fprintf(&b, "North star: gap_w → 0 W\n\n")
	fmt.Fprintf(&b, "  %-10s %8s  %18s  %14s\n", "", "watts", "hours @ own pack", "hours @ 60Wh")
	fmt.Fprintf(&b, "  %-10s %8s  %18s  %14s\n", "Mac", fmtOpt(g.Mac.Watts), fmtOpt(g.Mac.HoursOwn), fmtOpt(g.Mac.Hours60))
	fmt.Fprintf(&b, "  %-10s %8s  %18s  %14s\n", "Omarchy", fmtOpt(g.Omarchy.Watts), fmtOpt(g.Omarchy.HoursOwn), fmtOpt(g.Omarchy.Hours60))
	if g.GapW != nil {
		pct := ""
		if g.GapPct != nil {
			pct = fmt.Sprintf(" (%+.0f%%)", *g.GapPct)
		}
		fmt.Fprintf(&b, "\n  Gap        %+.2f W%s\n", *g.GapW, pct)
		if *g.GapW <= 0 {
			fmt.Fprintf(&b, "  Omarchy is at or under the Mac for this scenario.\n")
		} else {
			fmt.Fprintf(&b, "  Omarchy is drawing more. That extra wattage is the whole project.\n")
		}
	}
	for _, warning := range g.Warnings {
		fmt.Fprintf(&b, "\n  warning: %s", warning)
	}
	if len(g.Warnings) > 0 {
		fmt.Fprintln(&b)
	}
	return b.String()
}

func fmtOpt(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%g", roundN(*v, 2))
}

func dash(s string) string {
	if s == "" {
		return "?"
	}
	return s
}

func newestPair(dirs []string) (string, string, error) {
	var mac, oma string
	var macT, omaT time.Time
	for _, dir := range dirs {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			name := e.Name()
			switch {
			case strings.Contains(name, "-mac-") || strings.HasPrefix(name, "darwin-"):
				if info.ModTime().After(macT) {
					mac, macT = path, info.ModTime()
				}
			case strings.Contains(name, "-omarchy-") || strings.HasPrefix(name, "linux-"):
				if info.ModTime().After(omaT) {
					oma, omaT = path, info.ModTime()
				}
			}
		}
	}
	if mac == "" || oma == "" {
		return "", "", fmt.Errorf("need one Mac result and one Omarchy result in this folder (omarchesse-mac-*.json and omarchesse-omarchy-*.json)")
	}
	return mac, oma, nil
}
