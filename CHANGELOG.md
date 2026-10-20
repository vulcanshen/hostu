# Changelog

Each release names the version of terminu design (tdp) it follows.

## Unreleased

- Profiles of the hosts file: `[1] PROFILES` lists them, the active one ticked in green; `[2] DOMAINS` shows the domains
  of the one under the cursor, each going to one address or more.
- `Enter` on a profile activates it: after a question that says how much it writes, and warns when it has no
  `localhost`, hostu writes the whole hosts file from it. When the file needs the administrator, `sudo -n` is tried
  first, then sudo asks for the password in a terminal inside hostu; on Windows, gsudo or sudo asks in a UAC window.
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
  `Enter` renames it to `.bak` and starts with a new one, `Esc` quits and prints what is wrong.
- `!` opens the App Log: every message hostu showed, what you did and what hostu did by itself, while it has been
  open.
