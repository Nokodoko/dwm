---
name: switch_display_edp
description: Switch back to the built-in laptop panel (eDP-1), turning every external monitor off, and rescale the wallpaper, conky widgets, dunst notifications and the dwmblocks bar back to the laptop's high-DPI geometry. Use for "back to the laptop", "undock", "turn off the external monitor", "revert the display".
param.resolution: string | Mode for the laptop panel as WxH, e.g. 2560x1600. Optional; the panel's preferred mode is used when omitted.
param.multi: string | Keep more than one output on. Pass an ordered comma-separated list like "edp,hdmi" where the FIRST entry becomes primary and leftmost. Rarely needed here, since this skill exists to go back to the panel alone.
param.dry_run: boolean | Report the plan without changing anything.
run: /home/n0ko/desktop-widgets/display-switch.sh edp
---

# /edp

Return the desktop to the built-in laptop panel and turn the externals off.

This is the rollback for `/hdmi` and `/dp`, and the thing to reach for when
undocking.

## Why this is not just "xrandr --auto"

eDP-1 is 2560x1600 in 288mm — about 226 dpi, more than twice the density of the
~96 dpi externals. The widget geometry that looks right on an external is half
the size it needs to be here. Coming back therefore has to restore the 2x
widget scale, not only the resolution: `display-profiles.conf` maps `eDP-1` to
scale 2, and dunst's config is re-rendered at that scale.

It also clears any CRTC still held by a now-disconnected output, which would
otherwise leave the root window wider than the visible panel and strand
windows off-screen.

## What it actually does

1. Resolves the first connected `eDP-*` output.
2. Applies the geometry with `xrandr` and switches every other output off.
3. **Verifies** the panel is genuinely driving pixels afterwards.
4. Rescales `Xft.dpi` → feh wallpaper → conky → dunst → dwmblocks → glava at
   the laptop's 2x profile.

## Examples

| Ask | Arguments |
|---|---|
| "back to the laptop" | *(none)* |
| "undock" | *(none)* |
| "laptop primary, HDMI to the right" | `multi=edp,hdmi` |

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
