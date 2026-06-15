# vkit — vault starter kit, one binary

`vkit` is a single cross-platform Go binary that scaffolds and maintains a
plain-folder knowledge vault Claude Code reads live off disk. It replaces the
bash starter kit (`setup.sh`, `build-moc.sh`, `watch.sh`, `lib/detect-os.sh`)
wholesale: the templates ship embedded in the executable, the OS is detected at
runtime, and there are no `find`/`grep`/`fswatch`/`inotifywait` dependencies.

## Install

**Recommended — `go install`** (requires Go 1.22+):

```bash
go install github.com/jz-wilson/vkit@latest
```

**GitHub Releases** — download a pre-built binary from the [releases page](https://github.com/jz-wilson/vkit/releases) and place it on `PATH`.

**From source:**

```bash
git clone https://github.com/jz-wilson/vkit
cd vkit
task install          # build + install to ~/.local/bin (requires https://taskfile.dev)
# or without task:
go build -o /tmp/vkit . && install -m755 /tmp/vkit ~/.local/bin/vkit
```

**Shell completion (zsh):**

```bash
task completion-zsh   # installs ~/.zfunc/_vkit
# or manually: vkit completion zsh > ~/.zfunc/_vkit
# then add to ~/.zshrc: fpath=(~/.zfunc $fpath) && autoload -Uz compinit && compinit
```

The pre-commit hook and watcher services call `vkit` by name — it must be on `PATH`.

## Commands

| Command | Does |
|---------|------|
| `vkit init [path]` | Scaffold a new vault (install mode): write the tree, `git init`, wire the hook, build the index, initial commit. |
| `vkit update [path] [--force\|--keep\|--dry-run]` | Eval-first update from the embedded kit. |
| `vkit moc` | Regenerate `MOC.md` (the Map of Content index). |
| `vkit watch [--poll] [--interval N]` | Rebuild `MOC.md` on every change (fsnotify, polling fallback). |
| `vkit validate [--staged] [files...]` | Validate note frontmatter. `--staged` is the pre-commit hook mode. |
| `vkit note <path> [--title T] [--tags a,b]` | Scaffold a note from the schema (refuses to overwrite). |
| `vkit rename <old> <new>` | Link-safe rename/move: `git mv` first (clean abort on failure), then rewrite inbound `[[links]]`. |
| `vkit sync [-m msg]` | Rebuild the index and commit docs only (never `git add -A`). |
| `vkit doctor` | Print detected OS / pkgmgr / systemd / tty / Obsidian state. |
| `vkit version` | Print the build version (`--version` also works). |

`--vault` (persistent flag) overrides vault discovery, which otherwise checks, in
order: a positional path arg, `$VKIT_VAULT`, a walk-up search for the `_format.md`
marker, then `$HOME/vault`.

**Known limitation — `vkit rename`:** the `git mv` succeeds before link rewrites
begin. If a link rewrite fails mid-scan (e.g. a read-only file), the note is already
moved and some links may be only partially rewritten. Re-run `rename` from the new
path to retry.

## Update model

`vkit update` is **eval-first**: it analyses the vault, prints exactly what
*would* change, and writes nothing until you approve. Files split into two tiers:

- **Tooling** (watcher services, `.gitignore`) — pure machinery, refreshed freely.
- **Templates** (`_format.md`, vault `CLAUDE.md`, the pre-commit hook,
  `.claude/settings.json` + commands) — files you may have customized.

With no flag and a usable terminal it prompts
`[a]ll / [s]afe / [c]ustomize / [d]iff / [q]uit` (default quit). Flags pre-answer
it: `--force` = all (overwrites changed templates, each dropping a `<file>.bak`
first), `--keep` = safe (refresh tooling + add new templates, keep your changes).
`--dry-run` shows the plan and exits. With no flag and no usable terminal, update
**quits without changing anything**. Update **never auto-commits** — it rebuilds
`MOC.md` and leaves staging/committing to you.

## Per-OS notes

- **macOS** — `services/com.vault.watch.plist` (launchd). The installer must
  inject your real `$PATH` so launchd can find `vkit`.
- **Linux** — `services/vault-watch.service` (systemd user unit), when a systemd
  user instance is available (`vkit doctor` reports `systemd-user`).
- **WSL / Windows (Git Bash) / no service manager** — no daemon; run
  `vkit watch` in the background or `vkit sync` on demand.

The watcher uses [fsnotify](https://github.com/fsnotify/fsnotify) (works on
macOS, Linux, Windows) and falls back to a zero-dependency mtime poll
(`--poll`, `--interval` seconds, default 5) when fsnotify is unavailable.

## Obsidian native mode

`vkit note` auto-detects the official [`obsidian` CLI](https://obsidian.md/cli)
via `PATH`. When found, it routes through Tier A (native); otherwise it uses the
portable schema scaffold (Tier B).

To disable native mode explicitly:

```bash
export VAULT_OBSIDIAN_CLI=0
```

`vkit doctor` reports the current detection state (binary found / CLI active / disabled).

## Layout

```
vkit/
├── main.go               -> cmd.Execute()
├── cmd/                   cobra layer (thin)
└── internal/
    ├── vaultpath/         root discovery + shared exclusion rules
    ├── osdetect/          GOOS + /proc/version WSL parse; pkgmgr; systemd; obsidian
    ├── moc/               MOC builder
    ├── watcher/           fsnotify + poll fallback
    ├── scaffold/          install + eval-first update; embedded template tree
    ├── validate/          frontmatter rules; staged-file mode
    ├── note/              portable scaffold (Tier B) + native obsidian (Tier A)
    └── rename/            link-safe rename
```

## Develop

Common tasks via [Taskfile](https://taskfile.dev):

```bash
task build          # go build -o /tmp/vkit .
task install        # build + install to ~/.local/bin
task test           # go test ./...
task lint           # go vet ./... && gofmt -l .
task release-check  # goreleaser check + snapshot
task doctor         # install then vkit doctor --vault ~/vault
```

Enable the pre-commit hook once per clone (runs `gofmt`/`go vet`/`go test`):

```bash
git config core.hooksPath .githooks
```

## CI / Release

GitHub Actions (`.github/workflows/`):

- **ci.yml** — on push to `main` and PRs: `go vet`, `go test -race -cover`, and
  `go build` across a linux/macOS/windows matrix.
- **release.yml** — on a `v*` tag: [goreleaser](https://goreleaser.com)
  (`.goreleaser.yaml`, v2) builds multi-arch tarballs (linux/darwin/windows ×
  amd64/arm64) with checksums and a changelog, and publishes a GitHub release.

The build version is injected at release time via
`-ldflags "-X main.version={{.Version}}"` and surfaced by `vkit version`.

Before cutting the first `v*` tag, sanity-check locally:

```bash
goreleaser check
goreleaser release --snapshot --clean   # dry run, no publish
```
