package skills

// Slash-command access to filesystem skills.
//
// A skill's tool name is written for the MODEL to choose well --
// `switch_display_edp` reads clearly in a tool list and is unambiguous next to
// its siblings. It is a miserable thing to type. The user who asked for these
// skills asked for `/edp`, `/hdmi`, `/dp`, and typing the tool name or
// describing the intent in prose to make the model pick it are both worse than
// the one-word command they already had in mind.
//
// So every filesystem skill is reachable two ways: the model calls it by
// `name`, and the user types `/<slug>`, where the slug is the skill's
// directory. Both routes end in the same Dispatch.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// LookupSlash resolves a typed command to a filesystem skill.
//
// It matches the slug first (`/edp`), then the full tool name
// (`/switch_display_edp`), so both work and the slug wins on a collision.
func (s *Set) LookupSlash(name string) (FSSkill, bool) {
	name = strings.TrimPrefix(strings.TrimSpace(name), "/")
	if name == "" {
		return FSSkill{}, false
	}
	for _, f := range s.fs {
		if f.Slug == name {
			return f, true
		}
	}
	for _, f := range s.fs {
		if f.Name == name {
			return f, true
		}
	}
	return FSSkill{}, false
}

// SlashNames lists the slash commands filesystem skills provide, for /help.
func (s *Set) SlashNames() []string {
	out := make([]string, 0, len(s.fs))
	for _, f := range s.fs {
		out = append(out, "/"+f.Slug)
	}
	return out
}

// ParseSlashArgs turns a typed argument string into skill arguments.
//
// It accepts the shapes a person actually types, rather than requiring JSON:
//
//	/hdmi                          -> {}
//	/hdmi -r 1920x1080             -> {"resolution": "1920x1080"}
//	/hdmi --resolution=1920x1080   -> same
//	/hdmi resolution=1920x1080     -> same
//	/hdmi -m hdmi,edp              -> {"multi": "hdmi,edp"}
//	/hdmi -m                       -> {"multi": "true"}   (declared boolean-ish)
//	/hdmi --dry-run                -> {"dry_run": true}
//
// Short flags resolve by unique prefix against the declared parameter names,
// which is what makes `-r` mean resolution and `-m` mean multi without either
// being spelled out anywhere. An ambiguous prefix is an error rather than a
// guess: silently picking one of two parameters is how a display switch ends
// up doing something the user did not ask for.
func ParseSlashArgs(sk FSSkill, arg string) (json.RawMessage, error) {
	out := map[string]any{}
	fields := strings.Fields(arg)

	setValue := func(pname, raw string) error {
		p, ok := sk.param(pname)
		if !ok {
			return fmt.Errorf("%s has no argument %q; it takes: %s", "/"+sk.Slug, pname, sk.paramList())
		}
		switch p.Type {
		case "integer":
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return fmt.Errorf("%s expects a whole number, got %q", pname, raw)
			}
			out[pname] = n
		case "number":
			f, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return fmt.Errorf("%s expects a number, got %q", pname, raw)
			}
			out[pname] = f
		case "boolean":
			b, err := strconv.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("%s expects true or false, got %q", pname, raw)
			}
			out[pname] = b
		default:
			out[pname] = raw
		}
		return nil
	}

	for i := 0; i < len(fields); i++ {
		tok := fields[i]

		// key=value, --key=value
		if k, v, ok := strings.Cut(tok, "="); ok && !strings.HasPrefix(tok, "-=") {
			name, err := sk.resolveParam(strings.TrimLeft(k, "-"))
			if err != nil {
				return nil, err
			}
			if err := setValue(name, v); err != nil {
				return nil, err
			}
			continue
		}

		if !strings.HasPrefix(tok, "-") {
			return nil, fmt.Errorf("unexpected %q; arguments look like -r 1920x1080 or resolution=1920x1080. %s takes: %s",
				tok, "/"+sk.Slug, sk.paramList())
		}

		name, err := sk.resolveParam(strings.TrimLeft(tok, "-"))
		if err != nil {
			return nil, err
		}
		p, _ := sk.param(name)

		// A following token that is not itself a flag is this flag's value.
		hasValue := i+1 < len(fields) && !strings.HasPrefix(fields[i+1], "-")

		switch {
		case p.Type == "boolean" && !hasValue:
			out[name] = true
		case hasValue:
			if err := setValue(name, fields[i+1]); err != nil {
				return nil, err
			}
			i++
		default:
			// A bare string flag with no value. `-m` on the display skills means
			// "multi, work the rest out yourself", and the scripts read that
			// value as boolean-ish, so "true" is the honest thing to send.
			out[name] = "true"
		}
	}

	if len(out) == 0 {
		return json.RawMessage("{}"), nil
	}
	return json.Marshal(out)
}

func (s FSSkill) param(name string) (fsParam, bool) {
	for _, p := range s.params {
		if p.Name == name {
			return p, true
		}
	}
	return fsParam{}, false
}

// resolveParam maps a typed flag to a declared parameter: exact name first,
// then dashes normalised to underscores, then a unique prefix.
func (s FSSkill) resolveParam(tok string) (string, error) {
	if tok == "" {
		return "", fmt.Errorf("empty flag")
	}
	if _, ok := s.param(tok); ok {
		return tok, nil
	}
	under := strings.ReplaceAll(tok, "-", "_")
	if _, ok := s.param(under); ok {
		return under, nil
	}

	var hits []string
	for _, p := range s.params {
		if strings.HasPrefix(p.Name, under) {
			hits = append(hits, p.Name)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return "", fmt.Errorf("/%s has no argument %q; it takes: %s", s.Slug, tok, s.paramList())
	default:
		return "", fmt.Errorf("%q is ambiguous — it could be %s", tok, strings.Join(hits, " or "))
	}
}

// paramList renders the declared parameters for an error message.
func (s FSSkill) paramList() string {
	if len(s.params) == 0 {
		return "(no arguments)"
	}
	out := make([]string, 0, len(s.params))
	for _, p := range s.params {
		out = append(out, p.Name)
	}
	return strings.Join(out, ", ")
}
