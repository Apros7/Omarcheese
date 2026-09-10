# How laptop battery actually gets spent

A battery is a tank of energy, measured in **watt-hours (Wh)**. Drain is a rate, measured in **watts (W)**.

```
hours left ≈ watt-hours remaining / watts right now
```

That is the whole game. "The Framework dies fast" almost always means **watts are high**, not that the pack is magically smaller.

A Framework 13 pack is typically 55–61 Wh. A MacBook Air pack is typically ~52 Wh. Those are close. If the Mac lasts twice as long, Omarchy is drawing roughly twice the watts.

## The metric this repo uses

For a named scenario S (idle, coding, browse, video):

```
gap_w(S) = mean_watts_omarchy(S) − mean_watts_macbook(S)
```

**North star: `gap_w → 0`.**

Hours are reported too, but they lie a little: different pack sizes, different battery health, different charge start. Watts do not. We also show "hours if both had a 60 Wh pack" so runtime is comparable.

Record **unplugged**. On AC you are measuring the charger, not the laptop.

## Where the watts go

Rough idle split on a modern 13" laptop, moderate brightness:

| Bucket | Typical watts | What it actually is |
| --- | --- | --- |
| Display backlight | 1–6 W | Panel + PWM. Often the largest single item. Match brightness on both machines or the comparison is junk. |
| CPU package | 1–8 W idle, much more loaded | Cores, caches, integrated GPU. On Linux this is RAPL. High idle here means the package is not sleeping (wakeups, a spinning process, a bad power profile). |
| Compositor / GPU | 0.3–3 W+ | Hyprland drawing blur, shadows, animations, 120 Hz. macOS does this cheaply on Apple silicon. |
| Wi-Fi / radios | 0.3–2 W | Scanning, a bad driver, leftover hotspot. |
| NVMe, USB, EC, fans | 0.5–3 W | Framework expansion cards are real: a USB-A HDMI module can cost watts even idle. |
| Discrete GPU (if any) | 3–15 W | Framework 16 only, unless it is stuck out of D3cold. |

If idle is 12–20 W, something is stuck awake. A well-tuned Framework 13 on Linux often sits around **4–7 W** at ~40% brightness. A MacBook Air often sits around **2–5 W**.

## What Omarchy does today

Omarchy is Arch + Hyprland with `power-profiles-daemon`:

- **AC → performance**
- **Battery → balanced** (not power-saver)
- Power-saver only kicks in when the battery is already low
- Default look: blur on, animations on
- A 120 Hz Framework panel stays at 120 Hz unless you change it

None of that is "wrong" for a desktop-class feel. It is expensive.

Do **not** stack TLP on top of power-profiles-daemon. They fight. Measure before adding another daemon.

## How we measure

**MacBook (unplugged):** SMC / `AppleSmartBattery`. Instantaneous watts from battery current × voltage, or `PowerTelemetryData.BatteryPower`.

**Omarchy (unplugged):** `/sys/class/power_supply/BAT*/power_now` (whole machine). Optional RAPL (`/sys/class/powercap/intel-rapl:*`) for CPU-package watts. The difference is "everything else" (panel, radios, USB, EC).

A 3–5 minute recording with 2 second samples is enough (`./omarchesse` on each laptop). One screenshot of "time remaining" is not: that estimate jumps around and assumes the current drain lasts forever.

## What "gap → 0" can and cannot mean

The MacBook is Apple silicon. The Framework is AMD or Intel. The SoCs are not equally efficient. **Zero gap may be unreachable for some workloads.** That is still the right target: it tells you how much OS/config waste is left vs how much is silicon.

Two ladders, both useful:

1. **MacBook parity** — `gap_w` for `coding` and `idle-desktop`. Aspirational.
2. **Well-tuned Linux Framework** — idle ~4–7 W. Achievable on this hardware regardless of the Mac.
