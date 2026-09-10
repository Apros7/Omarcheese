package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type finding struct {
	severity string
	title    string
	detail   string
	fix      string
	save     string
}

func runAudit() {
	s := &Sampler{}
	snap := snapshot(s)
	var findings []finding
	add := func(sev, title, detail, fix, save string) {
		findings = append(findings, finding{sev, title, detail, fix, save})
	}

	if runtime.GOOS != "linux" {
		add("info", "Audit is for the Omarchy laptop", "Run this on the Framework. On the MacBook, just record.", "", "")
		if !snap.onBattery {
			add("warn", "This Mac is on AC / has no battery", "The Studio cannot be a battery baseline. Use the MacBook unplugged.", "", "")
		}
		fmt.Println(formatAudit(findings))
		return
	}

	if !snap.onBattery {
		add("warn", "Currently on AC", "Unplug before recording. Watt readings change on the charger.", "", "")
	}
	ctx := snap.context
	switch ctx.PowerProfile {
	case "performance":
		add("high", "Power profile is performance", "If this is on battery, the machine is not trying to save energy.", "omarchy powerprofiles set battery power-saver", "2–8 W")
	case "balanced":
		add("med", "Power profile is balanced (Omarchy's battery default)", "Omarchy uses balanced on battery, not power-saver.", "omarchy powerprofiles set battery power-saver", "1–4 W")
	case "power-saver":
		add("ok", "Power profile is power-saver", "Right profile for a Mac comparison.", "", "")
	}
	if ctx.HyprAnimations != nil && *ctx.HyprAnimations {
		add("med", "Hyprland animations are on", "Keeps the GPU busier than macOS for the same idle desktop.", "hyprctl keyword animations:enabled false", "0.3–1.5 W")
	}
	if ctx.HyprBlur != nil && *ctx.HyprBlur {
		add("med", "Hyprland blur is on", "A compositor tax. Cheap on Apple silicon, not cheap here.", "hyprctl keyword decoration:blur:enabled false", "0.5–2 W")
	}
	if ctx.RefreshHz != nil && *ctx.RefreshHz > 75 {
		add("med", fmt.Sprintf("Display is at %.0f Hz", *ctx.RefreshHz), "macOS typically downclocks the panel on battery.", "hyprctl keyword monitor eDP-1,preferred@60,auto,1", "0.5–2 W")
	}
	if ctx.Brightness != nil && *ctx.Brightness > 0.55 {
		add("med", fmt.Sprintf("Backlight is at %.0f%%", *ctx.Brightness*100), "Match brightness with the MacBook before comparing.", "brightnessctl set 40%", "1–4 W")
	}
	if systemdActive("tlp.service") && (systemdActive("power-profiles-daemon.service") || lookPath("powerprofilesctl")) {
		add("high", "TLP and power-profiles-daemon both look active", "They fight over the same knobs. Omarchy ships power-profiles-daemon.", "sudo systemctl disable --now tlp.service", "1–5 W")
	}
	if ctx.AMDPState != "" && ctx.AMDPState != "active" {
		add("med", "amd_pstate is "+ctx.AMDPState, "Measure before changing this. Active + power-profiles-daemon is the usual Framework AMD setup.", "", "")
	}
	if readFile("/proc/sys/kernel/nmi_watchdog") == "1" {
		add("low", "NMI watchdog is enabled", "Tiny periodic CPU wakeups.", "echo 0 | sudo tee /proc/sys/kernel/nmi_watchdog", "0.1–0.5 W")
	}
	if snap.watts != nil {
		w := *snap.watts
		switch {
		case w >= 12:
			add("high", fmt.Sprintf("Right now the machine is drawing %.1f W", w), "A well-tuned Framework 13 on Linux often idles around 4–7 W. Something is stuck awake.", "", "")
		case w >= 8:
			add("med", fmt.Sprintf("Right now the machine is drawing %.1f W", w), "Usable, but still well above a MacBook Air idle.", "", "")
		default:
			add("ok", fmt.Sprintf("Right now the machine is drawing %.1f W", w), "Sane Linux-Framework band. Compare it to the MacBook next.", "", "")
		}
	}
	fmt.Println(formatAudit(findings))
}

func formatAudit(findings []finding) string {
	order := map[string]int{"high": 0, "med": 1, "low": 2, "warn": 3, "ok": 4, "info": 5}
	for i := 0; i < len(findings); i++ {
		for j := i + 1; j < len(findings); j++ {
			if order[findings[j].severity] < order[findings[i].severity] {
				findings[i], findings[j] = findings[j], findings[i]
			}
		}
	}
	mark := map[string]string{
		"high": "[HIGH]", "med": "[MED] ", "low": "[LOW] ",
		"warn": "[WARN]", "ok": "[OK]  ", "info": "[INFO]",
	}
	var b strings.Builder
	fmt.Fprintln(&b, "Omarchy power audit (read-only)")
	fmt.Fprintln(&b)
	for _, item := range findings {
		fmt.Fprintf(&b, "%s %s\n", mark[item.severity], item.title)
		fmt.Fprintf(&b, "       %s\n", item.detail)
		if item.save != "" {
			fmt.Fprintf(&b, "       typical save: %s\n", item.save)
		}
		if item.fix != "" {
			fmt.Fprintf(&b, "       try: %s\n", item.fix)
		}
		fmt.Fprintln(&b)
	}
	return strings.TrimRight(b.String(), "\n")
}

func systemdActive(unit string) bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", unit)
	return cmd.Run() == nil
}

func lookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
