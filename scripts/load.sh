#!/usr/bin/env sh
set -eu
url="${1:-http://localhost:80/projeto-korp}"
count="${2:-300}"
i=0
while [ "$i" -lt "$count" ]; do
  curl --fail --silent --show-error --max-time 5 "$url" >/dev/null
  i=$((i + 1))
  sleep 0.2
done
printf '%s requisições concluídas.\n' "$count"
