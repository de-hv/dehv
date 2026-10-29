# Changelog

## 0.2.0

- dehv is an executable, `bin/dehv`, installed with Homebrew (`brew install de-hv/tap/dehv`) instead of a plugin loaded from the rc file. `install.sh`, `dehv.plugin.zsh`, the link `~/.local/share/dehv` and the block in `~/.zshrc` are gone: brew installs, upgrades and uninstalls it.
- The greeting is gone, and with it `dehv hi -d` and the config key `hi.daily`; `dehv hi` in the rc file shows the card in every new terminal.
- `dehv where` shows the command, the folder with its files, where it comes from (Homebrew, a clone or a folder) and the version.
- `dehv doctor` checks that the `dehv` in the PATH is the one running and that its completion is there, instead of the link and the block.
- git is only asked for the branch and commit when dehv runs from a clone of its own: from Homebrew's Cellar it would have found Homebrew's own repo.
- The completion loads the commands' files itself, from the folder of the `dehv` in the PATH.
- `dehv tap`: the commands to release an app (a cask) or a command line tool (a formula) to my homebrew-tap, with the placeholders filled in from the arguments or from `build/*.app` and `./VERSION`; piped, as a script. Its GitHub owner is `tap.owner` in the config.
- The `dehv` card has a header with the version, branch and commit, and a footer with whether `dehv doctor` passes.
- `__dehv.box` draws without a title, rules across the frame (`$__dehv_rule`) and lines with a part pushed to the right edge (after a tab), dropped first when the terminal is narrow.
- `dehv` alone shows one line per command, in a card never wider than the terminal; the options of each are in `dehv help <command>`.
- No card is wider than the terminal: a command's help wraps its descriptions, and the rest cut their lines to fit.
- The colours are One Dark's, in 24-bit colour where the terminal has it.

## 0.1.0

- dehv in a repo of its own, out of my dotfiles: a plugin loaded through the link `~/.local/share/dehv`, from a clone that can live anywhere.
- `install.sh`: install, update, uninstall, relink and status, from a menu or as actions; checks what dehv needs first and asks before installing any of it.
- `dehv where`: the link, the clone, the rc file that loads it and the version.
- `dehv doctor` now checks dehv's own install: the link, the block, the completion, what it needs and its files.
- The `dehv hi` greeting and the skills folder are settings in `~/.config/dehv/config`.
