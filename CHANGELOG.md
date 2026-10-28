# Changelog

## Unreleased

- `dehv tap`: the commands to release an app (a cask) or a command line tool (a formula) to my homebrew-tap, with the placeholders filled in from the arguments or from `build/*.app` and `./VERSION`; piped, as a script. Its GitHub owner is `tap.owner` in the config.
- The `dehv` card has a header with the version, branch and commit, and a footer with whether `dehv doctor` passes and whether the greeting is on.
- `__dehv.box` draws without a title, rules across the frame (`$__dehv_rule`) and lines with a part pushed to the right edge (after a tab), dropped first when the terminal is narrow.
- `dehv` alone shows one line per command, in a card never wider than the terminal; the options of each are in `dehv help <command>`.
- No card is wider than the terminal: a command's help wraps its descriptions, and the rest cut their lines to fit.
- The colours are One Dark's, in 24-bit colour where the terminal has it.
- `install.sh` always loads dehv from `~/.zshrc`, creating it if there is none, and through its link when a dotfiles repo keeps it; it no longer offers `~/.zshrc.local`, and moves a block found there.

## 0.1.0

- dehv in a repo of its own, out of my dotfiles: a plugin loaded through the link `~/.local/share/dehv`, from a clone that can live anywhere.
- `install.sh`: install, update, uninstall, relink and status, from a menu or as actions; checks what dehv needs first and asks before installing any of it.
- `dehv where`: the link, the clone, the rc file that loads it and the version.
- `dehv doctor` now checks dehv's own install: the link, the block, the completion, what it needs and its files.
- The `dehv hi` greeting and the skills folder are settings in `~/.config/dehv/config`.
