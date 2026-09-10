package main

import "time"

const (
	version = "0.2.0"
	schema  = "omarchesse.record/v1"
)

type Machine struct {
	Product string `json:"product,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
	CPU     string `json:"cpu,omitempty"`
	Kernel  string `json:"kernel,omitempty"`
	HWModel string `json:"hw_model,omitempty"`
}

type Battery struct {
	Installed  *bool    `json:"installed,omitempty"`
	Status     string   `json:"status,omitempty"`
	Percent    *float64 `json:"percent,omitempty"`
	NowWh      *float64 `json:"now_wh,omitempty"`
	FullWh     *float64 `json:"full_wh,omitempty"`
	DesignWh   *float64 `json:"design_wh,omitempty"`
	CycleCount *int     `json:"cycle_count,omitempty"`
}

type Context struct {
	PowerProfile   string   `json:"power_profile,omitempty"`
	LowPowerMode   *bool    `json:"low_power_mode,omitempty"`
	Governor       string   `json:"governor,omitempty"`
	EPP            string   `json:"epp,omitempty"`
	AMDPState      string   `json:"amd_pstate,omitempty"`
	Brightness     *float64 `json:"brightness,omitempty"`
	RefreshHz      *float64 `json:"refresh_hz,omitempty"`
	HyprAnimations *bool    `json:"hypr_animations,omitempty"`
	HyprBlur       *bool    `json:"hypr_blur,omitempty"`
}

type Proc struct {
	PID    int     `json:"pid"`
	CPUPct float64 `json:"cpu_pct"`
	RSSMB  float64 `json:"rss_mb"`
	Comm   string  `json:"comm"`
}

type Sample struct {
	T       float64  `json:"t"`
	Watts   *float64 `json:"watts"`
	CPUPkgW *float64 `json:"cpu_pkg_w"`
	Percent *float64 `json:"percent"`
}

type SummaryStats struct {
	N     int      `json:"n"`
	Mean  *float64 `json:"mean"`
	P50   *float64 `json:"p50"`
	P95   *float64 `json:"p95"`
	Min   *float64 `json:"min"`
	Max   *float64 `json:"max"`
	Stdev float64  `json:"stdev"`
}

type Summary struct {
	Watts               SummaryStats `json:"watts"`
	CPUPkgW             SummaryStats `json:"cpu_pkg_w"`
	HoursAtFullCharge   *float64     `json:"hours_at_full_charge"`
	HoursNormalized60Wh *float64     `json:"hours_normalized_60wh"`
}

type Record struct {
	Schema         string   `json:"schema"`
	ToolVersion    string   `json:"tool_version"`
	Scenario       string   `json:"scenario"`
	StartedAt      string   `json:"started_at"`
	DurationS      float64  `json:"duration_s"`
	IntervalS      float64  `json:"interval_s"`
	SettleS        float64  `json:"settle_s"`
	Note           string   `json:"note,omitempty"`
	Partial        bool     `json:"partial,omitempty"`
	OS             string   `json:"os"`
	OSPretty       string   `json:"os_pretty"`
	Hostname       string   `json:"hostname"`
	User           string   `json:"user,omitempty"`
	Machine        Machine  `json:"machine"`
	Context        Context  `json:"context"`
	Battery        Battery  `json:"battery"`
	OnBattery      bool     `json:"on_battery"`
	WattSource     string   `json:"watt_source,omitempty"`
	Samples        []Sample `json:"samples"`
	Summary        Summary  `json:"summary"`
	ProcessesAtEnd []Proc   `json:"processes_at_end,omitempty"`
}

type reading struct {
	watts      *float64
	cpuPkgW    *float64
	onBattery  bool
	onAC       bool
	wattSource string
	battery    Battery
	context    Context
	machine    Machine
}

type Sampler struct {
	raplTime time.Time
	raplUJ   int64
	hasRAPL  bool
}

type scenario struct {
	name     string
	title    string
	duration time.Duration
	why      string
	steps    []string
}

var scenarios = map[string]scenario{
	"idle-desktop": {
		name:     "idle-desktop",
		title:    "Idle desktop",
		duration: 3 * time.Minute,
		why:      "The floor: OS + display + radios while you do nothing.",
		steps: []string{
			"Unplug the charger.",
			"Set brightness to 40% on both laptops. No auto-brightness.",
			"Wi-Fi on, keyboard backlight off, extra apps closed.",
			"Don't touch the machine until it finishes.",
		},
	},
	"coding": {
		name:     "coding",
		title:    "Coding session",
		duration: 5 * time.Minute,
		why:      "Daily-use north star: same editor, same project, both unplugged.",
		steps: []string{
			"Unplug. Brightness 40%. Wi-Fi on.",
			"Open Cursor with the same project and the same 2–3 files.",
			"No compile, no video, no extra browser.",
		},
	},
	"browse": {
		name:     "browse",
		title:    "Light browsing",
		duration: 5 * time.Minute,
		why:      "Browser + compositor GPU cost.",
		steps: []string{
			"Unplug. Brightness 40%. Same three mostly-static tabs on both machines.",
		},
	},
	"video": {
		name:     "video",
		title:    "Video playback",
		duration: 5 * time.Minute,
		why:      "Checks whether video decode is offloaded to the GPU.",
		steps: []string{
			"Unplug. Brightness 40%. Same 1080p video on both machines.",
		},
	},
}

func aliases(name string) string {
	switch name {
	case "idle", "idle-desktop":
		return "idle-desktop"
	default:
		return name
	}
}
