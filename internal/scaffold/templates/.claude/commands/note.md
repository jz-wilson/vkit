---
description: Create a new vault note from the schema
---
Create a new note at the path: $ARGUMENTS

Native mode is ON iff `.obsidian-cli-enabled` exists in the vault root. Do NOT
probe `obsidian` to detect it (that can launch the GUI).

**Tier A — native mode on (`.obsidian-cli-enabled` present, desktop app running):**
1. `obsidian create path="$ARGUMENTS" content="# <Title>\n\n## Summary\n\n## Notes\n\n## Related"`
   (derive <Title> from the filename: kebab → Title Case.)
2. `obsidian property:set name="updated" value="$(date +%F)" path="$ARGUMENTS"`
3. Add `tags` if any apply: `obsidian property:set name="tags" type="list" value="..." path="$ARGUMENTS"`
4. Tell me the path and the one-line summary.

**Tier B — native mode off (default / headless):**
Run `vkit note <path> [--title "<Title>"] [--tags a,b]`. It refuses to overwrite
an existing file, writes valid frontmatter (`updated:` = today, plus `tags:` if
given), scaffolds the `# Title` / `## Summary` / `## Notes` / `## Related` body,
and rebuilds MOC.md. Then tell me the path and the one-line summary.
