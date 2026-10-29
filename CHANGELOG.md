# Changelog

Each release names the version of terminu design (tdp) it follows.

## Unreleased

- A new family mark in `docs/icon.svg` and the splash (`V`): a gold H fills the frame, an O with a T in it above its
  bar and an S below, and the family's U frame is hostu's u.

## v1.0.1 — tdp v1.1.0

Everything since v1.0.0, the betas v1.0.1-beta.1 and beta.2 included:

- On terminu-go v1.1.0 (tdp v1.1.0); hostu looks and works as in v1.0.0.
- A release may have betas first: a tag like `v1.0.1-beta.1` is a GitHub pre-release; Homebrew, Scoop and
  `install.sh` keep the stable release, and `go install github.com/vulcanshen/hostu/cmd/hostu@<the beta's tag>`
  installs the beta.
- The binaries' build information names the tag alone, no longer `+dirty`.
- The release build sets the version through go.mod's module path, as every app of the family now does; `hostu
  version` prints the tag as before.
- The repository no longer keeps Claude Code's files: `.gitignore` leaves out `CLAUDE.md` and `.claude/`, all but
  harness's `.claude/rules/terminu.md`.
- The README's terminu family section links terminu design, terminu-go and terminu-harness, and no longer lists the
  other apps.

## v1.0.1-beta.2 — tdp v1.1.0-beta.3

- On terminu-go v1.1.0-beta.3 (tdp v1.1.0-beta.3); hostu looks and works as in v1.0.0.
- The release build sets the version through go.mod's module path, as every app of the family now does; `hostu
  version` prints the tag as before.

## v1.0.1-beta.1 — tdp v1.1.0-beta.2

- On terminu-go v1.1.0-beta.2 (tdp v1.1.0-beta.2); hostu looks and works as in v1.0.0.
- A beta is a GitHub pre-release: Homebrew, Scoop and `install.sh` keep the stable release, and
  `go install github.com/vulcanshen/hostu/cmd/hostu@v1.0.1-beta.1` installs the beta.
- The binaries' build information names the tag alone, no longer `+dirty`.
- The repository no longer keeps Claude Code's files: `.gitignore` leaves out `CLAUDE.md` and `.claude/`, all but
  harness's `.claude/rules/terminu.md`.
- The README's terminu family section links terminu design, terminu-go and terminu-harness, and no longer lists the
  other apps.

## v1.0.0 — tdp v1.0.0

- Profiles of the hosts file: `[1] PROFILES` lists them, the active one ticked in green; `[2] DOMAINS` shows the domains
  of the one under the cursor, each going to one address or more.
- `Enter` on a profile activates it: after a question that says how much it writes, and warns when it has no
  `localhost`, hostu writes the whole hosts file from it. When the file needs the administrator, `sudo -n` is tried
  first, then hostu asks for sudo's password in a `PASSWORD` popup and hands it to sudo alone (doas asks in a terminal
  inside hostu); on Windows, gsudo or sudo asks in a UAC window.
- The hosts file as it was is kept in `~/.hostu/hosts.original` before hostu first writes it.
- The first start reads the hosts file into a profile named `default`; `i` reads it into a new profile at any time.
- Profiles: new, rename, copy, delete (not the active one), and a preview of the hosts file each writes.
- Domains: the NEW DOMAIN and EDIT forms, a domain and its addresses (IPv4 and IPv6, several between spaces), each
  checked as it is typed; delete, after asking.
- The statusbar shows the hosts file and the active profile, and says when the hosts file differs from it.
- Profiles are kept in `profiles.yaml` in the data directory (`~/.hostu`, `HOSTU__DATA`); `HOSTU__HOSTS_FILE` names
  another hosts file to write; `config.yaml` takes `input_tab_width`.
- A `profiles.yaml` changed by hand is checked as it is read: a name with a control character, two profiles of one
  name, a domain or an address a hosts file can't take. When it, or `config.yaml`, can't be read, hostu asks first:
  `Enter` renames it to `.bak` and starts as the first time, `Esc` quits and prints what is wrong.
- `!` opens the App Log: every message hostu showed, what you did and what hostu did by itself, while it has been
  open.
- Works in a terminal 60 columns wide and 24 rows high, the two panels side by side; in a smaller one hostu shows its
  name, the size it needs and how to quit (`Ctrl-C` twice).
