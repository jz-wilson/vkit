---
description: Rename or move a note WITHOUT breaking its wikilinks
---
Rename/move a note. Arguments: `<old-path>` `<new-path-or-name>` ($ARGUMENTS).

Native mode is ON iff `.obsidian-cli-enabled` exists in the vault root. Do NOT
probe `obsidian` to detect it (that can launch the GUI).

**Tier A — native mode on (`.obsidian-cli-enabled` present, desktop app running):**
- Rename in place:  `obsidian rename path="<old-path>" name="<new-name>"`
- Move to new path: `obsidian move path="<old-path>" to="<new-path>"`
Both auto-refactor every `[[wikilink]]` pointing at the note (requires
"automatically update internal links" enabled in vault settings — confirm once).
Then run `vkit moc` and report the new path.

**Tier B — native mode off (default / headless):**
Run `vkit rename <old-path> <new-path>`. It scans inbound `[[links]]` first,
does a `git mv` (preserving history), rewrites every inbound `[[old]]` → `[[new]]`,
and rebuilds MOC.md. It prints the list of files it touched — show me that list
so I can verify no link was missed.

Never do a bare `mv` of a note — it orphans every link silently.
