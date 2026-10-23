# Changelog

## Unreleased

- `dehv` alone shows one line per command, in a card never wider than the terminal; the options of each are in `dehv help <command>`.
- No card is wider than the terminal: a command's help wraps its descriptions, and the rest cut their lines to fit.
- The colours are One Dark's, in 24-bit colour where the terminal has it.

## 0.1.0

- dehv in a repo of its own, out of my dotfiles: a plugin loaded through the link `~/.local/share/dehv`, from a clone that can live anywhere.
- `install.sh`: install, update, uninstall, relink and status, from a menu or as actions; checks what dehv needs first and asks before installing any of it.
- `dehv where`: the link, the clone, the rc file that loads it and the version.
- `dehv doctor` now checks dehv's own install: the link, the block, the completion, what it needs and its files.
- The `dehv hi` greeting and the skills folder are settings in `~/.config/dehv/config`.
