# Spec Review Report

| Field | Value |
|-------|-------|
| Spec | `/home/n0ko/bling/dwm/SPEC.md` |
| Reviewed | 2026-04-08T14:57:21Z |
| Iteration | 1 of 3 |
| Verdict | **PASS WITH WARNINGS** (0 critical, 4 warnings, 3 info) |

## Summary

The spec is thorough and well-structured for a deployment/porting task. All source files exist on monty, all 9 SHA-256 checksums verified correct against disk, the Rule struct fields match the spec's window rule entries exactly (12 fields: class, instance, title, tags, isfloating, monitor, iscentered, bw, borderscheme, bordertitle, floatw, floath), and the dependency graph is acyclic with correct ordering. The two blocking open questions (lewis unreachable, lewis IP unknown) are correctly identified as prerequisites. Four warnings relate to: the Data Model not referencing the critical Rule struct, a missing checksum for wezterm-egl-fix.sh, a fragile verify command for T10/T11, and unresolved open questions not gated in the task graph. No critical issues found.

## Verified Claims

The following spec claims were verified against the filesystem:

- **All 11 source files exist on monty** at the paths specified in On-Disk Format
- **All 9 SHA-256 checksums match** (vivaldi-leader.sh, vivaldi_base.py, vivaldi_leader.py, tabTiler.py, tabkill.py, helpers.py, sb-session, vivaldi-stable.conf, xkeyread binary)
- **monty config.h already contains all vivaldi entries**: vivaldileadercmd array (line 197), keybinding (line 218), Vivaldi-stable rule (line 80), wezterm-tabtiler rule (line 76)
- **monty blocks.h already contains sb-session** at signal 10 (line 8)
- **Rule struct has 12 fields** matching the spec's window rule column layout (dwm.c lines 141-154)
- **SchemeOlr is a valid identifier** in config.h (line 26)
- **xkeyread exits 1 on timeout** confirmed in source (xkeyread.c line 7)
- **xkeyread Makefile build command** matches spec: `cc -O2 -o xkeyread xkeyread.c -lX11`
- **helpers.py uses match/case** (lines 23-35), confirming Python 3.10+ requirement
- **lewis is NOT in ~/.ssh/config** -- confirmed absent, consistent with Open Question #1
- **Actual LOC** (1,873 total) aligns with spec estimate (~1,882)
- **Dependency graph is acyclic** -- T1 -> {T2,T4,T5,T7,T8,T9} -> T2->T3 -> {T6,T10,T11} -> T12

## Dimension Results

| Dimension | Status | Findings | Critical |
|-----------|--------|----------|----------|
| completeness | PASS | 2 | 0 |
| clarity | PASS | 1 | 0 |
| correctness | PASS | 1 | 0 |
| consistency | PASS | 2 | 0 |
| sdlc | PASS | 0 | 0 |
| actionability | PASS | 2 | 0 |
| rebuild fidelity | N/A | 0 | 0 |

## Findings

### Dimension 1: Completeness

### COMP-001 [WARNING] -- Data Model section is a justified N/A but omits the critical Rule struct schema

**Section:** Data Model
**Issue:** The section reads "Not applicable. This is a deployment/porting spec with no structured data entities." While the justification is reasonable, the spec _does_ depend on a data structure -- the `Rule` struct from dwm.c -- which is critical to the correctness of config.h patches. The 12-field Rule struct (`class, instance, title, tags, isfloating, monitor, iscentered, bw, borderscheme, bordertitle, floatw, floath`) is documented implicitly in the config.h entries section and the Failure Modes table, but not formalized. Open Question #3 specifically calls out struct mismatch risk, yet the struct definition is never shown.
**Suggestion:** Add a brief subsection in Data Model showing the Rule typedef from dwm.c (lines 141-154) as the schema that config.h window rules must conform to on lewis. This surfaces the struct mismatch risk (Open Question #3) at the point where an agent would look for data definitions.

### COMP-002 [INFO] -- File Integrity Checksums table omits xkeyread source files and wezterm-egl-fix.sh

**Section:** Success Criteria / File Integrity Checksums
**Issue:** The checksums table includes the xkeyread binary (`~/.local/bin/xkeyread`) but omits the source files that are actually transferred: `~/.local/src/xkeyread/xkeyread.c` and `~/.local/src/xkeyread/Makefile`. The binary checksum is not useful for verification on lewis since xkeyread is compiled from source there (producing a different binary). The table also omits `~/scripts/wezterm-egl-fix.sh` which is transferred in T4.
**Suggestion:** Add SHA-256 entries for `xkeyread.c`, `Makefile`, and `wezterm-egl-fix.sh`. Consider removing or annotating the xkeyread binary entry as monty-reference-only.

---

### Dimension 2: Clarity

### CLAR-001 [WARNING] -- Ambiguous package manager assumption in Open Question #2

**Section:** Open Questions, row #2
**Issue:** The suggested default states: "Assume Arch Linux (pacman) since monty appears to be Arch-based." The word "appears" is a hedge word. T3's description says "Install missing system packages on lewis (based on T2 probe results)" without specifying which package manager or install commands to use. If the assumption is wrong, the unix-coder agent will need to improvise install commands for an unknown distribution.
**Suggestion:** Replace the suggested default with a deterministic procedure: "T2 probe must detect the package manager by running `which pacman apt dnf zypper 2>/dev/null` and record the result. T3 must use the detected manager. If no supported manager is found, T3 reports `Blocked: unknown package manager on lewis`."

---

### Dimension 3: Correctness

### CORR-001 [INFO] -- Estimated Size total LOC minor rounding discrepancy

**Section:** Estimated Size
**Issue:** The spec claims "Total ~1,882 LOC". Actual line counts verified on disk: 42 + 85 + 308 + 62 + 984 + 208 + 49 + 105 + 14 + 13 + 3 = 1,873. With ~6 lines added each for config.h and blocks.h patches: ~1,885. Neither matches 1,882 exactly.
**Suggestion:** No change required. The tilde (~) prefix makes this an acceptable approximation.

---

### Dimension 4: Consistency

### CONS-001 [WARNING] -- wezterm-egl-fix.sh missing from File Integrity Checksums

**Section:** Success Criteria / File Integrity Checksums vs On-Disk Format + Target State
**Issue:** `~/scripts/wezterm-egl-fix.sh` appears in the On-Disk Format (line 55), Target State (line 374), Migration table (line 151), and is transferred in T4. However, it has no entry in the File Integrity Checksums table (lines 497-508). Every other transferred file has a checksum entry.
**Suggestion:** Add a SHA-256 entry for `~/scripts/wezterm-egl-fix.sh` to the checksums table. The actual hash can be obtained from monty with `sha256sum ~/scripts/wezterm-egl-fix.sh`.

### CONS-002 [INFO] -- wezterm-egl-fix.sh line count approximation

**Section:** Estimated Size
**Issue:** The spec claims "~15" LOC for wezterm-egl-fix.sh. Actual: 13 lines. Minor discrepancy.
**Suggestion:** No change required.

---

### Dimension 5: SDLC Alignment

No findings. All 13 success criteria map to testable predicates: `contains_pattern` (grep-based config entry checks), `structural_check` (test -x executable presence), and exit-code verification. Every criterion is a concrete shell command that exits 0 on success.

---

### Dimension 6: Actionability

### ACTN-001 [WARNING] -- T10 and T11 Verify Commands use fragile grep-based success detection instead of exit codes

**Section:** Task Manifest, T10 and T11
**Issue:** T10's verify command is: `ssh lewis "cd ~/bling/dwm && make clean && make 2>&1 | tail -1 | grep -qv 'Error'"`. This checks that the _last line_ of make output does not contain the string "Error". This is fragile for three reasons: (1) if make fails on a non-final line, the check may still pass because `tail -1` only examines the last line; (2) `grep -qv 'Error'` inverts the match, so any line _not_ containing "Error" succeeds -- including a warning line; (3) the pipe to `grep` masks the exit code of `make` itself. T11 has the same pattern.
**Suggestion:** Replace both with simple exit-code checks. T10: `ssh lewis "cd ~/bling/dwm && make clean && make"`. T11: `ssh lewis "cd ~/bling/dwmblocks && make clean && make"`. The exit code of `make` is 0 on success and non-zero on failure, which is the correct signal.

### ACTN-002 [WARNING] -- Open Questions #1 and #6 block all execution but the task graph has no pre-execution gate

**Section:** Open Questions, Execution Order
**Issue:** Open Question #1 ("What is lewis's hostname or IP address? Lewis is not in `~/.ssh/config` and SSH connection failed with 'No route to host'.") and #6 ("Is lewis currently powered on and network-reachable? SSH probe failed during spec generation.") are both documented as blocking T1 and therefore all downstream tasks. However, the Execution Order begins directly with "Phase 1: T1 Add lewis to SSH config" with no gate to confirm these prerequisites are resolved. An agent dispatching this spec will attempt T1 immediately, fail at `ssh lewis hostname`, and waste an execution cycle.
**Suggestion:** Add a "Phase 0: Prerequisites" to the Execution Order: "User must provide (a) lewis's IP address or resolvable hostname, and (b) confirmation that lewis is powered on and network-reachable. T1 is blocked until both are confirmed." Alternatively, add a T0 task to the Task Manifest: `"T0 | unix-coder | Confirm lewis IP and reachability with user | -- | -- | -- | ping -c1 -W5 <lewis-ip>"` with T1 depending on T0.
