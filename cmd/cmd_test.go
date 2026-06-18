package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeNote drops a file at rel (slash path) under dir, creating parents.
func writeNote(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// vault makes a minimal but valid vault dir (has the _format.md marker so
// vaultpath.Resolve walk-up / IsVault agree) and points VKIT_VAULT at it for the
// duration of the test. Returns the absolute vault path.
func vault(t *testing.T) string {
	t.Helper()
	v := t.TempDir()
	writeNote(t, v, "_format.md", "# format\n")
	t.Setenv("VKIT_VAULT", v)
	t.Setenv("VAULT_OBSIDIAN_CLI", "0") // no Obsidian running in tests; force Tier B
	return v
}

// runRoot drives the package-level rootCmd with args, capturing the cobra
// command's own writer output (help text, command errors) into one buffer and
// returning (output, error). It resets the shared --vault flag so repeated runs
// don't leak state from each other.
//
// NOTE (characterized behavior): the leaf commands (moc, note, sync, doctor,
// init, update) print their human output via fmt.Printf to the real os.Stdout,
// NOT via cmd.OutOrStdout(). So this buffer only captures cobra-generated text
// (usage/help and the framework's argument-validation errors). Use
// runRootCapture to assert on a command's fmt.Printf output.
func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	vaultFlag = "" // reset persistent flag state between runs
	// cobra's auto-added --help/--version flags are sticky on the shared rootCmd:
	// once a prior run parses --help, its Changed/value persists and short-circuits
	// later Execute calls. Reset them so each run starts clean.
	for _, name := range []string{"help", "version"} {
		if f := rootCmd.Flags().Lookup(name); f != nil {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		}
	}
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

// runRootCapture is runRoot but it also redirects the process's real os.Stdout
// through an OS pipe so the commands that print via fmt.Printf can be asserted
// on. Returns the captured stdout and the command error.
func runRootCapture(t *testing.T, args ...string) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	_, runErr := runRoot(t, args...)

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String(), runErr
}

// ----------------------------------------------------------------------------
// root command wiring
// ----------------------------------------------------------------------------

func TestRootCommandsRegistered(t *testing.T) {
	want := []string{"init", "update", "validate", "note", "rename", "sync", "doctor"}
	have := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		have[c.Name()] = true
	}
	for _, w := range want {
		if !have[w] {
			t.Errorf("subcommand %q not registered", w)
		}
	}
}

func TestRootFlags(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("vault") == nil {
		t.Error("--vault persistent flag missing")
	}
	if !rootCmd.SilenceUsage {
		t.Error("SilenceUsage expected true")
	}
	if !rootCmd.SilenceErrors {
		t.Error("SilenceErrors expected true")
	}
}

func TestRootHelp(t *testing.T) {
	out, err := runRoot(t, "--help")
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if !strings.Contains(out, "vkit") {
		t.Errorf("help output missing program name: %q", out)
	}
	// every registered subcommand should appear in help text
	for _, name := range []string{"init", "validate", "note", "sync", "doctor"} {
		if !strings.Contains(out, name) {
			t.Errorf("help missing subcommand %q", name)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	_, err := runRoot(t, "nope-not-a-command")
	if err == nil {
		t.Error("expected error for unknown command, got nil")
	}
}

// ----------------------------------------------------------------------------
// flag parsing per subcommand (parse-only; we assert the flags exist + bind)
// ----------------------------------------------------------------------------

func TestSubcommandFlagsDefined(t *testing.T) {
	cases := []struct {
		cmd, flag string
	}{
		{"validate", "staged"},
		{"note", "title"},
		{"note", "tags"},
		{"sync", "message"},
		{"update", "force"},
		{"update", "keep"},
		{"update", "dry-run"},
	}
	for _, tc := range cases {
		var found bool
		for _, c := range rootCmd.Commands() {
			if c.Name() == tc.cmd {
				if c.Flags().Lookup(tc.flag) == nil {
					t.Errorf("%s: flag --%s not defined", tc.cmd, tc.flag)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not found", tc.cmd)
		}
	}
}

// ----------------------------------------------------------------------------
// version — subcommand + --version flag both surface the injected build version
// ----------------------------------------------------------------------------

func TestVersionSubcommand(t *testing.T) {
	old := Version
	setVersion("9.9.9-test")
	defer setVersion(old)

	out, err := runRoot(t, "version")
	if err != nil {
		t.Fatalf("version error: %v (out=%q)", err, out)
	}
	if !strings.Contains(out, "9.9.9-test") {
		t.Errorf("version output missing injected version: %q", out)
	}
}

func TestVersionFlag(t *testing.T) {
	old := Version
	setVersion("1.2.3-flag")
	defer setVersion(old)

	out, err := runRoot(t, "--version")
	if err != nil {
		t.Fatalf("--version error: %v (out=%q)", err, out)
	}
	if !strings.Contains(out, "1.2.3-flag") {
		t.Errorf("--version output missing injected version: %q", out)
	}
}

func TestVersionDefault(t *testing.T) {
	if Version != "dev" {
		t.Errorf("default Version = %q, want \"dev\"", Version)
	}
}

// ----------------------------------------------------------------------------
// validate — success path (exit 0 / nil error) via direct RunE
// ----------------------------------------------------------------------------

func TestValidateSuccessPrintsCount(t *testing.T) {
	v := vault(t)
	writeNote(t, v, "good.md", "---\nupdated: 2026-06-14\n---\n\n# Good\n")
	out, err := runRootCapture(t, "validate")
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if !strings.Contains(out, "notes valid") {
		t.Errorf("validate success missing 'notes valid':\n%s", out)
	}
}

func TestValidatePassesOnCleanVault(t *testing.T) {
	v := vault(t)
	writeNote(t, v, "good.md", "---\nupdated: 2026-06-14\ntags: [x]\n---\n\n# Good Note\n\nbody [[link]]\n")

	out, err := runRoot(t, "validate")
	if err != nil {
		t.Fatalf("validate on clean vault returned error: %v (out=%q)", err, out)
	}
}

func TestValidateExplicitFileArg(t *testing.T) {
	v := vault(t)
	writeNote(t, v, "good.md", "---\nupdated: 2026-06-14\n---\n\n# Good Note\n")
	out, err := runRoot(t, "validate", filepath.Join(v, "good.md"))
	if err != nil {
		t.Fatalf("validate <file> returned error: %v (out=%q)", err, out)
	}
}

// ----------------------------------------------------------------------------
// validate — failure path uses os.Exit(1), so re-exec the binary and assert the
// process exit code. This is the highest-value characterization: validation
// failure must terminate non-zero.
// ----------------------------------------------------------------------------

func TestValidateExitsNonZeroOnFailure(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	v := t.TempDir()
	writeNote(t, v, "_format.md", "# format\n")
	// nofront.md has no frontmatter -> validate.Files reports a problem -> exit 1.
	writeNote(t, v, "bad.md", "# No Frontmatter\n\nbody\n")

	repoRoot := repoRoot(t)
	cmd := exec.Command("go", "run", ".", "validate")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "VKIT_VAULT="+v)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		t.Fatalf("expected non-zero exit on validation failure, got success (stderr=%q)", stderr.String())
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if ee.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", ee.ExitCode())
	}
	if !strings.Contains(stderr.String(), "Validation failed") {
		t.Errorf("stderr missing failure message: %q", stderr.String())
	}
}

func TestValidateExitsZeroOnCleanVault(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	v := t.TempDir()
	writeNote(t, v, "_format.md", "# format\n")
	writeNote(t, v, "good.md", "---\nupdated: 2026-06-14\n---\n\n# Good\n")

	cmd := exec.Command("go", "run", ".", "validate")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "VKIT_VAULT="+v)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("expected exit 0 on clean vault, got err=%v out=%q", err, out)
	}
}

// repoRoot walks up from the test's CWD to the dir holding go.mod (the module
// root), so `go run .` builds the whole binary.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from test dir")
		}
		dir = parent
	}
}

// ----------------------------------------------------------------------------
// note — portable scaffold path (no obsidian, no native mode)
// ----------------------------------------------------------------------------

func TestNoteCreatesScaffold(t *testing.T) {
	v := vault(t)
	out, err := runRoot(t, "note", "projects/my-cool-note.md")
	if err != nil {
		t.Fatalf("note error: %v (out=%q)", err, out)
	}
	b, err := os.ReadFile(filepath.Join(v, "projects", "my-cool-note.md"))
	if err != nil {
		t.Fatalf("note file not created: %v", err)
	}
	body := string(b)
	// title derived from filename (kebab -> Title Case)
	if !strings.Contains(body, "# My Cool Note") {
		t.Errorf("derived title wrong:\n%s", body)
	}
	// frontmatter + body skeleton present
	for _, want := range []string{"updated:", "## Summary", "## Notes", "## Related"} {
		if !strings.Contains(body, want) {
			t.Errorf("note missing %q:\n%s", want, body)
		}
	}
}

func TestNoteTitleAndTagsFlags(t *testing.T) {
	v := vault(t)
	_, err := runRoot(t, "note", "n.md", "--title", "Explicit Title", "--tags", "a, b ,c")
	if err != nil {
		t.Fatalf("note error: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(v, "n.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "# Explicit Title") {
		t.Errorf("explicit --title not used:\n%s", body)
	}
	// tags are trimmed and comma-joined into the frontmatter list
	if !strings.Contains(body, "tags: [a, b, c]") {
		t.Errorf("tags not rendered as expected:\n%s", body)
	}
}

func TestNoteOutputShowsTitleAndPath(t *testing.T) {
	vault(t)
	out, err := runRootCapture(t, "note", "projects/my-note.md", "--title", "My Note")
	if err != nil {
		t.Fatalf("note error: %v", err)
	}
	for _, want := range []string{"My Note", "projects/my-note.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("note output missing %q:\n%s", want, out)
		}
	}
}

func TestNoteOutputShowsTags(t *testing.T) {
	vault(t)
	out, err := runRootCapture(t, "note", "tagged.md", "--tags", "a,b")
	if err != nil {
		t.Fatalf("note error: %v", err)
	}
	if !strings.Contains(out, "tags") {
		t.Errorf("note output missing 'tags' row:\n%s", out)
	}
}

func TestNoteOutputRoutedPath(t *testing.T) {
	v := vault(t)
	obsDir := filepath.Join(v, ".obsidian")
	if err := os.MkdirAll(obsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(obsDir, "app.json"),
		[]byte(`{"newFileLocation":"folder","newFileFolderPath":"inbox"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runRootCapture(t, "note", "bare-stem.md")
	if err != nil {
		t.Fatalf("note error: %v", err)
	}
	if !strings.Contains(out, "inbox") {
		t.Errorf("note output should show routed folder 'inbox':\n%s", out)
	}
}

func TestNoteRefusesOverwrite(t *testing.T) {
	v := vault(t)
	writeNote(t, v, "exists.md", "---\nupdated: 2026-06-14\n---\n\n# Exists\n")
	_, err := runRoot(t, "note", "exists.md")
	if err == nil {
		t.Fatal("expected error when target note already exists")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("unexpected overwrite error: %v", err)
	}
}

func TestNoteRequiresExactlyOneArg(t *testing.T) {
	vault(t)
	if _, err := runRoot(t, "note"); err == nil {
		t.Error("expected error with zero args (ExactArgs(1))")
	}
}

// ----------------------------------------------------------------------------
// doctor — environment report; assert the fixed label keys are present
// ----------------------------------------------------------------------------

func TestDoctorOutput(t *testing.T) {
	vault(t)
	out, err := runRootCapture(t, "doctor")
	if err != nil {
		t.Fatalf("doctor error: %v", err)
	}
	for _, key := range []string{"System", "Obsidian", "OS", "Pkg mgr", "systemd", "TTY", "Binary", "CLI mode", "Vault"} {
		if !strings.Contains(out, key) {
			t.Errorf("doctor output missing %q:\n%s", key, out)
		}
	}
}

// ----------------------------------------------------------------------------
// sync — git-dependent; characterizes the rebuild + commit-docs path. Skipped if
// git is unavailable.
// ----------------------------------------------------------------------------

func TestSyncRebuildsAndCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	v := vault(t)
	// init a git repo so `git add` / commit have somewhere to land
	mustGit(t, v, "init", "-q")
	mustGit(t, v, "config", "user.name", "test")
	mustGit(t, v, "config", "user.email", "test@local")
	// sync stages a FIXED pathspec set: "*.md" and the dirs
	// decisions/ infrastructure/ projects/ reference/. git treats a literal
	// pathspec that matches nothing as fatal and aborts the whole `git add`, so
	// all four dirs must exist for the stage (and thus the commit) to happen.
	for _, d := range []string{"decisions", "infrastructure", "projects", "reference"} {
		writeNote(t, v, d+"/.keep", "")
	}
	writeNote(t, v, "projects/p.md", "---\nupdated: 2026-06-14\n---\n\n# P\n")

	out, err := runRootCapture(t, "sync", "-m", "vault: test sync")
	if err != nil {
		t.Fatalf("sync error: %v (out=%q)", err, out)
	}
	for _, want := range []string{"Staged", "Committed"} {
		if !strings.Contains(out, want) {
			t.Errorf("sync output missing %q:\n%s", want, out)
		}
	}
	// a commit should now exist with our message
	log := gitOut(t, v, "log", "--oneline")
	if !strings.Contains(log, "vault: test sync") {
		t.Errorf("expected commit message in log, got:\n%s", log)
	}
}

// TestSyncSkipsMissingDir: sync must still commit when one of the named note
// dirs is absent — it should skip the missing dir rather than letting git abort
// the whole add (the pre-fix behaviour was a silent no-op exit 0).
func TestSyncSkipsMissingDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	v := vault(t)
	mustGit(t, v, "init", "-q")
	mustGit(t, v, "config", "user.name", "test")
	mustGit(t, v, "config", "user.email", "test@local")
	// Only create 2 of the 4 dirs; "infrastructure" and "reference" are absent.
	for _, d := range []string{"decisions", "projects"} {
		writeNote(t, v, d+"/.keep", "")
	}
	writeNote(t, v, "projects/p.md", "---\nupdated: 2026-06-14\n---\n\n# P\n")

	out, err := runRootCapture(t, "sync", "-m", "partial dirs sync")
	if err != nil {
		t.Fatalf("sync error: %v (out=%q)", err, out)
	}
	log := gitOut(t, v, "log", "--oneline")
	if !strings.Contains(log, "partial dirs sync") {
		t.Errorf("expected commit despite missing dirs, git log:\n%s", log)
	}
}

// ----------------------------------------------------------------------------
// rename — git-dependent output assertions
// ----------------------------------------------------------------------------

func TestRenameOutput(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	v := vault(t)
	mustGit(t, v, "init", "-q")
	mustGit(t, v, "config", "user.name", "test")
	mustGit(t, v, "config", "user.email", "test@local")
	writeNote(t, v, "old.md", "---\nupdated: 2026-06-14\n---\n\n# Old\n")
	writeNote(t, v, "linker.md", "see [[old]] here.\n")
	mustGit(t, v, "add", "-A")

	out, err := runRootCapture(t, "rename", "old.md", "new.md")
	if err != nil {
		t.Fatalf("rename error: %v (out=%q)", err, out)
	}
	for _, want := range []string{"git mv", "Scanned", "Rewrote", "Renamed"} {
		if !strings.Contains(out, want) {
			t.Errorf("rename output missing %q:\n%s", want, out)
		}
	}
}

// ----------------------------------------------------------------------------
// init — git-dependent output assertions
// ----------------------------------------------------------------------------

func TestInitOutput(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	v := t.TempDir()
	out, err := runRootCapture(t, "init", v)
	if err != nil {
		t.Fatalf("init error: %v (out=%q)", err, out)
	}
	for _, want := range []string{"Scaffolded", "hooksPath", "Initial commit"} {
		if !strings.Contains(out, want) {
			t.Errorf("init output missing %q:\n%s", want, out)
		}
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, out)
	}
	return string(out)
}
