---
name: switch_display_dp
description: Switch the display to the DisplayPort monitor, turning the laptop panel off, and rescale the wallpaper, conky widgets, dunst notifications and the dwmblocks bar to the new geometry. Use for "switch to DisplayPort", "switch to DP", "use the dock monitor". Pass multi to keep the laptop panel on as well.
param.resolution: string | Mode for the DisplayPort output as WxH, e.g. 2560x1440. Optional; the monitor's preferred mode is used when omitted. Rejected if the monitor does not advertise it.
param.multi: string | Keep more than one output on. Pass "true" for the laptop panel plus DP, or an ordered comma-separated list like "dp,edp" where the FIRST entry becomes primary and leftmost.
param.dry_run: boolean | Report the plan without changing anything. Use this first when the user is unsure.
run: /home/n0ko/desktop-widgets/display-switch.sh dp
---

# /dp

Move the desktop to the DisplayPort monitor.

By default this is an exclusive switch: DP comes on and every other output goes
dark. `multi` makes it additive instead.

## The output name is discovered, not assumed

This machine enumerates `DP-1` through `DP-7` permanently, and all but one are
usually disconnected. Which one carries the monitor depends on the port and on
whether the Dell WD22TB4 dock's MST hub is in play.

`~/.screenlayout/mobile.sh` learned this the hard way — it hardcoded `DP-1` and
blacked the screen the moment the monitor came up as `DP-2`. This skill scans
for the first **connected** `DP-*` every time.

If nothing is plugged in it fails with the list of what *is* connected, rather
than blanking the screen.

## What it actually does

1. Resolves the first connected `DP-*` output.
2. Applies the geometry with `xrandr`.
3. **Verifies** the output is genuinely driving pixels afterwards — a zero exit
   from `xrandr` is not evidence the display came up. A failed verify aborts
   before any widget is touched.
4. Reads the new Xinerama index and DPI, looks up the widget scale in
   `~/desktop-widgets/display-profiles.conf`, and rescales:
   `Xft.dpi` → feh wallpaper → conky → dunst → dwmblocks → glava.

## Examples

| Ask | Arguments |
|---|---|
| "switch to DisplayPort" | *(none)* |
| "DP at 1440p" | `resolution=2560x1440` |
| "keep the laptop screen too" | `multi=true` |
| "DP primary, laptop to its right" | `multi=dp,edp` |
| "what would that do?" | `dry_run=true` |

## Arguments

Each argument reaches the script as an environment variable, never as a
command-line substitution:

- `resolution` → `$SKILL_ARG_RESOLUTION`
- `multi` → `$SKILL_ARG_MULTI`
- `dry_run` → `$SKILL_ARG_DRY_RUN`

## Verify

```sh
~/desktop-widgets/display-switch.sh --list
```

## Rollback

```sh
~/desktop-widgets/display-switch.sh edp
```
