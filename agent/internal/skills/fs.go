package skills

// Filesystem skills: user-authored capabilities loaded from .agents/skills.
//
// The builtin skills in skills.go are thin wrappers over dwm IPC and are
// compiled in because they are structural -- they cannot change without the
// IPC command set changing. Everything else the user wants monty to do is
// policy, not structure, and recompiling the agent to teach it a new trick is
// the wrong shape. This file makes a directory of markdown files the second
// source of skills.
//
// The format is deliberately a Claude-compatible SKILL.md: YAML frontmatter
// delimited by `---`, with `name` and `description` as plain scalars so the
// same file is a valid skill for Claude Code and for monty. The monty-specific
// keys (`param.*`, `required`, `run`) are flat, dotted scalars rather than
// nested YAML so this package can parse them without a YAML dependency, and so
// Claude ignores them harmlessly.
//
//	---
//	name: switch_display_hdmi
//	description: Switch the primary display to HDMI, rescaling all widgets.
//	param.resolution: string | Target mode as WxH, e.g. 1920x1080. Optional.
//	param.multi: string | Comma-separated outputs; first is primary and leftmost.
//	required:
//	run: /home/n0ko/desktop-widgets/display-switch.sh hdmi
//	---
//
//	Markdown body: the human- and model-readable procedure. Ignored by the
//	dispatcher, read by Claude and shown by /skills --list --verbose.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Nokodoko/dwm-agent/internal/dwmipc"
)

// SkillsEnv names the environment variable that overrides the search path.
const SkillsEnv = "DWM_AGENT_SKILLS"

// argEnvPrefix prefixes every argument exported into a skill's environment.
//
// Arguments travel as environment variables rather than being interpolated
// into the command string. The model chooses these values and window titles
// reach the model as context, so a substituted argument is a shell injection
// waiting to happen; an environment variable is inert no matter what it holds.
const argEnvPrefix = "SKILL_ARG_"

// fsSkillTimeout bounds a filesystem skill the same way runShell bounds an
// ad-hoc command. Display switching restarts several programs and sleeps
// between them, so it needs more headroom than a shell one-liner.
const fsSkillTimeout = 120 * time.Second

// errNotExecutable marks a well-formed skill that monty cannot run, because it
// declares no `run` command. It is a filter, not a failure.
var errNotExecutable = errors.New("skill is prompt-only (no `run`)")

// FSSkill is a skill parsed from a SKILL.md file.
type FSSkill struct {
	Name        string
	Description string
	Body        string // markdown after the frontmatter
	Path        string // source file, for error messages and /skills --list
	Slug        string // directory name, usable as a /slash command
	Run         string // command line, evaluated by zsh -c

	params   []fsParam
	required []string
}

type fsParam struct {
	Name string
	Type string
	Desc string
}

// DefaultSearchPath returns the directories scanned for SKILL.md files.
//
// $DWM_AGENT_SKILLS overrides it entirely (colon-separated, like $PATH).
// Otherwise the live dwm tree comes first, then the user-global directory, so
// a checkout can shadow a personal skill of the same name during development.
func DefaultSearchPath() []string {
	if v := os.Getenv(SkillsEnv); v != "" {
		return filepath.SplitList(v)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, "bling", "dwm-pertag", ".agents", "skills"),
		filepath.Join(home, "bling", "dwm", ".agents", "skills"),
		filepath.Join(home, ".agents", "skills"),
	}
}

// LoadDir parses every <dir>/*/SKILL.md and every <dir>/*.md.
//
// A missing directory is not an error: the search path lists candidates, and
// most users will have only one of them. Malformed skills are returned as
// errors alongside the ones that did parse, so one bad file cannot silence the
// rest.
func LoadDir(dir string) ([]FSSkill, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, []error{fmt.Errorf("%s: %w", dir, err)}
	}

	var out []FSSkill
	var errs []error
	for _, e := range entries {
		var path string
		switch {
		case e.IsDir():
			path = filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(path); err != nil {
				continue // a directory without a SKILL.md is not a skill
			}
		case strings.HasSuffix(e.Name(), ".md") && e.Name() != "README.md":
			path = filepath.Join(dir, e.Name())
		default:
			continue
		}

		sk, err := parseSkillFile(path)
		if errors.Is(err, errNotExecutable) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, sk)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, errs
}

// parseSkillFile reads one SKILL.md.
func parseSkillFile(path string) (FSSkill, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return FSSkill{}, fmt.Errorf("%s: %w", path, err)
	}

	front, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return FSSkill{}, fmt.Errorf("%s: %w", path, err)
	}

	// The slug is what the user types as a slash command. A skill's tool name
	// is written for the model to choose well (switch_display_edp), which makes
	// a poor thing to type; the containing directory is the short name the user
	// already thinks in (.agents/skills/edp -> /edp).
	slug := filepath.Base(filepath.Dir(path))
	if filepath.Base(path) != "SKILL.md" {
		slug = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	sk := FSSkill{Path: path, Slug: slug, Body: strings.TrimSpace(body)}
	for _, line := range strings.Split(front, "\n") {
		key, val, ok := splitKeyValue(line)
		if !ok {
			continue
		}
		switch {
		case key == "name":
			sk.Name = val
		case key == "description":
			sk.Description = val
		case key == "run":
			sk.Run = val
		case key == "required":
			sk.required = splitList(val)
		case strings.HasPrefix(key, "param."):
			p, err := parseParam(strings.TrimPrefix(key, "param."), val)
			if err != nil {
				return FSSkill{}, fmt.Errorf("%s: %w", path, err)
			}
			sk.params = append(sk.params, p)
		}
	}

	// The name is what the model calls and what collides with builtins, and the
	// description is the only thing it uses to decide when to call it. A skill
	// missing either is not dispatchable, so refuse it loudly at load rather
	// than advertising a tool that cannot work.
	if sk.Name == "" {
		return FSSkill{}, fmt.Errorf("%s: frontmatter has no `name`", path)
	}
	if sk.Description == "" {
		return FSSkill{}, fmt.Errorf("%s: skill %q has no `description`", path, sk.Name)
	}
	// A skill with no `run` is prompt-only: instructions for a model to read,
	// with nothing for monty to execute. ~/.agents/skills is shared with Claude
	// Code, whose skills are all of this kind, so treating a missing `run` as an
	// error made every monty start print a screenful of complaints about files
	// that were never meant for it. Skip them quietly instead.
	if sk.Run == "" {
		return FSSkill{}, errNotExecutable
	}
	for _, r := range sk.required {
		if !sk.hasParam(r) {
			return FSSkill{}, fmt.Errorf("%s: skill %q requires %q, which is not a declared param", path, sk.Name, r)
		}
	}
	return sk, nil
}

func (s FSSkill) hasParam(name string) bool {
	for _, p := range s.params {
		if p.Name == name {
			return true
		}
	}
	return false
}

// splitFrontmatter separates the leading `---` block from the markdown body.
func splitFrontmatter(s string) (front, body string, err error) {
	// A BOM or leading blank lines would otherwise defeat the prefix check,
	// and hand-authored files pick those up easily.
	s = strings.TrimPrefix(s, "\uFEFF")
	t := strings.TrimLeft(s, " \t\r\n")
	if !strings.HasPrefix(t, "---") {
		return "", "", errors.New("no `---` frontmatter block")
	}
	rest := t[3:]
	rest = strings.TrimPrefix(rest, "\r")
	rest = strings.TrimPrefix(rest, "\n")

	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", errors.New("frontmatter block is not closed with `---`")
	}
	front = rest[:end]
	body = rest[end+len("\n---"):]
	if i := strings.Index(body, "\n"); i >= 0 {
		body = body[i+1:]
	} else {
		body = ""
	}
	return front, body, nil
}

// splitKeyValue parses one `key: value` frontmatter line.
//
// Comments and blank lines are skipped. Values may be quoted, which is how a
// description containing a colon stays readable as YAML.
func splitKeyValue(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	i := strings.Index(line, ":")
	if i < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:i])
	val = strings.TrimSpace(line[i+1:])
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
	}
	return key, val, key != ""
}

// splitList parses `required: [a, b]` or `required: a, b`.
func splitList(v string) []string {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(v), "["), "]"))
	if v == "" {
		return nil
	}
	var out []string
	for _, f := range strings.Split(v, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// parseParam parses `param.<name>: <type> | <description>`.
func parseParam(name, val string) (fsParam, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return fsParam{}, errors.New("param with an empty name")
	}
	typ, desc := "string", ""
	if i := strings.Index(val, "|"); i >= 0 {
		typ = strings.TrimSpace(val[:i])
		desc = strings.TrimSpace(val[i+1:])
	} else if v := strings.TrimSpace(val); v != "" {
		typ = v
	}
	switch typ {
	case "string", "integer", "number", "boolean":
	default:
		return fsParam{}, fmt.Errorf("param %q has unsupported type %q (want string, integer, number or boolean)", name, typ)
	}
	return fsParam{Name: name, Type: typ, Desc: desc}, nil
}

// AddFromDirs loads every directory in the search path and registers what it
// finds. It returns the skills it added and any load errors.
//
// A filesystem skill may not shadow a builtin: the builtins are the model's
// only route to dwm itself, and a typo in a markdown file silently breaking
// window management would be a miserable failure mode.
func (s *Set) AddFromDirs(dirs []string) ([]FSSkill, []error) {
	var added []FSSkill
	var errs []error
	seen := map[string]string{}

	for _, dir := range dirs {
		found, loadErrs := LoadDir(dir)
		errs = append(errs, loadErrs...)

		for _, fs := range found {
			if _, isBuiltin := s.skills[fs.Name]; isBuiltin {
				if prev, ok := seen[fs.Name]; ok {
					errs = append(errs, fmt.Errorf("%s: skill %q already loaded from %s", fs.Path, fs.Name, prev))
				} else {
					errs = append(errs, fmt.Errorf("%s: skill %q shadows a builtin; rename it", fs.Path, fs.Name))
				}
				continue
			}
			s.addFS(fs)
			seen[fs.Name] = fs.Path
			added = append(added, fs)
		}
	}
	return added, errs
}

// addFS registers one filesystem skill as a callable tool.
func (s *Set) addFS(fs FSSkill) {
	props := obj{}
	for _, p := range fs.params {
		props[p.Name] = prop(p.Type, p.Desc)
	}
	req := fs.required
	if req == nil {
		req = []string{}
	}

	s.fs = append(s.fs, fs)
	s.add(fs.Name, fs.Description, schema(props, req...),
		func(ctx context.Context, _ *dwmipc.Conn, raw json.RawMessage) (string, error) {
			return fs.exec(ctx, raw)
		})
}

// exec runs the skill's `run` command with its arguments in the environment.
func (fs FSSkill) exec(ctx context.Context, raw json.RawMessage) (string, error) {
	args := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return "", fmt.Errorf("bad arguments: %w", err)
		}
	}
	for _, r := range fs.required {
		if v, ok := args[r]; !ok || v == nil || v == "" {
			return "", fmt.Errorf("missing required argument %q", r)
		}
	}

	env := os.Environ()
	for _, p := range fs.params {
		v, ok := args[p.Name]
		if !ok || v == nil {
			continue
		}
		env = append(env, argEnvPrefix+strings.ToUpper(p.Name)+"="+scalarString(v))
	}
	// The skill's own directory is a useful anchor for a `run` that calls a
	// sibling script, and it is where a reader would expect relative paths to
	// resolve from.
	env = append(env, "SKILL_DIR="+filepath.Dir(fs.Path))

	cctx, cancel := context.WithTimeout(ctx, fsSkillTimeout)
	defer cancel()

	// Non-interactive, unlike runShell: a skill's `run` is authored text, not
	// something the user typed, so it has no business depending on .zshrc
	// aliases -- and skipping .zshrc skips the startup noise runShell has to
	// strip with a marker.
	cmd := exec.CommandContext(cctx, zshPath(), "-c", fs.Run)
	cmd.Env = env
	if home, err := os.UserHomeDir(); err == nil {
		cmd.Dir = home
	}
	out, err := cmd.CombinedOutput()

	body := strings.TrimRight(string(out), "\n")
	if len(body) > shellMaxOutput {
		body = body[:shellMaxOutput] + fmt.Sprintf("\n… truncated at %d bytes", shellMaxOutput)
	}
	if body == "" {
		body = "(no output)"
	}

	if cctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timed out after %s; partial output:\n%s", fsSkillTimeout, body)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return fmt.Sprintf("exit %d\n%s", exitErr.ExitCode(), body), nil
	}
	if err != nil {
		return "", fmt.Errorf("could not run: %w", err)
	}
	return body, nil
}

// scalarString renders a JSON argument for the environment.
//
// encoding/json decodes every number into a float64, so an integer argument
// would otherwise reach the script as "1920.000000" and break a mode string.
func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	default:
		return fmt.Sprint(t)
	}
}
