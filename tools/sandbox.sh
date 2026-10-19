#!/bin/sh
# Runs hostu (the program given, ./hostu by default) where it can't touch the
# real hosts file or the real home: every start a new temporary directory,
# with a hosts file as a system has it, a home, and the settings and the data
# inside it. Each start begins as a first start. The hosts file is a sample,
# not a copy of /etc/hosts: what a screen shows may end up in a judge's
# evidence or a recording, and /etc/hosts may name a company's machines.
#
#   tools/sandbox.sh [./hostu] [args...]
set -e
app=${1:-./hostu}
[ $# -gt 0 ] && shift
tmp=${TMPDIR:-/tmp}
dir=$(mktemp -d "${tmp%/}/hostu-sandbox.XXXXXX")
mkdir -p "$dir/home"
printf '##\n# Host Database\n##\n127.0.0.1\tlocalhost\n255.255.255.255\tbroadcasthost\n::1\tlocalhost\n10.0.0.20\tnas.home.lan\n' > "$dir/hosts"
unset XDG_CONFIG_HOME
export HOME="$dir/home" HOSTU__HOSTS_FILE="$dir/hosts" HOSTU__CONFIG="$dir/home/.config/hostu" HOSTU__DATA="$dir/home/.hostu"
exec "$app" "$@"
