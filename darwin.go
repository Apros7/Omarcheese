//go:build darwin

package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	machineOnce sync.Once
	cachedMach  Machine
)

func platformSample(s *Sampler) reading {
	_ = s
	batt := ioregBattery()
	telemetry, _ := batt["PowerTelemetryData"].(map[string]any)

	installed, _ := asBool(batt["BatteryInstalled"])
	external, _ := asBool(batt["ExternalConnected"])
	charging, _ := asBool(batt["IsCharging"])
	fully, _ := asBool(batt["FullyCharged"])

	voltageMV, _ := asFloat(batt["Voltage"])
	ampsRaw, _ := asInt64(first(batt, "InstantAmperage", "Amperage"))
	ampsMA := signedCurrentMA(ampsRaw)
	var viWatts *float64
	if voltageMV > 0 {
		viWatts = fptr(abs(float64(ampsMA))*voltageMV/1e6, 3)
	}

	onBattery := installed && !external && !charging
	var watts *float64
	source := ""
	if onBattery {
		if mw, ok := asFloat(telemetry["BatteryPower"]); ok && mw != 0 {
			watts = fptr(abs(mw)/1000.0, 3)
			source = "smc.BatteryPower"
		} else if viWatts != nil {
			watts = viWatts
			source = "ioreg.V*I"
		}
	} else if mw, ok := asFloat(first(telemetry, "SystemLoad", "SystemPowerIn")); ok && mw != 0 {
		watts = fptr(abs(mw)/1000.0, 3)
		source = "smc.SystemLoad"
	}

	volts := 11.5
	if voltageMV > 0 {
		volts = voltageMV / 1000.0
	}
	nowMah := first(batt, "AppleRawCurrentCapacity", "CurrentCapacity")
	fullMah := first(batt, "AppleRawMaxCapacity", "MaxCapacity")
	designMah := first(batt, "DesignCapacity", "NominalChargeCapacity")

	status := "unknown"
	switch {
	case charging:
		status = "charging"
	case fully:
		status = "full"
	case onBattery:
		status = "discharging"
	}

	var percent *float64
	if installed {
		if cap, ok := asFloat(batt["CurrentCapacity"]); ok && cap > 0 && cap <= 100 {
			percent = fptr(cap, 1)
		} else if n, ok1 := asFloat(nowMah); ok1 {
			if f, ok2 := asFloat(fullMah); ok2 && f > 0 {
				percent = fptr(100.0*n/f, 1)
			}
		}
	}

	profile := macosPowerMode()
	low := profile == "low-power"
	inst := installed
	b := Battery{
		Installed:  &inst,
		Status:     status,
		Percent:    percent,
		NowWh:      mahToWh(nowMah, volts),
		FullWh:     mahToWh(fullMah, volts),
		DesignWh:   mahToWh(designMah, volts),
		CycleCount: intPtr(batt["CycleCount"]),
	}

	return reading{
		watts:      watts,
		onBattery:  onBattery,
		onAC:       external || !installed,
		wattSource: source,
		battery:    b,
		context: Context{
			PowerProfile: profile,
			LowPowerMode: &low,
		},
		machine: macosMachine(),
	}
}

func mahToWh(v any, volts float64) *float64 {
	n, ok := asFloat(v)
	if !ok || n <= 0 {
		return nil
	}
	if n <= 100 {
		return nil
	}
	return fptr((n/1000.0)*volts, 2)
}

func intPtr(v any) *int {
	n, ok := asInt64(v)
	if !ok {
		return nil
	}
	i := int(n)
	return &i
}

func first(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

func ioregBattery() map[string]any {
	raw, err := exec.Command("ioreg", "-r", "-c", "AppleSmartBattery", "-a").Output()
	if err != nil || len(raw) == 0 {
		return map[string]any{}
	}
	js, err := pipe(5*time.Second, raw, "plutil", "-convert", "json", "-o", "-", "-")
	if err != nil || len(js) == 0 {
		return map[string]any{}
	}
	var arr []map[string]any
	dec := json.NewDecoder(bytes.NewReader(js))
	dec.UseNumber()
	if err := dec.Decode(&arr); err != nil || len(arr) == 0 {
		return map[string]any{}
	}
	return normalizeJSON(arr[0])
}

func normalizeJSON(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = convertJSON(v)
	}
	return out
}

func convertJSON(v any) any {
	switch t := v.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		f, _ := t.Float64()
		return f
	case map[string]any:
		return normalizeJSON(t)
	case []any:
		for i := range t {
			t[i] = convertJSON(t[i])
		}
		return t
	default:
		return v
	}
}

func macosPowerMode() string {
	batt := run(2*time.Second, "pmset", "-g", "batt")
	pm := run(2*time.Second, "pmset", "-g")
	low := false
	for _, line := range strings.Split(pm, "\n") {
		if strings.Contains(strings.ToLower(line), "lowpowermode") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[len(fields)-1] == "1" {
				low = true
			}
		}
	}
	if low {
		return "low-power"
	}
	if strings.Contains(batt, "Battery Power") {
		return "battery"
	}
	if strings.Contains(batt, "AC Power") {
		return "ac"
	}
	return ""
}

func macosMachine() Machine {
	machineOnce.Do(func() {
		model := run(2*time.Second, "sysctl", "-n", "hw.model")
		chip := run(2*time.Second, "sysctl", "-n", "machdep.cpu.brand_string")
		profiler := run(8*time.Second, "system_profiler", "SPHardwareDataType")
		var parts []string
		for _, line := range strings.Split(profiler, "\n") {
			trim := strings.TrimSpace(line)
			for _, prefix := range []string{"Model Name:", "Chip:", "Model Identifier:"} {
				if strings.HasPrefix(trim, prefix) {
					parts = append(parts, strings.TrimSpace(strings.TrimPrefix(trim, prefix)))
				}
			}
		}
		product := strings.Join(parts, " | ")
		if product == "" {
			product = model
		}
		cachedMach = Machine{
			Product: product,
			Vendor:  "Apple",
			CPU:     chip,
			Kernel:  run(2*time.Second, "uname", "-r"),
			HWModel: model,
		}
	})
	return cachedMach
}
