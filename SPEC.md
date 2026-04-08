# Vivaldi Tab Control -- Port from Monty to Lewis

Cross-host deployment of the Vivaldi tab management system. Shell scripts, Python CDP tools, xkeyread C utility, dwm keybinding, dwmblocks integration, and Vivaldi debug-port config -- all ported from monty to lewis with zero functional changes.

## Why

The Vivaldi tab control system on monty provides leader-key-driven tab tiling, stacking, killing, and switching via Chrome DevTools Protocol. Lewis has none of this infrastructure:

- **No leader key dispatch.** Lewis has no `vivaldi-leader.sh`, no `xkeyread` binary, and no dwm keybinding to trigger the workflow. Alt+Ctrl+V does nothing.
- **No CDP scripts.** The Python suite (`vivaldi_base.py`, `tabTiler.py`, `tabkill.py`, `vivaldi_leader.py`, `helpers.py`) does not exist on lewis.
- **No dwmblocks indicator.** The `sb-session` status block that shows `[vivaldi]` during leader mode is absent. Signal 10 has nothing to update.
- **No debug port.** Without `~/.config/vivaldi-stable.conf` containing `--remote-debugging-port=9222`, CDP cannot connect to Vivaldi.
- **No xkeyread.** The X11 keyboard grabber needs to be compiled from source on lewis (requires libX11-dev).
- **Lewis is not in SSH config.** The host must be added to `~/.ssh/config` on monty before any rsync/scp operations work with the short name `lewis`.

This spec covers exactly the surface needed: file deployment, binary compilation, config injection, dwm rebuild, and verification.

## Design Principles

1. **Identical paths.** Both hosts use `/home/n0ko/` as home. Every file lands at the same absolute path on lewis as it occupies on monty. No path translation.
2. **No modifications.** Scripts, configs, and source are deployed as-is from monty. No lewis-specific patches, no path rewrites, no feature additions.
3. **Compile on target.** `xkeyread` is compiled from C source on lewis (not binary-copied), ensuring it links against lewis's libX11. The Python scripts are interpreted and need no compilation.
4. **Idempotent deployment.** Running the port a second time overwrites files with identical content and does not break anything. dwm rebuild via `make clean && make` is always safe.
5. **Config.h additive only.** The lewis `config.h` must gain the vivaldi-leader entries (command array, keybinding, window rules for `wezterm-tabtiler`) without disturbing existing lewis-specific settings (monitor layout, tag map, other keybindings).
6. **Verify before rebuild.** Every file is checksummed after transfer. dwm is rebuilt only after all dependencies are confirmed present on lewis.

## On-Disk Format

Files on monty that must exist on lewis at identical paths:

```
/home/n0ko/
  .local/bin/
    vivaldi-leader.sh                   # Leader key dispatcher (shell)
    xkeyread                            # X11 keyboard grabber (compiled binary)
    statusbar/
      sb-session                        # dwmblocks status indicator
  .local/src/
    xkeyread/
      xkeyread.c                        # Source for keyboard grabber
      Makefile                          # Build: cc -O2 -o xkeyread xkeyread.c -lX11
  .config/
    vivaldi-stable.conf                 # Vivaldi launch flags (debug port)
  programming/python_projects/scripts/
    vivaldi_base.py                     # CDP utilities (port 9222, tab mgmt)
    vivaldi_leader.py                   # dmenu leader dispatcher
    tabTiler.py                         # Multi-select tab tiler via gum
    tabkill.py                          # Close tabs by pattern match
    helpers.py                          # Shared dmenu/notify utilities
  bling/dwm/
    config.h                            # DWM config (needs vivaldi entries)
  bling/dwmblocks/
    blocks.h                            # dwmblocks config (needs sb-session block)
  scripts/
    wezterm-egl-fix.sh                  # Wezterm EGL workaround (dependency of tabTiler respawn)
```

### vivaldi-leader.sh

Shell dispatcher invoked by Alt+Ctrl+V. Sets leader indicator via dwmblocks signal 10, grabs keyboard with xkeyread, dispatches to Python scripts.

```sh
#!/bin/sh
LEADER_FILE="/tmp/leader_mode"
SCRIPTS="/home/n0ko/programming/python_projects/scripts"
echo "vivaldi" > "$LEADER_FILE"
pkill -RTMIN+10 dwmblocks
key=$(/home/n0ko/.local/bin/xkeyread -t 3)
rc=$?
rm -f "$LEADER_FILE"
pkill -RTMIN+10 dwmblocks
if [ $rc -ne 0 ] || [ -z "$key" ]; then exit 0; fi
case "$key" in
    t) "$SCRIPTS/tabTiler.py" ;;
    l) "$SCRIPTS/tabTiler.py" --list ;;
    u) "$SCRIPTS/tabTiler.py" --untile ;;
    s) "$SCRIPTS/tabTiler.py" --stack ;;
    w) "$SCRIPTS/tabTiler.py" --switch ;;
    x) "$SCRIPTS/tabkill.py" ;;
    *) notify-send "Vivaldi" "Unknown key: $key" --urgency=low ;;
esac
```

### vivaldi-stable.conf

Single-line config loaded by the Vivaldi wrapper script at `/opt/vivaldi/vivaldi`.

```
# Vivaldi user flags - loaded by /opt/vivaldi/vivaldi wrapper script
# Enable Chrome DevTools Protocol for programmatic tab management
--remote-debugging-port=9222
```

### config.h entries (additive)

These entries must be present in lewis's `config.h`. They may already exist if lewis shares the same config.h via git.

**Command array:**
```c
static const char *vivaldileadercmd[] = { "/home/n0ko/.local/bin/vivaldi-leader.sh", NULL };
```

**Keybinding (in `keys[]` array):**
```c
{ Mod1Mask|ControlMask, XK_v, spawn, {.v = vivaldileadercmd } },
```

**Window rules (in `rules[]` array):**
```c
{"Vivaldi-stable",        NULL,     NULL,       1,                 0,           1,      0,          -1, -1,           NULL,        0,      0},
{"wezterm-tabtiler",  NULL,     NULL,           0,                     1,          -1,      1,           1, SchemeOlr,    "tiles",     0,      0},
```

### blocks.h entry (additive)

The `sb-session` block at signal 10 must be present in lewis's `blocks.h`:

```c
{"",     "~/.local/bin/statusbar/sb-session",  0,       10},
```

## Data Model

Not applicable. This is a deployment/porting spec with no structured data entities -- all components are scripts, configs, and a compiled binary.

## CLI

Not applicable. This spec deploys existing tools; it does not create a new CLI interface. The deployed tools (`vivaldi-leader.sh`, `tabTiler.py`, `tabkill.py`, `vivaldi_leader.py`, `xkeyread`) retain their existing interfaces unchanged.

## JSON Output Format

Not applicable. The deployed scripts use `notify-send` for user feedback and stdout for inter-process communication; there is no JSON API.

## Concurrency Model

Not applicable. Deployment is a one-shot sequential operation. The deployed scripts have no concurrent access concerns beyond what they already handle on monty.

## Migration

| Component | Current (lewis) | Target (lewis after port) |
|-----------|----------------|---------------------------|
| vivaldi-leader.sh | Missing | `/home/n0ko/.local/bin/vivaldi-leader.sh` |
| xkeyread | Missing | Compiled from source at `/home/n0ko/.local/src/xkeyread/` |
| vivaldi_base.py | Missing | `/home/n0ko/programming/python_projects/scripts/vivaldi_base.py` |
| vivaldi_leader.py | Missing | `/home/n0ko/programming/python_projects/scripts/vivaldi_leader.py` |
| tabTiler.py | Missing | `/home/n0ko/programming/python_projects/scripts/tabTiler.py` |
| tabkill.py | Missing | `/home/n0ko/programming/python_projects/scripts/tabkill.py` |
| helpers.py | Missing | `/home/n0ko/programming/python_projects/scripts/helpers.py` |
| sb-session | Missing | `/home/n0ko/.local/bin/statusbar/sb-session` |
| vivaldi-stable.conf | Missing | `~/.config/vivaldi-stable.conf` with `--remote-debugging-port=9222` |
| wezterm-egl-fix.sh | Unknown | `/home/n0ko/scripts/wezterm-egl-fix.sh` (needed by tabTiler respawn) |
| config.h | Missing vivaldi entries | Add vivaldileadercmd, keybinding, window rules |
| blocks.h | Missing sb-session block | Add sb-session signal 10 block |
| SSH config | No lewis entry | Add `Host lewis` to `~/.ssh/config` on monty |
| gum | Unknown | `gum` binary must be installed (used by tabTiler.py picker) |
| wezterm | Unknown | Must be installed (used by tabTiler.py self-respawn) |
| dmenu | Unknown | Must be installed (used by tabkill.py, vivaldi_leader.py, vivaldi_base.py) |
| notify-send | Unknown | Must be installed (used by helpers.py, vivaldi-leader.sh) |
| Vivaldi | Unknown | Must be installed for any of this to be useful |
| libX11-dev | Unknown | Required to compile xkeyread |
| Python 3.10+ | Unknown | Required for match/case in helpers.py |
| jq | Unknown | Required by sb-session for session state parsing |

### Migration Steps

1. **Add lewis to SSH config** (on monty) -- prerequisite for all file transfers
2. **Probe lewis** -- SSH in and check which dependencies are present (gum, wezterm, dmenu, python3, vivaldi, jq, libX11-dev, notify-send)
3. **Install missing system packages** on lewis (gum, wezterm, Vivaldi, libX11-dev, jq, python3 if missing)
4. **Transfer files** -- rsync/scp all scripts, configs, and xkeyread source from monty to lewis
5. **Compile xkeyread** on lewis: `cd ~/.local/src/xkeyread && make clean && make install`
6. **Patch config.h** on lewis -- add vivaldileadercmd, keybinding, and window rules if missing
7. **Patch blocks.h** on lewis -- add sb-session block if missing
8. **Rebuild dwm** on lewis: `cd ~/bling/dwm && make clean && make && make install`
9. **Rebuild dwmblocks** on lewis: `cd ~/bling/dwmblocks && make clean && make && make install`
10. **Restart Vivaldi** with debug port enabled (or just restart if config was newly placed)
11. **Verify** -- run Alt+Ctrl+V on lewis, confirm `[vivaldi]` indicator appears, press `t` to launch tab tiler

## Integration

### DWM Keybinding

The entry point is Alt+Ctrl+V (Mod1Mask|ControlMask, XK_v) which spawns `vivaldi-leader.sh`.

| Key Sequence | Action |
|-------------|--------|
| Alt+Ctrl+V | Enter vivaldi leader mode |
| (leader) t | `tabTiler.py` -- multi-select tab tiler |
| (leader) l | `tabTiler.py --list` -- show tiled tabs |
| (leader) u | `tabTiler.py --untile` -- remove tiling |
| (leader) s | `tabTiler.py --stack` -- create tab stack |
| (leader) w | `tabTiler.py --switch` -- switch between stacks |
| (leader) x | `tabkill.py` -- close tabs by pattern |

### dwmblocks Integration

`sb-session` at signal 10 shows `[vivaldi]` in red when `/tmp/leader_mode` contains "vivaldi". The leader script writes and removes this file to bracket the key-capture window.

### CDP Integration

All Python scripts connect to Vivaldi via `http://localhost:9222` (Chrome DevTools Protocol). This requires `--remote-debugging-port=9222` in `~/.config/vivaldi-stable.conf`.

### tabTiler Self-Respawn

`tabTiler.py` detects whether it's running inside wezterm (via `WEZTERM_PANE` env var). When invoked from a DWM keybinding (no TTY), it re-execs itself inside a floating wezterm window with class `wezterm-tabtiler`. This class is matched by a dwm window rule that makes it floating + centered.

### Component Dependency Graph

```
Alt+Ctrl+V (dwm keybind)
    |
    v
vivaldi-leader.sh
    |
    +---> xkeyread (keyboard grab, 3s timeout)
    |
    +---> dwmblocks signal 10 ---> sb-session
    |
    +---> tabTiler.py ------> wezterm (self-respawn)
    |         |                    |
    |         +---> gum (picker)   +---> wezterm-tabtiler (dwm rule: floating)
    |         |
    |         +---> vivaldi_base.py ---> CDP ws://localhost:9222
    |
    +---> tabkill.py
    |         |
    |         +---> dmenu (pattern input + confirmation)
    |         +---> vivaldi_base.py ---> CDP http://localhost:9222
    |
    +---> helpers.py (notify-send wrapper)
```

## What It Does NOT Do

Explicitly out of scope:

- **No script modifications.** All scripts are deployed as-is. Lewis-specific path adjustments, feature additions, or bug fixes are not part of this spec.
- **No monitor layout changes.** Lewis may have a different monitor setup; the dwm config.h entries for tag-to-monitor mapping, defaulttags, and tagmonmap are NOT modified by this spec. Only vivaldi-specific entries are added.
- **No Vivaldi installation.** If Vivaldi is not installed on lewis, this spec does not install it. It is a prerequisite.
- **No Python venv management.** The scripts use stdlib modules only (no pip packages required). No virtualenv is created.
- **No dotfile sync framework.** This is a targeted port of one feature set, not a general dotfile manager. The `update-hosts` skill may be used for transfer, but this spec does not depend on or create a broader sync system.
- **No xkeyread cross-compilation.** xkeyread is compiled on lewis from transferred source, not cross-compiled on monty.

## Tech Stack

| Concern | Choice | Rationale |
|---------|--------|-----------|
| Transfer | rsync over SSH | Preserves permissions, timestamps; idempotent; works with SSH config |
| Shell scripts | POSIX sh | `vivaldi-leader.sh` and `sb-session` use `/bin/sh`; no bashisms |
| Python scripts | Python 3.10+ | `helpers.py` uses `match`/`case` (3.10 feature) |
| C compiler | system cc (gcc/clang) | xkeyread.c compiles with `cc -O2 -o xkeyread xkeyread.c -lX11` |
| System deps | gum, wezterm, dmenu, jq, notify-send, libX11 | Required at runtime by various scripts |
| Config format | C header (config.h), C header (blocks.h), plain text (vivaldi-stable.conf) | dwm/dwmblocks configuration pattern |
| Build system | make | Both dwm and dwmblocks use Makefile-based builds |

## Project Infrastructure

### Directory Structure (lewis target state)

```
/home/n0ko/
  .ssh/config                                # (monty) Add Host lewis entry
  .local/bin/
    vivaldi-leader.sh                        # Leader key dispatcher
    xkeyread                                 # Compiled X11 key grabber
    statusbar/
      sb-session                             # dwmblocks [vivaldi] indicator
  .local/src/
    xkeyread/
      xkeyread.c                             # Key grabber source
      Makefile                               # Build recipe
  .config/
    vivaldi-stable.conf                      # --remote-debugging-port=9222
  programming/python_projects/scripts/
    vivaldi_base.py                          # CDP utilities
    vivaldi_leader.py                        # dmenu leader dispatcher
    tabTiler.py                              # Tab tiler (gum + wezterm)
    tabkill.py                               # Tab killer (dmenu + CDP)
    helpers.py                               # Shared notify/dmenu utilities
  bling/dwm/
    config.h                                 # Add vivaldi entries
  bling/dwmblocks/
    blocks.h                                 # Add sb-session block
  scripts/
    wezterm-egl-fix.sh                       # Wezterm EGL workaround
```

### Version Management

Not applicable -- this is a deployment spec, not a versioned project.

### CHANGELOG.md

Not applicable.

### CI Workflow

Not applicable. Deployment is manual (or agent-driven) and one-shot.

### Scripts

No new scripts are created. Existing scripts are transferred from monty to lewis.

## Estimated Size

| Area | Files | LOC |
|------|-------|-----|
| Shell scripts | 2 (vivaldi-leader.sh, sb-session) | ~128 |
| Python scripts | 5 (vivaldi_base, vivaldi_leader, tabTiler, tabkill, helpers) | ~1,611 |
| C source | 2 (xkeyread.c, Makefile) | ~119 |
| Config files | 1 (vivaldi-stable.conf) | ~3 |
| Config patches | 2 (config.h, blocks.h) | ~6 lines added each |
| Wezterm wrapper | 1 (wezterm-egl-fix.sh) | ~15 |
| **Total** | **13** | **~1,882** |

## 15. Task Manifest

| ID | Agent | Description | File Scope (read) | File Scope (write) | Depends On | Verify Command |
|----|-------|-------------|--------------------|--------------------|------------|----------------|
| T1 | unix-coder | Add lewis to SSH config on monty | `~/.ssh/config` | `~/.ssh/config` | -- | `ssh -o ConnectTimeout=5 lewis hostname` |
| T2 | unix-coder | Probe lewis for installed dependencies (gum, wezterm, dmenu, python3, vivaldi, jq, libX11, notify-send) and report missing | -- | `/tmp/lewis-probe-results.txt` | T1 | `test -f /tmp/lewis-probe-results.txt` |
| T3 | unix-coder | Install missing system packages on lewis (based on T2 probe results) | `/tmp/lewis-probe-results.txt` | -- | T2 | `ssh lewis "which gum wezterm dmenu python3 jq notify-send && python3 --version && pkg-config --exists x11"` |
| T4 | unix-coder | Transfer shell scripts to lewis (vivaldi-leader.sh, sb-session, wezterm-egl-fix.sh) | monty: `~/.local/bin/vivaldi-leader.sh`, `~/.local/bin/statusbar/sb-session`, `~/scripts/wezterm-egl-fix.sh` | lewis: same paths | T1 | `ssh lewis "test -x ~/.local/bin/vivaldi-leader.sh && test -x ~/.local/bin/statusbar/sb-session && test -x ~/scripts/wezterm-egl-fix.sh"` |
| T5 | unix-coder | Transfer Python scripts to lewis (vivaldi_base.py, vivaldi_leader.py, tabTiler.py, tabkill.py, helpers.py) | monty: `~/programming/python_projects/scripts/{vivaldi_base,vivaldi_leader,tabTiler,tabkill,helpers}.py` | lewis: same paths | T1 | `ssh lewis "python3 -c 'import sys; sys.path.insert(0, \"/home/n0ko/programming/python_projects/scripts\"); import vivaldi_base; import helpers; print(\"OK\")'"` |
| T6 | unix-coder | Transfer xkeyread source and compile on lewis | monty: `~/.local/src/xkeyread/{xkeyread.c,Makefile}` | lewis: `~/.local/src/xkeyread/`, `~/.local/bin/xkeyread` | T3 | `ssh lewis "test -x ~/.local/bin/xkeyread && ~/.local/bin/xkeyread -t 1; test \$? -eq 1"` |
| T7 | unix-coder | Deploy vivaldi-stable.conf to lewis | monty: `~/.config/vivaldi-stable.conf` | lewis: `~/.config/vivaldi-stable.conf` | T1 | `ssh lewis "grep -q 'remote-debugging-port=9222' ~/.config/vivaldi-stable.conf"` |
| T8 | unix-coder | Patch lewis config.h with vivaldi entries (vivaldileadercmd array, keybinding, window rules) | lewis: `~/bling/dwm/config.h` | lewis: `~/bling/dwm/config.h` | T1 | `ssh lewis "grep -q 'vivaldileadercmd' ~/bling/dwm/config.h && grep -q 'wezterm-tabtiler' ~/bling/dwm/config.h"` |
| T9 | unix-coder | Patch lewis blocks.h with sb-session block at signal 10 | lewis: `~/bling/dwmblocks/blocks.h` | lewis: `~/bling/dwmblocks/blocks.h` | T1 | `ssh lewis "grep -q 'sb-session' ~/bling/dwmblocks/blocks.h"` |
| T10 | unix-coder | Rebuild dwm on lewis (make clean && make && make install) | lewis: `~/bling/dwm/` | lewis: dwm binary | T8 | `ssh lewis "cd ~/bling/dwm && make clean && make 2>&1 \| tail -1 \| grep -qv 'Error'"` |
| T11 | unix-coder | Rebuild dwmblocks on lewis (make clean && make && make install) | lewis: `~/bling/dwmblocks/` | lewis: dwmblocks binary | T9 | `ssh lewis "cd ~/bling/dwmblocks && make clean && make 2>&1 \| tail -1 \| grep -qv 'Error'"` |
| T12 | unix-coder | End-to-end verification: confirm all files present, binaries runnable, Vivaldi config correct | all deployed files on lewis | -- | T4, T5, T6, T7, T10, T11 | `ssh lewis "~/.local/bin/xkeyread -t 1; test -x ~/.local/bin/vivaldi-leader.sh && python3 -c 'import sys; sys.path.insert(0, \"/home/n0ko/programming/python_projects/scripts\"); from vivaldi_base import CDP_PORT; assert CDP_PORT == 9222' && grep -q vivaldileadercmd ~/bling/dwm/config.h && grep -q sb-session ~/bling/dwmblocks/blocks.h && grep -q 9222 ~/.config/vivaldi-stable.conf"` |

## 16. Dependency Graph

```
Phase 1 (single): [T1]
  T1: Add lewis to SSH config on monty

Phase 2 (parallel, after Phase 1): [T2, T4, T5, T7, T8, T9]
  T2: Probe lewis for dependencies
  T4: Transfer shell scripts
  T5: Transfer Python scripts
  T7: Deploy vivaldi-stable.conf
  T8: Patch config.h with vivaldi entries
  T9: Patch blocks.h with sb-session

Phase 3 (after T2): [T3]
  T3: Install missing packages on lewis

Phase 4 (parallel, after T3 + T8/T9): [T6, T10, T11]
  T6: Compile xkeyread on lewis (needs libX11 from T3)
  T10: Rebuild dwm (needs patched config.h from T8)
  T11: Rebuild dwmblocks (needs patched blocks.h from T9)

Final: [T12] -- end-to-end verification (after T4, T5, T6, T7, T10, T11)
```

## 17. Target State

Files created on lewis:

| File Path | Lines | Executable |
|-----------|-------|------------|
| `~/.local/bin/vivaldi-leader.sh` | 42 | Yes |
| `~/.local/bin/xkeyread` | (binary) | Yes |
| `~/.local/bin/statusbar/sb-session` | 85 | Yes |
| `~/.local/src/xkeyread/xkeyread.c` | 105 | No |
| `~/.local/src/xkeyread/Makefile` | 14 | No |
| `~/.config/vivaldi-stable.conf` | 3 | No |
| `~/programming/python_projects/scripts/vivaldi_base.py` | 308 | Yes |
| `~/programming/python_projects/scripts/vivaldi_leader.py` | 62 | Yes |
| `~/programming/python_projects/scripts/tabTiler.py` | 984 | Yes |
| `~/programming/python_projects/scripts/tabkill.py` | 208 | Yes |
| `~/programming/python_projects/scripts/helpers.py` | 49 | Yes |
| `~/scripts/wezterm-egl-fix.sh` | ~15 | Yes |

Files modified on lewis:

- `~/bling/dwm/config.h` -- add vivaldileadercmd, keybinding, window rules
- `~/bling/dwmblocks/blocks.h` -- add sb-session block

Files modified on monty:

- `~/.ssh/config` -- add Host lewis entry

Files deleted: None

## 18. Verification Plan

**Per-task checks:** (derived from Task Manifest Verify Command column)

- T1: `ssh -o ConnectTimeout=5 lewis hostname`
- T2: `test -f /tmp/lewis-probe-results.txt`
- T3: `ssh lewis "which gum wezterm dmenu python3 jq notify-send && python3 --version && pkg-config --exists x11"`
- T4: `ssh lewis "test -x ~/.local/bin/vivaldi-leader.sh && test -x ~/.local/bin/statusbar/sb-session"`
- T5: `ssh lewis "python3 -c 'import sys; sys.path.insert(0, \"/home/n0ko/programming/python_projects/scripts\"); import vivaldi_base; import helpers'"`
- T6: `ssh lewis "test -x ~/.local/bin/xkeyread && ~/.local/bin/xkeyread -t 1; test \$? -eq 1"`
- T7: `ssh lewis "grep -q 'remote-debugging-port=9222' ~/.config/vivaldi-stable.conf"`
- T8: `ssh lewis "grep -q vivaldileadercmd ~/bling/dwm/config.h"`
- T9: `ssh lewis "grep -q sb-session ~/bling/dwmblocks/blocks.h"`
- T10: `ssh lewis "cd ~/bling/dwm && make 2>&1 | tail -1"`
- T11: `ssh lewis "cd ~/bling/dwmblocks && make 2>&1 | tail -1"`
- T12: Full integration check (see below)

**Integration check:**

```bash
ssh lewis bash -c '
  set -e
  # 1. xkeyread binary runs (exits 1 on timeout, which is expected)
  ~/.local/bin/xkeyread -t 1 || test $? -eq 1

  # 2. Python imports work
  python3 -c "
import sys
sys.path.insert(0, \"/home/n0ko/programming/python_projects/scripts\")
from vivaldi_base import CDP_PORT, cdp_list_tabs, cdp_available
from helpers import notify_send
assert CDP_PORT == 9222
print(\"Python OK\")
"

  # 3. DWM config has all vivaldi entries
  grep -q vivaldileadercmd ~/bling/dwm/config.h
  grep -q wezterm-tabtiler ~/bling/dwm/config.h
  grep -q "Mod1Mask|ControlMask.*XK_v" ~/bling/dwm/config.h

  # 4. dwmblocks has sb-session
  grep -q sb-session ~/bling/dwmblocks/blocks.h

  # 5. Vivaldi debug port config
  grep -q 9222 ~/.config/vivaldi-stable.conf

  # 6. Shell scripts executable
  test -x ~/.local/bin/vivaldi-leader.sh
  test -x ~/.local/bin/statusbar/sb-session

  echo "ALL CHECKS PASSED"
'
```

**Rollback:** If integration fails:
- DWM config.h: `cd ~/bling/dwm && git checkout config.h && make clean && make && make install`
- dwmblocks blocks.h: `cd ~/bling/dwmblocks && git checkout blocks.h && make clean && make && make install`
- Deployed files: remove individually (all paths are known from Target State)

#### Binary Install Verification

**xkeyread compiled correctly on lewis:**

```bash
ssh lewis "~/.local/bin/xkeyread -t 1; test \$? -eq 1 && echo 'xkeyread timeout exit OK'"
```

**DWM rebuilt successfully on lewis:**

```bash
ssh lewis "cd ~/bling/dwm && make clean && make 2>&1 | grep -v '^cc\|^cp' | head -5; test -x dwm && echo 'dwm binary OK'"
```

#### Layout/Config Validation

**config.h contains all required vivaldi entries:**

```bash
ssh lewis '
  grep -q "vivaldileadercmd" ~/bling/dwm/config.h &&
  grep -q "wezterm-tabtiler" ~/bling/dwm/config.h &&
  grep -q "Vivaldi-stable" ~/bling/dwm/config.h &&
  echo "config.h OK"
'
```

**blocks.h contains sb-session block:**

```bash
ssh lewis 'grep -q "sb-session.*10" ~/bling/dwmblocks/blocks.h && echo "blocks.h OK"'
```

## 19. Success Criteria (Machine-Verifiable)

- [ ] `ssh -o ConnectTimeout=5 lewis hostname` exits 0 -- lewis reachable via SSH short name
- [ ] `ssh lewis "test -x ~/.local/bin/vivaldi-leader.sh"` exits 0 -- leader script deployed and executable
- [ ] `ssh lewis "test -x ~/.local/bin/xkeyread"` exits 0 -- xkeyread compiled and installed
- [ ] `ssh lewis "~/.local/bin/xkeyread -t 1; test \$? -eq 1"` exits 0 -- xkeyread runs (timeout exit is expected)
- [ ] `ssh lewis "test -x ~/.local/bin/statusbar/sb-session"` exits 0 -- sb-session deployed
- [ ] `ssh lewis "python3 -c 'import sys; sys.path.insert(0, \"/home/n0ko/programming/python_projects/scripts\"); import vivaldi_base; import helpers; print(\"OK\")'"` exits 0 -- Python scripts importable
- [ ] `ssh lewis "grep -q vivaldileadercmd ~/bling/dwm/config.h"` exits 0 -- vivaldi keybinding in config.h
- [ ] `ssh lewis "grep -q wezterm-tabtiler ~/bling/dwm/config.h"` exits 0 -- tabtiler window rule in config.h
- [ ] `ssh lewis "grep -q sb-session ~/bling/dwmblocks/blocks.h"` exits 0 -- sb-session in blocks.h
- [ ] `ssh lewis "grep -q 'remote-debugging-port=9222' ~/.config/vivaldi-stable.conf"` exits 0 -- CDP port configured
- [ ] `ssh lewis "cd ~/bling/dwm && make clean && make"` exits 0 -- dwm compiles with vivaldi entries
- [ ] `ssh lewis "cd ~/bling/dwmblocks && make clean && make"` exits 0 -- dwmblocks compiles with sb-session
- [ ] `ssh lewis "which gum wezterm dmenu python3 jq notify-send"` exits 0 -- all runtime dependencies present

### File Integrity Checksums (monty source files)

| File | SHA-256 |
|------|---------|
| `~/.local/bin/vivaldi-leader.sh` | `2e14dc69732a89f5f5d54109a3e24a4d2d7822efa0c1afd0da53c76287c8fe87` |
| `~/programming/python_projects/scripts/vivaldi_base.py` | `9578f4c324c5c3fdf8ce53ef5ae8292adf646ddd73c594e60f2dd2b7ecb6b951` |
| `~/programming/python_projects/scripts/vivaldi_leader.py` | `e9c88cf8e1f9da1732d36b3420ffcddb8101ca8e4163ee6af1d87948c6a05979` |
| `~/programming/python_projects/scripts/tabTiler.py` | `598edaef93199dff51d45b597a570c6a9864e0e28cb408e1c23097642c96f372` |
| `~/programming/python_projects/scripts/tabkill.py` | `71c808b34b77763091fc3469b8dab25439da1a17b8de39e4d7736bfc0d4e7192` |
| `~/programming/python_projects/scripts/helpers.py` | `d45549c89e1d152eddbef8e372b6e53395bdbedcc6c4d84e875b64b458eaf72b` |
| `~/.local/bin/statusbar/sb-session` | `ed75f7f2573d64bb41424de5834d1833fccb1593c34823e2ade2e2d3d2240547` |
| `~/.config/vivaldi-stable.conf` | `7dd514a18f8b9a5017751be896d99c8c1f23ec1c99160d7a766919ed5b8ba792` |
| `~/.local/bin/xkeyread` (binary, monty) | `0acc68079735821b49aac91d04015b55848cac1eb631c5204d8695a61e21d8eb` |

Post-transfer verification: `ssh lewis "sha256sum <path>"` must match for all non-compiled files. xkeyread is compiled on lewis, so its hash will differ from monty's binary.

## Agent Assignments

| Task | Agent | Rationale |
|------|-------|-----------|
| SSH config + probe (T1, T2) | `unix-coder` | Shell-level SSH config editing and remote probing |
| Package installation (T3) | `unix-coder` | Remote package manager operations |
| File transfer (T4, T5, T7) | `unix-coder` | rsync/scp operations |
| xkeyread compile (T6) | `unix-coder` | Remote C compilation |
| Config patching (T8, T9) | `unix-coder` | C header file editing (additive patch) |
| Build (T10, T11) | `unix-coder` | Remote make invocations |
| Verification (T12) | `unix-coder` | Remote shell verification commands |

## Execution Order

```
Phase 1: SSH Setup
  +-- T1: Add lewis to SSH config (agent: unix-coder)

Phase 2: Probe + Transfer [blocked by Phase 1]
  +-- T2: Probe lewis dependencies (agent: unix-coder)
  +-- T4: Transfer shell scripts (agent: unix-coder)     [parallel]
  +-- T5: Transfer Python scripts (agent: unix-coder)     [parallel]
  +-- T7: Deploy vivaldi-stable.conf (agent: unix-coder)  [parallel]
  +-- T8: Patch config.h (agent: unix-coder)              [parallel]
  +-- T9: Patch blocks.h (agent: unix-coder)              [parallel]

Phase 3: Package Install [blocked by T2]
  +-- T3: Install missing packages (agent: unix-coder)

Phase 4: Compile + Rebuild [blocked by T3, T8, T9]
  +-- T6: Compile xkeyread (agent: unix-coder)            [parallel]
  +-- T10: Rebuild dwm (agent: unix-coder)                [parallel]
  +-- T11: Rebuild dwmblocks (agent: unix-coder)          [parallel]

Phase 5: Verification [blocked by T4, T5, T6, T7, T10, T11]
  +-- T12: End-to-end verification (agent: unix-coder)
```

Recommended directive: `/pai` -- this is a sequential plan-then-implement pipeline. A single `unix-coder` agent handles all tasks since they all involve SSH remote operations against the same host. Parallel fan-out within phases is handled by running independent SSH commands concurrently.

## Failure Modes

| Failure | Detection | Recovery |
|---------|-----------|----------|
| Lewis unreachable via SSH | T1 verify fails: `ssh lewis hostname` returns non-zero | User must provide lewis IP/hostname and ensure SSH keys are set up. Check firewall, ensure lewis is powered on. |
| Missing system packages on lewis | T2 probe reports missing binaries | T3 installs them. If package manager is unfamiliar (not pacman/apt/dnf), agent must ask user for install commands. |
| xkeyread compile failure | T6 verify fails | Check `pkg-config --exists x11` on lewis. Install libX11 headers if missing. |
| config.h patch breaks dwm build | T10 `make` exits non-zero | Review patch for syntax errors (missing comma, wrong struct fields). Check that lewis config.h has the same Rule struct fields as monty (floatw, floath, borderscheme, etc.). If struct differs, the additive patch must match lewis's schema. |
| blocks.h patch breaks dwmblocks build | T11 `make` exits non-zero | Review patch for syntax errors. Ensure Block struct on lewis matches monty's format. |
| Python version too old | T5 import fails with SyntaxError on `match`/`case` | `helpers.py` uses `match`/`case` (Python 3.10+). Install Python 3.10+ on lewis or rewrite the match statement. |
| Vivaldi not installed on lewis | CDP unavailable at runtime | Install Vivaldi on lewis (out of scope for this spec, but T2 probe will flag it). |
| gum not installed | tabTiler.py picker fails | T3 should install gum. Verify with `which gum`. |
| wezterm not installed | tabTiler.py respawn fails | T3 should install wezterm. Verify with `which wezterm`. |
| Lewis dwm config.h has different Rule struct | T8 patch compiles but crashes | Must audit lewis config.h struct fields (floatw, floath, iscentered, bw, borderscheme, bordertitle). If struct differs, window rules must be adapted. |
| wezterm-egl-fix.sh not needed on lewis | Script harmlessly wraps wezterm; no-op if NVIDIA workaround condition is false | No recovery needed -- script is safe on all systems |

## Open Questions

| # | Question | Impact | Suggested Default |
|---|----------|--------|-------------------|
| 1 | What is lewis's hostname or IP address? Lewis is not in `~/.ssh/config` and SSH connection failed with "No route to host". | Blocks T1 (SSH config setup) and all downstream tasks. | Ask user for lewis's IP address and SSH credentials/key setup. |
| 2 | What is lewis's package manager (pacman, apt, dnf, etc.)? | Determines install commands in T3. | Assume Arch Linux (pacman) since monty appears to be Arch-based. |
| 3 | Does lewis's dwm config.h use the same Rule struct fields as monty? Monty uses extended fields: `iscentered`, `bw`, `borderscheme`, `bordertitle`, `floatw`, `floath`. | If lewis has a different/simpler Rule struct, the window rules for Vivaldi-stable and wezterm-tabtiler will cause compile errors. | Audit lewis config.h before patching. If struct differs, adapt the rule entries to match lewis's struct. |
| 4 | Does lewis already have `~/bling/dwm` and `~/bling/dwmblocks` git repos? | Determines whether config.h and blocks.h exist to patch. | Assume yes (user stated both machines run dwm from `~/bling/dwm`). |
| 5 | Does lewis have the `wezterm-egl-fix.sh` wrapper, or should it be transferred? config.h references it for terminal spawning. | tabTiler.py self-respawn uses plain `wezterm` (not the wrapper), so this is not strictly blocking for vivaldi functionality. However, other config.h entries depend on it. | Transfer it to be safe -- it is a no-op if the NVIDIA workaround is not needed. |
| 6 | Is lewis currently powered on and network-reachable? SSH probe failed during spec generation. | Blocks all execution. | User must bring lewis online before executing this spec. |

## xkeyread Source Reference

xkeyread is a small C utility that grabs the X11 keyboard, waits for a single keypress, and prints it to stdout. It is the critical bridge between dwm's leader keybinding and the shell dispatcher.

### Build

```bash
cd ~/.local/src/xkeyread
cc -O2 -o xkeyread xkeyread.c -lX11
```

### Install

```bash
make install   # copies to ~/.local/bin/xkeyread
```

### Dependencies

- `libX11` (runtime)
- `libX11` headers (build-time; `libx11-dev` on Debian, `libx11` on Arch)

### Behavior

- Grabs keyboard on the X root window (`XGrabKeyboard`)
- Waits for a single keypress with configurable timeout (`-t <seconds>`, default 3)
- Prints the key character to stdout
- Escape cancels (exit code 1, no output)
- Timeout exits with code 1
- Polls every 10ms (not busy-wait)

### Source (xkeyread.c, 105 lines)

```c
/* xkeyread - grab keyboard and read a single keypress
 *
 * Build: cc -o xkeyread xkeyread.c -lX11
 */
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/keysym.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

int
main(int argc, char *argv[])
{
    Display *dpy;
    Window root;
    XEvent ev;
    KeySym ksym;
    char buf[32];
    int len;
    int timeout_sec = 3;

    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "-t") && i + 1 < argc) {
            timeout_sec = atoi(argv[++i]);
            if (timeout_sec <= 0)
                timeout_sec = 3;
        }
    }

    dpy = XOpenDisplay(NULL);
    if (!dpy) {
        fprintf(stderr, "xkeyread: cannot open display\n");
        return 1;
    }

    root = DefaultRootWindow(dpy);

    if (XGrabKeyboard(dpy, root, True, GrabModeAsync, GrabModeAsync,
                       CurrentTime) != GrabSuccess) {
        fprintf(stderr, "xkeyread: cannot grab keyboard\n");
        XCloseDisplay(dpy);
        return 1;
    }

    time_t start = time(NULL);
    int got_key = 0;

    while (!got_key) {
        if (time(NULL) - start >= timeout_sec) {
            XUngrabKeyboard(dpy, CurrentTime);
            XCloseDisplay(dpy);
            return 1;
        }

        while (XPending(dpy)) {
            XNextEvent(dpy, &ev);
            if (ev.type == KeyPress) {
                len = XLookupString(&ev.xkey, buf, sizeof(buf) - 1,
                                    &ksym, NULL);

                if (ksym == XK_Escape) {
                    XUngrabKeyboard(dpy, CurrentTime);
                    XCloseDisplay(dpy);
                    return 1;
                }

                if (len > 0) {
                    buf[len] = '\0';
                    printf("%s", buf);
                    got_key = 1;
                }
            }
        }

        if (!got_key) {
            struct timespec ts = { 0, 10000000 };
            nanosleep(&ts, NULL);
        }
    }

    XUngrabKeyboard(dpy, CurrentTime);
    XFlush(dpy);
    XCloseDisplay(dpy);
    return 0;
}
```
