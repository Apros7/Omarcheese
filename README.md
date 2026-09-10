# Omarchesse

Battery-saving work for [Omarchy](https://omarchy.org) on a Framework laptop, measured against a MacBook doing the same thing.

The north-star metric is not "hours remaining" in the menu bar. It is:

```
gap_w = mean watts on Omarchy − mean watts on the MacBook
```

for a named workload, both unplugged, brightness matched. The project is done when `gap_w` is 0.

Background on why watts, where they go, and what Omarchy's defaults cost: [docs/how-power-works.md](docs/how-power-works.md).

## First experiment

Clone this repo on **both** laptops. Python 3.9+ from the system is enough. No packages to install.

On each machine, unplug, set brightness to 40%, then:

```bash
python3 -m omarchesse protocol idle-desktop
python3 -m omarchesse record idle-desktop
```

Copy the two JSON files from `results/` onto one computer and:

```bash
python3 -m omarchesse compare results/darwin-idle-desktop-....json results/linux-idle-desktop-....json
```

That printout is the baseline. After that, the `coding` scenario is the one that matters for daily use.

On the Framework only:

```bash
python3 -m omarchesse snapshot    # watts right now, plus Hyprland/power-profile context
python3 -m omarchesse audit       # read-only list of likely waste (does not change anything)
```

`bin/omarchesse` is the same CLI if you prefer a script on `PATH`.

## Rules for a fair comparison

- Unplug. AC numbers are a different measurement.
- Same scenario, same brightness, Wi-Fi on, keyboard backlight off.
- Let the machine sit through the settle period. Don't poke it during `idle-desktop`.
- Don't compare a MacBook on Low Power Mode to Omarchy on `performance`.
- This Mac Studio (if you develop here) has no battery. It cannot be the Mac baseline.

## What happens after we have numbers

1. Read the idle gap. If idle is already 8 W worse, fix the floor (profile, refresh rate, blur, wakeups) before chasing apps.
2. Read `audit` on Omarchy. Apply one change at a time.
3. Re-record the same scenario. Keep the JSON. The history is the project.
4. Only then look at coding/browse/video. Those sit on top of the idle floor.
# Omarcheese
