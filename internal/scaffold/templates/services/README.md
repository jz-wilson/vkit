# Watcher services

These run `vkit watch`, which rebuilds MOC.md on every note change (fsnotify
backend, with a zero-dependency polling fallback). `vkit doctor` shows what your
OS supports. Install the one for your OS:

## macOS (launchd)
```bash
# Substitute BOTH the vault path and your real $PATH (launchd has no shell
# profile, so it can't see Homebrew/asdf/mise shims — or vkit — otherwise).
sed -e "s#__VAULT__#$HOME/vault#g" -e "s#__PATH__#$PATH#g" \
  services/com.vault.watch.plist > ~/Library/LaunchAgents/com.vault.watch.plist
launchctl load ~/Library/LaunchAgents/com.vault.watch.plist
# stop:  launchctl unload ~/Library/LaunchAgents/com.vault.watch.plist
```

## Linux (systemd user)
```bash
mkdir -p ~/.config/systemd/user
sed "s#__VAULT__#$HOME/vault#g" services/vault-watch.service \
  > ~/.config/systemd/user/vault-watch.service
systemctl --user daemon-reload
systemctl --user enable --now vault-watch.service
# logs:  journalctl --user -u vault-watch -f
```

## WSL / Git Bash (Windows) / no service manager
No daemon — run the watcher in the background, or just use `vkit sync`:
```bash
nohup vkit watch --vault "$HOME/vault" >/dev/null 2>&1 &   # background
# or skip the watcher entirely and run `vkit sync` / `vkit moc` on demand
```
Windows Task Scheduler can also launch `vkit watch` at logon if you want it
persistent.
