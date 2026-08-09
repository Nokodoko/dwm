package skills

import (
	"encoding/json"
	"strings"
	"testing"
)

func loadTestSkill(t *testing.T) FSSkill {
	t.Helper()
	root := t.TempDir()
	writeSkill(t, root, "hdmi", goodSkill)
	got, errs := LoadDir(root)
	if len(errs) != 0 || len(got) != 1 {
		t.Fatalf("load: %v %v", got, errs)
	}
	return got[0]
}

func TestSlugFromDirectory(t *testing.T) {
	sk := loadTestSkill(t)
	if sk.Slug != "hdmi" {
		t.Errorf("Slug = %q, want hdmi (the directory, not the tool name %q)", sk.Slug, sk.Name)
	}
}

func TestLookupSlashMatchesSlugAndName(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "hdmi", goodSkill)
	s := New()
	s.AddFromDirs([]string{root})

	for _, in := range []string{"/hdmi", "hdmi", "switch_display_hdmi", "/switch_display_hdmi"} {
		if _, ok := s.LookupSlash(in); !ok {
			t.Errorf("LookupSlash(%q) missed", in)
		}
	}
	if _, ok := s.LookupSlash("/nope"); ok {
		t.Error("LookupSlash matched a command that does not exist")
	}
	if _, ok := s.LookupSlash(""); ok {
		t.Error("empty command should not match")
	}
}

func TestParseSlashArgs(t *testing.T) {
	sk := loadTestSkill(t)

	cases := []struct {
		in   string
		want map[string]any
	}{
		{"", map[string]any{}},
		{"-r 1920x1080", map[string]any{"resolution": "1920x1080"}},
		{"--resolution 1920x1080", map[string]any{"resolution": "1920x1080"}},
		{"--resolution=1920x1080", map[string]any{"resolution": "1920x1080"}},
		{"resolution=1920x1080", map[string]any{"resolution": "1920x1080"}},
		{"-m hdmi,edp", map[string]any{"multi": "hdmi,edp"}},
		// A bare string flag: "-m" alone means multi, sort it out.
		{"-m", map[string]any{"multi": "true"}},
		// A declared boolean needs no value.
		{"--dry-run", map[string]any{"dry_run": true}},
		{"--dry_run", map[string]any{"dry_run": true}},
		{"-r 1920x1080 -m hdmi,edp --dry-run", map[string]any{
			"resolution": "1920x1080", "multi": "hdmi,edp", "dry_run": true,
		}},
	}

	for _, c := range cases {
		t.Run("["+c.in+"]", func(t *testing.T) {
			raw, err := ParseSlashArgs(sk, c.in)
			if err != nil {
				t.Fatalf("ParseSlashArgs(%q): %v", c.in, err)
			}
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for k, v := range c.want {
				if got[k] != v {
					t.Errorf("%s = %#v, want %#v", k, got[k], v)
				}
			}
		})
	}
}

func TestParseSlashArgsRejects(t *testing.T) {
	sk := loadTestSkill(t)

	for _, in := range []string{
		"--nosuchflag x",
		"nosuch=1",
		"bareword",
	} {
		if _, err := ParseSlashArgs(sk, in); err == nil {
			t.Errorf("ParseSlashArgs(%q) should have failed", in)
		}
	}
}

// An ambiguous short flag must not be guessed at: picking one of two
// parameters silently is how a display switch does the wrong thing.
func TestParseSlashArgsRefusesAmbiguousPrefix(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "amb", "---\nname: amb\ndescription: d\n"+
		"param.mode: string | one\nparam.monitor: string | two\nrun: true\n---\n")
	got, errs := LoadDir(root)
	if len(errs) != 0 {
		t.Fatal(errs)
	}

	_, err := ParseSlashArgs(got[0], "-m x")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("err = %v, want an ambiguity error", err)
	}
}

func TestSlashNames(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "hdmi", goodSkill)
	s := New()
	s.AddFromDirs([]string{root})

	got := s.SlashNames()
	if len(got) != 1 || got[0] != "/hdmi" {
		t.Errorf("SlashNames = %v", got)
	}
}
