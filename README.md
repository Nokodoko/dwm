<!--
 █████╗  ██████╗ ███████╗███╗   ██╗████████╗██╗ ██████╗
██╔══██╗██╔════╝ ██╔════╝████╗  ██║╚══██╔══╝██║██╔════╝
███████║██║  ███╗█████╗  ██╔██╗ ██║   ██║   ██║██║
██╔══██║██║   ██║██╔══╝  ██║╚██╗██║   ██║   ██║██║
██║  ██║╚██████╔╝███████╗██║ ╚████║   ██║   ██║╚██████╗
╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚═╝ ╚═════╝

██████╗ ██╗    ██╗███╗   ███╗
██╔══██╗██║    ██║████╗ ████║
██║  ██║██║ █╗ ██║██╔████╔██║
██║  ██║██║███╗██║██║╚██╔╝██║
██████╔╝╚███╔███╔╝██║ ╚═╝ ██║
╚═════╝  ╚══╝╚══╝ ╚═╝     ╚═╝
-->

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:0D1117,50:00D4FF,100:00FF9F&height=200&section=header&text=AGENTIC-DWM&fontSize=42&fontColor=FFFFFF&animation=fadeIn&fontAlignY=35&desc=dwm%20with%20an%20embedded%20local%20LLM%20that%20manages%20your%20X%20session&descAlignY=55&descSize=18" width="100%"/>

<div align="center">
  <img src="https://img.shields.io/badge/C-00599C?style=for-the-badge&logo=c&logoColor=white"/>
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white"/>
  <img src="https://img.shields.io/badge/Linux-FCC624?style=for-the-badge&logo=linux&logoColor=black"/>
  <img src="https://img.shields.io/badge/Arch_Linux-1793D1?style=for-the-badge&logo=arch-linux&logoColor=white"/>
  <img src="https://img.shields.io/badge/X11-F28834?style=for-the-badge&logo=x.org&logoColor=white"/>
</div>

*A personal fork of suckless dwm with `monty` — a local-LLM terminal agent that drives the window manager over dwm's IPC socket.*

## > cat /etc/project.conf

```
╭──────────────────────────────────────────────────────────╮
│                                                          │
│   PROJECT="agentic-dwm"                                  │
│   BASE="dwm 6.6 (suckless) + mihirlad55 dwm-ipc"         │
│   AGENT="monty"                                          │
│   LANGUAGE="C + Go"                                      │
│   IPC_SOCKET="/tmp/dwm.sock"                             │
│   KEYBIND="Alt+a"                                        │
│   LICENSE="MIT/X Consortium"                             │
│                                                          │
╰──────────────────────────────────────────────────────────╯
```

## Table of Contents

- [Features](#--agentic-dwm---features)
- [Architecture](#--agentic-dwm-architecture---map)
- [Prerequisites](#--cat-etcrequirements)
- [Installation](#--install---quick)
- [Usage](#--monty---help)
- [Skills](#--monty-skills---list)
- [Configuration](#--monty-env---vars)
- [Security Model](#--monty-security---audit)
- [Footer](#)

## > agentic-dwm --features

| Feature | Description |
|---------|-------------|
| **Embedded agent** | `monty`, a Bubble Tea TUI, toggled with `Alt+a` as a dwm scratchpad — floats, pins to its own tag so it survives tag switches, closes on the same key |
| **Native window control** | 10 window-management skills driven over dwm's IPC socket, plus a zsh shell skill |
| **Two new IPC commands** | `focuswin()` and `spawnsafe()`, added to dwm because the upstream command set could not express them |
| **Hot-swappable models** | `/model <name>` switches backend mid-conversation without losing history; falls back automatically when off-network |
| **Shell passthrough** | `!<command>` runs immediately in zsh, bypassing the model entirely; its output is recorded into conversation context |
| **Allowlisted spawning** | Program launches are gated by a compile-time list in dwm itself, not merely discouraged in a prompt |

## > agentic-dwm architecture --map

```
+--------------------------------------------------------------------------+
|                                X SESSION                                 |
|                                                                          |
|   +------------------------+              +------------------------+     |
|   |          dwm           |              |     ai-scratchpad      |     |
|   |      C, ~2k lines      |              |    wezterm + monty     |     |
|   |                        |              |                        |     |
|   |   focuswin(Window)     |              |   Bubble Tea TUI       |     |
|   |   spawnsafe(str)       |              |   11 skills            |     |
|   |   spawnallow[] gate    |              |   Tokyo Night Storm    |     |
|   +-----------+------------+              +-----------+------------+     |
|               |                                       |                  |
|               |      /tmp/dwm.sock  (yajl JSON)       |                  |
|               +<------------------------------------->+                  |
|                                                       |                  |
+-------------------------------------------------------|------------------+
                                                        |
                                                        v
                                          +-----------------------------+
                                          |   OpenAI-compatible API     |
                                          |                             |
                                          |   qwen   -> vLLM   (remote) |
                                          |   local  -> llama.cpp       |
                                          +-----------------------------+
```

The agent is a separate process, not linked into dwm. Restarting it never touches the running
window manager, so the model can be swapped or rebuilt without disturbing the X session.

<details>
<summary><b>Go package layout</b></summary>

| Package | Responsibility |
|---------|----------------|
| `internal/dwmipc` | dwm-ipc protocol client — framing, typed requests, async event stream |
| `internal/llm` | OpenAI-compatible chat client and the hot-swappable backend registry |
| `internal/skills` | The 11 callable tools and the live X-session snapshot |
| `internal/agentloop` | The model/tool cycle shared by the TUI and headless `-ask` mode |
| `internal/tui` | Bubble Tea interface, input routing, transcript rendering |
| `internal/theme` | Tokyo Night Storm palette and lipgloss styles |
| `cmd/probe` | Standalone dwm-ipc smoke test |

</details>

## > cat /etc/requirements

| Requirement | Notes |
|-------------|-------|
| Xlib headers | dwm build |
| Xft / freetype2 | font rendering |
| yajl | JSON for the IPC layer |
| Xinerama | optional, toggle in `config.mk` |
| Go 1.24+ | agent build |
| zsh | the `shell` skill runs `zsh -ic` |
| An OpenAI-compatible endpoint | vLLM, llama.cpp, or anything speaking `/v1/chat/completions` |

## > ./install --quick

<details>
<summary><b>Window manager</b></summary>

```bash
make clean && make
sudo make install
```

Configuration is source-level, per suckless convention: edit `config.h` (or the
`config.*.h` variant your setup symlinks to it) and rebuild.

</details>

<details>
<summary><b>Agent</b></summary>

```bash
cd agent
make
sudo make install        # installs as /usr/local/bin/monty
```

`-buildvcs=false` is set in the Makefile. It is required, not cosmetic: the module lives
inside a git worktree, where Go's VCS stamping fails with
`error obtaining VCS status: exit status 128`.

The installed binary name **must** match the one `aiscratchpadcmd` launches in `config.h`.
If they drift, every `make install` will appear to succeed while `Alt+a` keeps running a
stale binary.

</details>

<details>
<summary><b>Wiring the keybind</b></summary>

The agent runs as a dwm scratchpad. In `config.h`:

```c
static const char *aiscratchpadcmd[] = {
    "/path/to/terminal-wrapper", "start", "--class", "ai-scratchpad",
    "--always-new-process", "--", "/usr/local/bin/monty", NULL
};

/* rules[] — float it and pin it to its own tag */
{ "ai-scratchpad", NULL, NULL, AI_SCRATCHPAD_TAG, 1, -1, 0, 1, SchemeAI, "AI", 0, 0 },

/* keys[] */
{ Mod1Mask, XK_a, togglescratch, {.v = aiscratchpadcmd } },
```

Note that `togglescratch` **hides** the scratchpad rather than terminating it. A running
agent survives any number of toggles, so after rebuilding the agent you must kill the
existing scratchpad process for a new one to pick up the change.

</details>

## > monty --help

```bash
monty                          # interactive TUI (how Alt+a launches it)
monty -ask "move vivaldi to tag 3"   # one headless turn, print result, exit
monty -model local             # start on a specific backend
monty -socket /tmp/other.sock  # non-default dwm socket
```

Inside the TUI:

| Input | Effect |
|-------|--------|
| *plain text* | Sent to the model, which may call skills |
| `!<command>` | Runs in zsh **immediately**, bypassing the model; output enters conversation context |
| `/model [name]` | List backends with reachability, or hot-swap to one |
| `/windows` | Live window census |
| `/skills` | List callable skills |
| `/clear` | Reset the conversation |
| `/help` | Command reference |
| `/quit` | Close |
| `Ctrl+L` | Clear the screen |
| `PgUp` / `PgDn` | Scroll the transcript |
| `Esc` | Quit |

## > monty skills --list

| Skill | Description |
|-------|-------------|
| `list_windows` | Every managed window: id, title, tag mask, monitor, float/fullscreen state |
| `focus_window` | Focus by window id, switching monitor and tag as needed |
| `close_window` | Ask a window to close |
| `move_window_to_tag` | Move a window to a tag bitmask |
| `view_tag` | Switch the focused monitor to a tag |
| `toggle_floating` | Toggle floating for a window |
| `set_layout` | Set layout by symbol — `[]=` tiled, `[M]` monocle, `><>` floating |
| `move_window_to_monitor` | Send the focused window to another monitor |
| `set_master_factor` | Set the master area fraction |
| `launch` | Start a program — gated by dwm's `spawnallow[]` |
| `shell` | Run a command in zsh and return combined output plus exit code |

Tags are a **bitmask, not an index**: tag 1 = `1`, tag 2 = `2`, tag 3 = `4`, tag 4 = `8`.

## > monty env --vars

| Variable / Flag | Default | Purpose |
|-----------------|---------|---------|
| `DWM_SOCKET` | `/tmp/dwm.sock` | dwm IPC socket path |
| `-socket` | *(unset)* | Overrides `DWM_SOCKET` |
| `-model` | *(first reachable)* | Backend to start on |
| `-ask` | *(unset)* | Run one turn headlessly and exit |

Backends live in `internal/llm/registry.go`. Each is a name, an OpenAI-compatible base URL,
and a model id. Startup probes them in preference order and selects the first that answers,
so an unreachable remote degrades to a local model instead of failing.

<details>
<summary><b>A note on tool-call parsers</b></summary>

If a model appears to "not support tool calling," check the server's tool-call parser before
blaming the model. A mismatched parser returns `tool_calls: null` and dumps a perfectly valid
call into the message content as prose — which looks exactly like a model that has no tools.
Qwen3-family models emit XML (`<function=name>`), not Hermes JSON.

</details>

## > monty security --audit

This section is not boilerplate. The threat is concrete.

**Window titles are attacker-influenced input.** The agent feeds the window census to the
model, and a web page sets its own title. Any text arriving from a title, a file read, or a
command's output is data to be reported — never an instruction to follow. The system prompt
says so explicitly, but a prompt is mitigation, not a boundary.

**`spawnallow[]` is the actual boundary.** `spawnsafe()` checks `argv[0]` against a
compile-time list in `config.h` and splits the command line on whitespace with **no shell**,
so `;`, `|`, and `$()` remain literal. A prompt-injected model cannot exec arbitrary programs
through dwm.

> Adding `sh`, `bash`, or `zsh` to `spawnallow[]` defeats the guard completely. Extend it
> deliberately.

**The `shell` skill deliberately bypasses that guard.** It runs `zsh -ic` with your full
privileges — interactive so your aliases resolve, which also means one-word aliases for
destructive commands are reachable by name. It has a 60-second timeout and a 32 KiB output
cap, but there is no sandbox. The system prompt instructs the model to confirm before
destructive operations; that is guidance, not enforcement. If you want a real boundary,
gate the skill behind an explicit confirmation before dispatch.

**The IPC socket is the trust boundary for the machine.** Anything that can write to
`/tmp/dwm.sock` can drive the window manager and invoke `spawnsafe`. Default permissions
restrict it to the owning user; keep it that way.

## > cat CONTRIBUTING.md

This is a personal desktop fork, published because parts of it may be useful — not a product,
and not seeking feature parity with anything. Issues and patches are welcome, but the
configuration is mine and will stay opinionated.

Upstream dwm documentation lives in [`README`](README). Its license (MIT/X Consortium) applies
to this fork.

<img src="https://capsule-render.vercel.app/api?type=waving&color=0:00FF9F,50:00D4FF,100:0D1117&height=120&section=footer" width="100%"/>

<div align="center">
  <img src="https://img.shields.io/badge/License-MIT%2FX_Consortium-00FF9F?style=for-the-badge"/>
</div>

<sub align="center">Built on <a href="https://dwm.suckless.org/">suckless dwm</a> and the <a href="https://github.com/mihirlad55/dwm-ipc">mihirlad55 dwm-ipc</a> patch.</sub>
