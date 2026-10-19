#!/bin/sh
# Runs hostu (the program given, ./hostu by default) where it can't touch the
# real hosts file or the real home: every start a new temporary directory,
# with a copy of the system's hosts file, a home, and the settings and the
# data inside it. Each start begins as a first start.
#
#   tools/sandbox.sh [./hostu] [args...]
set -e
app=${1:-./hostu}
[ $# -gt 0 ] && shift
tmp=${TMPDIR:-/tmp}
dir=$(mktemp -d "${tmp%/}/hostu-sandbox.XXXXXX")
mkdir -p "$dir/home"
cp /etc/hosts "$dir/hosts" 2>/dev/null || printf '127.0.0.1 localhost\n::1 localhost\n' > "$dir/hosts"
unset XDG_CONFIG_HOME
export HOME="$dir/home" HOSTU__HOSTS_FILE="$dir/hosts" HOSTU__CONFIG="$dir/home/.config/hostu" HOSTU__DATA="$dir/home/.hostu"
exec "$app" "$@"
