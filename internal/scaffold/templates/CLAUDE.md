# Vault — Project Memory

Loads only when the working directory is inside this vault.

## Rules
- All notes follow @_format.md
- Before external research, grep this vault first.
- Dense facts only. No filler, no restating what's obvious from context.
- On edit: bump `updated:` frontmatter to today.
- Check for an existing note before creating a new one — update, don't duplicate.
- Link related notes with `[[wikilinks]]`, never absolute filesystem paths.

## Lookup & edits — two tiers
This vault works headless (find/grep) AND with the Obsidian app (official
`obsidian` CLI, v1.12+). Tier is chosen by an explicit opt-in, NOT by probing:
native mode is ON iff the file `.obsidian-cli-enabled` exists in the vault root
(or `VAULT_OBSIDIAN_CLI=1`). Do not run `obsidian --version`/`command -v` to
detect it — that can launch the desktop GUI and false-positives on aliases.

**If `.obsidian-cli-enabled` is present (native mode), prefer the CLI:**
- Search:        `obsidian search query="keyword"`
- Search+context: `obsidian search:context query="symbolOrPhrase"`  (tighter, app-ranked — fewer tokens than dumping a file)
- Read a note:   `obsidian read file="Note Name"`
- Set a field:   `obsidian property:set name="updated" value="YYYY-MM-DD" path="<path>"`
- **Rename/move: ALWAYS use `vkit rename`** (it refactors every `[[wikilink]]`
  pointing at the note) or `obsidian rename`/`obsidian move`. NEVER a bare `mv`
  or `git mv` of a note — raw moves silently break links.

**If `.obsidian-cli-enabled` is absent (default / headless / no GUI), use:**
- Search: `rg -i --type md "keyword"` — falls back to `grep -rin --include='*.md'`
- List:   `fd -e md .` — falls back to `find . -name '*.md'`
- Read:   read the file directly
- Edit:   edit the file, hand-bump `updated:`
- Rename: `vkit rename <old> <new>` — link-safe (scans inbound `[[links]]`,
  `git mv`, rewrites them). Never a bare `mv`.

Either tier: the pre-commit hook (`vkit validate --staged`) validates
frontmatter, and MOC.md is the committed, grep-able index Claude reads — both run
regardless of the CLI.

## Index
@MOC.md

## Layout
| Dir | Purpose |
|-----|---------|
| `decisions/` | Monthly decision logs (`YYYY-MM.md`) |
| `infrastructure/` | Homelab / infra notes |
| `projects/` | Per-project notes |
| `reference/` | Stable reference material, glossaries |
| `archive/` | Stale notes (>90 days). Excluded from MOC + searches. |

## Note Lifecycle
1. Create from schema (`vkit note <path>`), set `updated:` to today.
2. Edit in place; bump `updated:` each meaningful change.
3. After 90 days idle, move to `archive/` (`vkit rename`).

## Commit Protocol
- Commit on save or on a schedule: `vkit sync -m "vault: <summary>"`.
- Pre-commit hook validates frontmatter (`vkit validate --staged`).
- Wire it once per clone: `git config core.hooksPath .githooks`.
