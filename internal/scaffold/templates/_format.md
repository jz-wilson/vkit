# Vault Note Format — Single Source of Truth

Every note in this vault follows this schema. Read before writing any note.

## Frontmatter

Raw YAML block, the absolute first thing in the file, never inside a code fence:

```yaml
---
updated: YYYY-MM-DD      # REQUIRED. Last meaningful edit; bump every change.
tags: [topic, ...]       # RECOMMENDED. Lowercase. Cross-cut folders — this is
                         #   how you grep one axis (e.g. k3s) across the vault.
created: YYYY-MM-DD      # optional
type: note               # optional — usually redundant with the folder
status: active           # optional — archive/ folder already implies archived
---
```

Field discipline:
- `updated:` is the only required field — it is NOT derivable from anything else.
- Keep `tags:` — they are the one retrieval axis the directory layout can't give
  you. A note's folder is a single category; tags let one note surface under
  many. Dropping them to save tokens is a false economy: frontmatter is only read
  when a note is *opened*, never on sweeps (the MOC is the sweep surface).
- Drop `type:`/`status:` unless you actually query them — the path and the
  `archive/` folder already encode that. Every kept field is tokens on each read.

## Body

- One H1 = the title = the note's identity. Exactly one per file.
- Standard Markdown: headings, tables, bullets, bold. No raw HTML.
- Sections in order: `# Title`, `## Summary`, `## Notes`, `## Related`.
- `## Related` lists `[[wikilinks]]` with a short context phrase after `—`.
- Wikilinks for all internal references. Never `file://` or absolute paths —
  they break across machines and are invisible to the link graph.
- End the file with a single newline. No trailing whitespace.

## Example

```markdown
---
updated: 2026-06-12
tags: [k3s, networking]
---

# K3s Cluster Networking

## Summary
One-line statement of what this note captures.

## Notes
Dense facts. Tables for structured data.

## Related
- [[infrastructure/nodes]] — node specs and IPs
```
