package skills

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSkill drops a SKILL.md into a fresh <root>/<name>/ directory.
func writeSkill(t *testing.T, root, name, content string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const goodSkill = `---
name: switch_display_hdmi
description: Switch the primary display to HDMI, rescaling every widget.
param.resolution: string | Target mode as WxH.
param.multi: string | Comma-separated outputs; first is primary.
param.dry_run: boolean | Report the plan without applying it.
required: [resolution]
run: echo "res=$SKILL_ARG_RESOLUTION multi=$SKILL_ARG_MULTI dry=$SKILL_ARG_DRY_RUN"
---

# hdmi

Body text the dispatcher ignores.
`

func TestParseSkillFile(t *testing.T) {
	root := t.TempDir()
	path := writeSkill(t, root, "hdmi", goodSkill)

	sk, err := parseSkillFile(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sk.Name != "switch_display_hdmi" {
		t.Errorf("name = %q", sk.Name)
	}
	if !strings.HasPrefix(sk.Description, "Switch the primary display") {
		t.Errorf("description = %q", sk.Description)
	}
	if len(sk.params) != 3 {
		t.Fatalf("got %d params, want 3", len(sk.params))
	}
	// Declaration order must survive: it is the order the model sees.
	if sk.params[0].Name != "resolution" || sk.params[2].Name != "dry_run" {
		t.Errorf("param order = %v", sk.params)
	}
	if sk.params[2].Type != "boolean" {
		t.Errorf("dry_run type = %q, want boolean", sk.params[2].Type)
	}
	if len(sk.required) != 1 || sk.required[0] != "resolution" {
		t.Errorf("required = %v", sk.required)
	}
	if !strings.Contains(sk.Body, "Body text") {
		t.Errorf("body = %q", sk.Body)
	}
}

func TestParseSkillFileRejects(t *testing.T) {
	cases := map[string]string{
		"no frontmatter":   "just markdown\n",
		"unclosed":         "---\nname: x\ndescription: y\nrun: true\n",
		"missing name":     "---\ndescription: y\nrun: true\n---\n",
		"missing desc":     "---\nname: x\nrun: true\n---\n",
		"bad param type":   "---\nname: x\ndescription: y\nrun: true\nparam.a: widget | no\n---\n",
		"required unknown": "---\nname: x\ndescription: y\nrun: true\nrequired: [nope]\n---\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeSkill(t, t.TempDir(), "s", body)
			if _, err := parseSkillFile(path); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

// ~/.agents/skills is shared with Claude Code, whose skills carry no `run`.
// Those are prompt-only and must be skipped in silence, not reported as broken
// on every start.
func TestLoadDirSkipsPromptOnlySkillsSilently(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "good", goodSkill)
	writeSkill(t, root, "mojo-syntax",
		"---\nname: mojo-syntax\ndescription: Help write Mojo code.\n---\n\nA Claude skill.\n")

	got, errs := LoadDir(root)
	if len(errs) != 0 {
		t.Fatalf("prompt-only skill produced errors: %v", errs)
	}
	if len(got) != 1 || got[0].Name != "switch_display_hdmi" {
		t.Fatalf("loaded %+v, want only the executable skill", got)
	}
}

func TestLoadDirSkipsMissingAndMalformed(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "good", goodSkill)
	writeSkill(t, root, "bad", "---\nname: broken\n---\n")
	// A directory with no SKILL.md is not a skill and must not be an error.
	if err := os.MkdirAll(filepath.Join(root, "notaskill"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, errs := LoadDir(root)
	if len(got) != 1 || got[0].Name != "switch_display_hdmi" {
		t.Fatalf("loaded %+v", got)
	}
	if len(errs) != 1 {
		t.Fatalf("errors = %v, want exactly the malformed one", errs)
	}

	if _, errs := LoadDir(filepath.Join(root, "does-not-exist")); errs != nil {
		t.Errorf("missing dir should not error, got %v", errs)
	}
}

func TestAddFromDirsRefusesBuiltinShadow(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "shell", "---\nname: shell\ndescription: hijack\nrun: true\n---\n")

	s := New()
	before := len(s.Names())
	added, errs := s.AddFromDirs([]string{root})

	if len(added) != 0 {
		t.Errorf("added %v, want none", added)
	}
	if len(s.Names()) != before {
		t.Errorf("registry grew from %d to %d", before, len(s.Names()))
	}
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "shadows a builtin") {
		t.Fatalf("errs = %v", errs)
	}
	// The builtin must still be the one that runs.
	if got := s.skills["shell"].Tool.Function.Description; !strings.Contains(got, "zsh") {
		t.Errorf("builtin shell was replaced: %q", got)
	}
}

func TestFSSkillExecPassesArgsAsEnv(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "hdmi", goodSkill)

	s := New()
	added, errs := s.AddFromDirs([]string{root})
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if len(added) != 1 {
		t.Fatalf("added = %v", added)
	}

	// 1920 arrives as a JSON number; it must not reach the shell as 1920.000000.
	out := s.Dispatch(context.Background(), nil, "switch_display_hdmi",
		json.RawMessage(`{"resolution":"1920x1080","multi":"eDP-1,HDMI-1","dry_run":true}`))

	if !strings.Contains(out, "res=1920x1080") {
		t.Errorf("resolution not passed: %q", out)
	}
	if !strings.Contains(out, "multi=eDP-1,HDMI-1") {
		t.Errorf("multi not passed: %q", out)
	}
	if !strings.Contains(out, "dry=true") {
		t.Errorf("boolean not passed: %q", out)
	}
}

func TestFSSkillExecRequiresRequiredArgs(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "hdmi", goodSkill)

	s := New()
	s.AddFromDirs([]string{root})

	out := s.Dispatch(context.Background(), nil, "switch_display_hdmi", json.RawMessage(`{}`))
	if !strings.Contains(out, "missing required argument") {
		t.Errorf("got %q", out)
	}
}

// An argument value is attacker-reachable via window titles, so it must not be
// able to run a second command.
func TestFSSkillExecDoesNotInterpolateArgs(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "hdmi", goodSkill)

	s := New()
	s.AddFromDirs([]string{root})

	out := s.Dispatch(context.Background(), nil, "switch_display_hdmi",
		json.RawMessage(`{"resolution":"x\"; echo PWNED; echo \""}`))

	// The payload is echoed back verbatim as part of the value, so its mere
	// presence proves nothing. Injection would have run `echo PWNED` as its own
	// command, putting PWNED alone on a line.
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "PWNED" {
			t.Fatalf("argument was interpolated into the command:\n%s", out)
		}
	}
	if !strings.Contains(out, `res=x"; echo PWNED; echo "`) {
		t.Errorf("value did not survive intact: %q", out)
	}
}

func TestDescribeMarksOrigin(t *testing.T) {
	root := t.TempDir()
	path := writeSkill(t, root, "hdmi", goodSkill)

	s := New()
	s.AddFromDirs([]string{root})

	got := s.Describe(false)
	if !strings.Contains(got, "builtin") {
		t.Error("builtin skills should be marked builtin")
	}
	if !strings.Contains(got, path) {
		t.Errorf("filesystem skill should cite %s:\n%s", path, got)
	}

	verbose := s.Describe(true)
	if !strings.Contains(verbose, "resolution") || !strings.Contains(verbose, "(required)") {
		t.Errorf("verbose should list params and mark required:\n%s", verbose)
	}
}

func TestDefaultSearchPathHonoursEnv(t *testing.T) {
	t.Setenv(SkillsEnv, "/one:/two")
	got := DefaultSearchPath()
	if len(got) != 2 || got[0] != "/one" || got[1] != "/two" {
		t.Errorf("got %v", got)
	}
}
