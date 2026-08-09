---
name: switch_display_hdmi
description: Switch the display to the HDMI monitor, turning the laptop panel off, and rescale the wallpaper, conky widgets, dunst notifications and the dwmblocks bar to the new geometry. Use for "switch to HDMI", "move to the external monitor", "use the TV". Pass multi to keep the laptop panel on as well.
param.resolution: string | Mode for the HDMI output as WxH, e.g. 1920x1080. Optional; the monitor's preferred mode is used when omitted. Rejected if the monitor does not advertise it.
param.multi: string | Keep more than one output on. Pass "true" for the laptop panel plus HDMI, or an ordered comma-separated list like "hdmi,edp" where the FIRST entry becomes primary and leftmost.
param.dry_run: boolean | Report the plan without changing anything. Use this first when the user is unsure.
run: /home/n0ko/desktop-widgets/display-switch.sh hdmi
---

# /hdmi

Move the desktop to the HDMI monitor.

By default this is an exclusive switch: HDMI comes on and every other output
goes dark. `multi` makes it additive instead.

## What it actually does

1. Resolves the first **connected** `HDMI-*` output. It is discovered, never
   hardcoded — the port that lights up varies with the dock.
2. Applies the geometry with `xrandr`.
3. **Verifies** the output is genuinely driving pixels afterwards. `xrandr`
   exits 0 for plenty of requests that leave a monitor dark, so a zero exit is
   not evidence the display came up. A failed verify aborts before any widget
   is touched.
4. Reads the new Xinerama index and DPI, looks up the widget scale in
   `~/desktop-widgets/display-profiles.conf`, and rescales:
   `Xft.dpi` → feh wallpaper → conky → dunst → dwmblocks → glava.

## Examples

| Ask | Arguments |
|---|---|
| "switch to HDMI" | *(none)* |
| "HDMI at 1080p" | `resolution=1920x1080` |
| "keep the laptop screen too" | `multi=true` |
| "HDMI on the left, laptop on the right" | `multi=hdmi,edp` |
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
