#!/bin/bash
# new-skill.sh — write a new SKILL.md into .agents/skills/.
#
# This is the executable half of the `/skill` metaprompt. The model decides
# WHAT the skill should be (that judgement lives in SKILL.md's body); this
# script does the mechanical part -- create the directory, emit well-formed
# frontmatter, and refuse the mistakes that would otherwise only surface as a
# silent load failure at monty's next start.
#
# Arguments arrive as SKILL_ARG_* environment variables, the same way every
# other filesystem skill receives them.
#
#   SKILL_ARG_NAME         required  snake_case tool name the model will call
#   SKILL_ARG_DESCRIPTION  required  one or two sentences; this is the ONLY
#                                    thing the model uses to decide when to call
#   SKILL_ARG_RUN          required  command line, evaluated by zsh -c
#   SKILL_ARG_PARAMS       optional  name:type:description; repeated with ;
#   SKILL_ARG_REQUIRED     optional  comma-separated subset of the param names
#   SKILL_ARG_BODY         optional  markdown procedure appended after the
#                                    frontmatter, for humans and for Claude
#   SKILL_ARG_FORCE        optional  overwrite an existing skill of that name

set -euo pipefail

ROOT="${DWM_AGENT_SKILLS_ROOT:-$HOME/bling/dwm-pertag/.agents/skills}"

die() { printf 'new-skill: error: %s\n' "$*" >&2; exit 1; }

NAME="${SKILL_ARG_NAME:-}"
DESCRIPTION="${SKILL_ARG_DESCRIPTION:-}"
RUN="${SKILL_ARG_RUN:-}"
PARAMS="${SKILL_ARG_PARAMS:-}"
REQUIRED="${SKILL_ARG_REQUIRED:-}"
BODY="${SKILL_ARG_BODY:-}"
FORCE="${SKILL_ARG_FORCE:-}"

[[ -n $NAME        ]] || die "name is required"
[[ -n $DESCRIPTION ]] || die "description is required"
[[ -n $RUN         ]] || die "run is required"

# The name becomes a JSON tool name and a directory, so constrain it to what is
# valid in both. A name with a space or a slash parses fine here and then fails
# in a way that points at the model instead of at this file.
[[ $NAME =~ ^[a-z][a-z0-9_]*$ ]] ||
    die "name '$NAME' must be snake_case: lowercase, digits and underscores, starting with a letter"

# These are monty's compiled-in skills. A markdown skill that shadows one is
# refused at load, so catching it here turns a silent no-op into a real message.
for builtin in list_windows focus_window close_window move_window_to_tag \
               view_tag toggle_floating set_layout move_window_to_monitor \
               set_master_factor launch shell; do
    [[ $NAME == "$builtin" ]] && die "'$NAME' is a builtin skill; pick another name"
done

DIR="$ROOT/$NAME"
FILE="$DIR/SKILL.md"

if [[ -e $FILE && -z $FORCE ]]; then
    die "$FILE already exists (pass force=true to overwrite)"
fi

mkdir -p "$DIR"

# Frontmatter values are single-line: a newline would end the key and silently
# truncate the description.
flatten() { printf '%s' "$1" | tr '\n' ' ' | sed 's/  */ /g; s/^ //; s/ $//'; }

# Compose into a temp file, not straight into $FILE.
#
# The param and required validation below happens while the frontmatter is
# already being emitted, so writing directly to the destination left a
# half-finished SKILL.md behind on every rejected input -- and a truncated file
# is worse than no file, because the loader then reports a parse error for a
# skill the user never successfully created.
tmp=$(mktemp "$DIR/.SKILL.md.XXXXXX")
trap 'rm -f "$tmp"' EXIT

{
    printf -- '---\n'
    printf 'name: %s\n' "$(flatten "$NAME")"
    printf 'description: %s\n' "$(flatten "$DESCRIPTION")"

    declare -a declared=()
    if [[ -n $PARAMS ]]; then
        IFS=';' read -r -a specs <<< "$PARAMS"
        for spec in "${specs[@]}"; do
            spec="${spec#"${spec%%[![:space:]]*}"}"   # ltrim
            [[ -n $spec ]] || continue
            pname="${spec%%:*}"
            rest="${spec#*:}"
            if [[ $rest == "$spec" ]]; then
                ptype="string"; pdesc=""
            else
                ptype="${rest%%:*}"
                pdesc="${rest#*:}"
                [[ $pdesc == "$rest" ]] && pdesc=""
            fi
            [[ -n $pname ]] || die "empty param name in '$spec'"
            [[ $pname =~ ^[a-z][a-z0-9_]*$ ]] || die "param '$pname' must be snake_case"
            case "$ptype" in
                string|integer|number|boolean) ;;
                *) die "param '$pname' has type '$ptype'; want string, integer, number or boolean" ;;
            esac
            declared+=("$pname")
            printf 'param.%s: %s | %s\n' "$pname" "$ptype" "$(flatten "$pdesc")"
        done
    fi

    if [[ -n $REQUIRED ]]; then
        IFS=',' read -r -a reqs <<< "$REQUIRED"
        for r in "${reqs[@]}"; do
            r="${r// /}"
            [[ -n $r ]] || continue
            found=0
            for d in ${declared[@]+"${declared[@]}"}; do
                [[ $d == "$r" ]] && found=1
            done
            (( found )) || die "required lists '$r', which is not a declared param"
        done
        printf 'required: [%s]\n' "$(printf '%s' "$REQUIRED" | tr -d ' ')"
    fi

    printf 'run: %s\n' "$(flatten "$RUN")"
    printf -- '---\n\n'

    printf '# %s\n\n' "$NAME"
    if [[ -n $BODY ]]; then
        printf '%s\n' "$BODY"
    else
        printf '%s\n' "$DESCRIPTION"
    fi

    if [[ ${#declared[@]} -gt 0 ]]; then
        printf '\n## Arguments\n\n'
        printf 'Each argument reaches the command as an environment variable, never\n'
        printf 'as a command-line substitution:\n\n'
        for d in "${declared[@]}"; do
            printf -- '- `%s` → `$SKILL_ARG_%s`\n' "$d" "$(printf '%s' "$d" | tr '[:lower:]' '[:upper:]')"
        done
    fi
} > "$tmp"

mv "$tmp" "$FILE"
trap - EXIT
chmod 644 "$FILE"

printf 'wrote %s\n' "$FILE"
printf '\n'
sed -n '1,/^---$/p' "$FILE" | sed -n '2,$p' | sed '$d'
printf '\nmonty loads it on next start; run /skills -l to confirm.\n'
