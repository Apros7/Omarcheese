# Setup

Battery-saving work for [Omarchy](https://omarchy.org) on a Framework, measured against a MacBook doing the same thing.

North-star metric:

```
gap_w = mean watts on Omarchy − mean watts on the MacBook
```

Same named workload, both unplugged, brightness matched. Done when `gap_w` is 0.

Why watts, where they go, what Omarchy's defaults cost: [docs/how-power-works.md](docs/how-power-works.md).

## Use it

Binaries live in `dist/` after `./script/build`:

| File | Machine |
| --- | --- |
| `dist/omarchesse-mac` | Apple silicon MacBook (and this Studio) |
| `dist/omarchesse-omarchy` | Framework running Omarchy |
| `dist/omarchesse-mac-intel` | Intel MacBook, if you have one |

Copy the matching file onto each laptop. No Python, no install.

On **each** laptop: unplug, brightness 40%, extra apps closed, then:

```bash
chmod +x omarchesse-mac          # once; on Omarchy use omarchesse-omarchy
./omarchesse-mac                 # 3 minutes of idle desktop
```

It writes a JSON file next to the binary, named like `omarchesse-mac-idle-desktop-….json`. Copy both JSON files back to this machine (this folder is fine) and:

```bash
./dist/omarchesse-mac compare
```

That picks the newest Mac + Omarchy files in the folder. Or pass them explicitly:

```bash
./dist/omarchesse-mac compare omarchesse-mac-….json omarchesse-omarchy-….json
```

Daily-use scenario, after idle:

```bash
./omarchesse-mac coding
```

On the Framework only, `./omarchesse-omarchy audit` is a read-only checklist. It does not change anything.

If macOS blocks the binary: right-click → Open, or `xattr -d com.apple.quarantine omarchesse-mac`.

## Fair comparison

- Unplug. This Studio has no battery and cannot be the Mac baseline.
- Same brightness, Wi-Fi on, keyboard backlight off.
- Don't compare Mac Low Power Mode to Omarchy `performance`.
- Don't poke the machine during idle.

## After you have numbers

1. If idle is already several watts worse, fix the floor (profile, refresh rate, blur) before chasing apps.
2. Change one thing on Omarchy, re-record, keep the JSON. The history is the project.
3. Then record `coding`.
