#!/usr/bin/env sh
set -eu

values_file="${1:-deploy/helm/values.yaml}"

for key in asyncProfiler threadHelper; do
  digest="$(awk "/${key}:/ {found=1} found && /sha256:/ {print \$2; exit}" "$values_file" | tr -d '"')"
  if ! printf '%s' "$digest" | grep -Eq '^[0-9a-f]{64}$'; then
    echo "invalid ${key} sha256 in ${values_file}: ${digest}" >&2
    exit 1
  fi
done
