//go:build linux

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func platformSample(s *Sampler) reading {
	bat := linuxBatteryDir()
	onAC := linuxOnAC()
	var watts *float64
	if bat != "" && !onAC {
		watts = linuxBatteryWatts(bat)
	}
	status := ""
	if bat != "" {
		status = readFile(filepath.Join(bat, "status"))
	}
	onBattery := bat != "" && !onAC && strings.EqualFold(status, "discharging")

	var percent *float64
	if cap, ok := readInt(filepath.Join(bat, "capacity")); ok {
		percent = fptr(float64(cap), 1)
	}

	now := time.Now()
	var pkg *float64
	if uj, ok := linuxRAPLUJ(); ok {
		if s.hasRAPL && now.After(s.raplTime) {
			dt := now.Sub(s.raplTime).Seconds()
			if dt > 0 {
				w := float64(uj-s.raplUJ) / dt / 1e6
				if w >= 0 {
					pkg = fptr(w, 3)
				}
			}
		}
		s.raplTime = now
		s.raplUJ = uj
		s.hasRAPL = true
	}

	profile := run(2*time.Second, "powerprofilesctl", "get")
	return reading{
		watts:      watts,
		cpuPkgW:    pkg,
		onBattery:  onBattery,
		onAC:       onAC,
		wattSource: "sysfs.power_now",
		battery: Battery{
			Status:   status,
			Percent:  percent,
			NowWh:    linuxEnergyWh(bat, "energy_now"),
			FullWh:   linuxEnergyWh(bat, "energy_full"),
			DesignWh: linuxEnergyWh(bat, "energy_full_design"),
		},
		context: Context{
			PowerProfile:   profile,
			Governor:       readFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor"),
			EPP:            readFile("/sys/devices/system/cpu/cpu0/cpufreq/energy_performance_preference"),
			AMDPState:      readFile("/sys/devices/system/cpu/amd_pstate/status"),
			Brightness:     linuxBrightness(),
			RefreshHz:      linuxRefreshHz(),
			HyprAnimations: hyprBool("animations:enabled"),
			HyprBlur:       hyprBool("decoration:blur:enabled"),
		},
		machine: linuxMachine(),
	}
}

func linuxBatteryDir() string {
	root := "/sys/class/power_supply"
	ents, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), "BAT") {
			return filepath.Join(root, e.Name())
		}
	}
	return ""
}

func linuxOnAC() bool {
	root := "/sys/class/power_supply"
	ents, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, e := range ents {
		dir := filepath.Join(root, e.Name())
		kind := readFile(filepath.Join(dir, "type"))
		online := readFile(filepath.Join(dir, "online"))
		if (kind == "Mains" || kind == "USB") && online == "1" {
			return true
		}
	}
	return false
}

func linuxBatteryWatts(bat string) *float64 {
	if p, ok := readInt(filepath.Join(bat, "power_now")); ok && p > 0 {
		return fptr(float64(p)/1e6, 3)
	}
	cur, ok1 := readInt(filepath.Join(bat, "current_now"))
	volt, ok2 := readInt(filepath.Join(bat, "voltage_now"))
	if ok1 && ok2 {
		w := absInt(cur) * absInt(volt) / 1e12
		return fptr(w, 3)
	}
	return nil
}

func absInt(v int64) float64 {
	if v < 0 {
		return float64(-v)
	}
	return float64(v)
}

func linuxEnergyWh(bat, name string) *float64 {
	if bat == "" {
		return nil
	}
	if n, ok := readInt(filepath.Join(bat, name)); ok {
		return fptr(float64(n)/1e6, 2)
	}
	chargeName := strings.Replace(name, "energy_", "charge_", 1)
	charge, ok1 := readInt(filepath.Join(bat, chargeName))
	volt, ok2 := readInt(filepath.Join(bat, "voltage_now"))
	if !ok2 {
		volt, ok2 = readInt(filepath.Join(bat, "voltage_min_design"))
	}
	if ok1 && ok2 {
		return fptr((float64(charge)/1e6)*(float64(volt)/1e6), 2)
	}
	return nil
}

func linuxRAPLUJ() (int64, bool) {
	matches, _ := filepath.Glob("/sys/class/powercap/intel-rapl:*")
	total := int64(0)
	found := false
	for _, dir := range matches {
		base := filepath.Base(dir)
		if strings.Count(base, ":") != 1 {
			continue
		}
		if n, ok := readInt(filepath.Join(dir, "energy_uj")); ok {
			total += n
			found = true
		}
	}
	return total, found
}

func linuxBrightness() *float64 {
	ents, err := os.ReadDir("/sys/class/backlight")
	if err != nil || len(ents) == 0 {
		return nil
	}
	dir := filepath.Join("/sys/class/backlight", ents[0].Name())
	cur, ok1 := readInt(filepath.Join(dir, "brightness"))
	max, ok2 := readInt(filepath.Join(dir, "max_brightness"))
	if !ok1 || !ok2 || max == 0 {
		return nil
	}
	return fptr(float64(cur)/float64(max), 3)
}

func linuxRefreshHz() *float64 {
	out := run(2*time.Second, "hyprctl", "monitors", "-j")
	if out == "" {
		return nil
	}
	var mons []map[string]any
	if err := json.Unmarshal([]byte(out), &mons); err != nil || len(mons) == 0 {
		return nil
	}
	pick := mons[0]
	for _, m := range mons {
		name, _ := m["name"].(string)
		focused, _ := m["focused"].(bool)
		if focused || strings.HasPrefix(name, "eDP") {
			pick = m
			break
		}
	}
	if hz, ok := asFloat(pick["refreshRate"]); ok {
		return fptr(hz, 2)
	}
	return nil
}

func hyprBool(option string) *bool {
	out := run(2*time.Second, "hyprctl", "getoption", option, "-j")
	if out == "" {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		return nil
	}
	if v, ok := asInt64(data["int"]); ok {
		b := v != 0
		return &b
	}
	if v, ok := asBool(data["set"]); ok {
		return &v
	}
	return nil
}

func linuxMachine() Machine {
	product := readFile("/sys/class/dmi/id/product_name")
	vendor := readFile("/sys/class/dmi/id/sys_vendor")
	cpuinfo := readFile("/proc/cpuinfo")
	cpu := ""
	re := regexp.MustCompile(`(?m)^model name\s*:\s*(.+)$`)
	if m := re.FindStringSubmatch(cpuinfo); len(m) == 2 {
		cpu = strings.TrimSpace(m[1])
	}
	return Machine{
		Product: product,
		Vendor:  vendor,
		CPU:     cpu,
		Kernel:  readFile("/proc/sys/kernel/osrelease"),
	}
}
