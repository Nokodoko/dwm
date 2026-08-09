---
name: create_skill
description: Create a new monty skill from a plain-language description of what it should do. Writes a SKILL.md into .agents/skills/ with valid frontmatter so monty can call it as a tool on next start. Use when the user says "make a skill that…", "teach yourself to…", "add a command for…", or "/skill".
param.name: string | snake_case tool name the model will call, e.g. mute_audio. Lowercase letters, digits and underscores only.
param.description: string | One or two sentences describing WHEN to use the skill. This is the only text the model sees when deciding whether to call it, so name the trigger phrases a user would actually say.
param.run: string | The command line, evaluated by zsh -c. Read arguments from $SKILL_ARG_<NAME> environment variables.
param.params: string | Argument declarations as name:type:description, separated by semicolons. Types are string, integer, number or boolean. Example: "volume:integer:Percentage 0-100;mute:boolean:Silence instead of setting a level".
param.required: string | Comma-separated subset of the declared param names that must always be supplied.
param.body: string | Markdown procedure written into the file after the frontmatter, for humans and for Claude. Optional; the description is used when omitted.
param.force: boolean | Overwrite an existing skill of the same name.
required: [name, description, run]
run: /home/n0ko/bling/dwm-pertag/.agents/skills/skill/new-skill.sh
---

# /skill — the skill that writes skills

monty's builtin skills are compiled into the binary because they are
structural: they wrap dwm's fixed IPC command set and cannot change without it
changing. Everything else is policy, and recompiling an agent to teach it a new
trick is the wrong shape. This skill is the other half — it writes a `SKILL.md`
into `.agents/skills/`, which monty loads at startup and Claude reads directly.

## How to turn a request into a skill

The user will say something short — "make a skill that mutes audio", "add a
command to dim the screen". Turn that into the four things a skill needs:

**1. A name the model will guess correctly.** snake_case, verb-first,
specific. `mute_audio`, not `audio` or `do_mute`. It must not collide with a
builtin (`list_windows`, `focus_window`, `close_window`, `move_window_to_tag`,
`view_tag`, `toggle_floating`, `set_layout`, `move_window_to_monitor`,
`set_master_factor`, `launch`, `shell`) — the loader refuses shadows, and a
refused skill is simply absent with no error at the point of use.

**2. A description that names the trigger, not the implementation.** This is
the *entire* basis on which the model decides to call it. Write the phrases a
user would actually say. "Mute system audio. Use for 'mute', 'silence',
'quiet'." beats "Runs pactl set-sink-mute."

**3. A `run` that is idempotent and verifiable.** Prefer a command that can be
run twice safely. If the action can silently fail — anything touching displays,
audio sinks, services, or the network — the command must *check afterwards*
and exit non-zero when the state did not actually change. A command that
reports success it did not achieve is worse than no skill.

**4. Arguments that are typed and mostly optional.** Every argument arrives as
`$SKILL_ARG_<UPPERCASE_NAME>`. They are passed through the environment and are
never substituted into the command string, so a value cannot become shell
syntax — write `"$SKILL_ARG_VOLUME"`, and do not try to build a command line
out of them.

## When NOT to make a skill

If the task is a one-off, monty already has `shell`. A skill earns its file
when it is *recurring*, *has a name the user says out loud*, or *needs steps in
a specific order*. Wrapping a single shell command that the model could have
written itself just adds a thing to keep working.

## Example

Request: *"make a skill that locks the screen"*

| Field | Value |
|---|---|
| `name` | `lock_screen` |
| `description` | `Lock the X session immediately. Use for "lock", "lock the screen", "I'm stepping away".` |
| `run` | `slock` |
| `params` | *(none)* |

Request: *"add a command to set the volume"*

| Field | Value |
|---|---|
| `name` | `set_volume` |
| `description` | `Set the system output volume as a percentage. Use for "volume to 40", "turn it down", "louder".` |
| `run` | `pactl set-sink-volume @DEFAULT_SINK@ "${SKILL_ARG_PERCENT}%" && pactl get-sink-volume @DEFAULT_SINK@` |
| `params` | `percent:integer:Target volume 0-100` |
| `required` | `percent` |

Note the `&& pactl get-sink-volume` — the skill reports the resulting state
rather than assuming the set worked.

## Where skills live

The loader scans, in order:

1. `~/bling/dwm-pertag/.agents/skills/` — the live dwm tree, and the default
   for anything written here
2. `~/bling/dwm/.agents/skills/`
3. `~/.agents/skills/`

`$DWM_AGENT_SKILLS` overrides the whole list, colon-separated. A directory
without a `SKILL.md` is skipped, and a malformed skill costs only itself — the
rest still load, with the parse error printed to stderr at startup.

## Verify

```sh
monty -ask '/skills'      # or, inside monty:  /skills -l
```

A new skill appears with its source path next to it. If it is missing, monty
printed the reason to stderr at startup.
