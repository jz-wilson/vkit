---
description: Rebuild the index and commit the vault
---
1. Run `vkit sync -m "vault: <short summary derived from the actual diff>"`.
   It rebuilds MOC.md, then stages ONLY documentation assets (`*.md`, `MOC.md`,
   and the note dirs decisions/ infrastructure/ projects/ reference/) — never
   `git add -A` — and commits.
2. Before that, show me `git status --short` and call out any unrelated
   untracked files (logs, build artifacts); leave them unstaged.
3. Do NOT push unless I explicitly say so.
