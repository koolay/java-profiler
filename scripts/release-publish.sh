#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/release-publish.sh --tag vX.Y.Z --commit SHA \
    --backend-image IMAGE --collector-image IMAGE --web-image IMAGE \
    [--backend-digest DIGEST] [--collector-digest DIGEST] [--web-digest DIGEST] \
    [--chart-dir DIR] [--assets-dir DIR]

Package the Helm chart for a release tag and create or update the matching
GitHub Release with the packaged chart and release metadata.
USAGE
}

tag=""
commit=""
backend_image=""
collector_image=""
web_image=""
backend_digest=""
collector_digest=""
web_digest=""
chart_dir="deploy/helm"
assets_dir=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag) tag="$2"; shift 2 ;;
    --commit) commit="$2"; shift 2 ;;
    --backend-image) backend_image="$2"; shift 2 ;;
    --collector-image) collector_image="$2"; shift 2 ;;
    --web-image) web_image="$2"; shift 2 ;;
    --backend-digest) backend_digest="$2"; shift 2 ;;
    --collector-digest) collector_digest="$2"; shift 2 ;;
    --web-digest) web_digest="$2"; shift 2 ;;
    --chart-dir) chart_dir="$2"; shift 2 ;;
    --assets-dir) assets_dir="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "$tag" || -z "$commit" || -z "$backend_image" || -z "$collector_image" || -z "$web_image" ]]; then
  usage >&2
  exit 2
fi

case "$tag" in
  v*) ;;
  *) echo "release tag must start with v: $tag" >&2; exit 2 ;;
esac

if [[ -z "$assets_dir" ]]; then
  assets_dir="$(mktemp -d)"
fi

release_version="${tag#v}"
work_dir="${assets_dir}/chart"
mkdir -p "$work_dir" "$assets_dir"
rm -f "$assets_dir"/*.tgz "$assets_dir"/SHA256SUMS "$assets_dir"/release-notes.md "$assets_dir"/release-manifest.txt

cp -R "$chart_dir"/. "$work_dir"/
rm -f "$work_dir/values_test.yaml"

perl -0pi -e "s/^version: .*/version: ${release_version}/m" "$work_dir/Chart.yaml"
perl -0pi -e "s/^appVersion: .*/appVersion: \"${tag}\"/m" "$work_dir/Chart.yaml"
perl -0pi -e "s#ghcr.io/koolay/java-profiler-backend:0.1.0#${backend_image}#g" "$work_dir/values.yaml"
perl -0pi -e "s#ghcr.io/koolay/java-profiler-collector:0.1.0#${collector_image}#g" "$work_dir/values.yaml"
perl -0pi -e "s#ghcr.io/koolay/java-profiler-web:0.1.0#${web_image}#g" "$work_dir/values.yaml"

helm lint "$work_dir"
helm package "$work_dir" \
  --destination "$assets_dir" \
  --version "$release_version" \
  --app-version "$tag"
sha256sum "$assets_dir"/*.tgz > "$assets_dir/SHA256SUMS"

cat > "$assets_dir/release-notes.md" <<EOF
# java-profiler ${tag}

- source commit: \`${commit}\`
- backend image: \`${backend_image}\`
- collector image: \`${collector_image}\`
- web image: \`${web_image}\`
- backend digest: \`${backend_digest:-unknown}\`
- collector digest: \`${collector_digest:-unknown}\`
- web digest: \`${web_digest:-unknown}\`
- helm chart version: \`${release_version}\`
EOF

cat > "$assets_dir/release-manifest.txt" <<EOF
tag=${tag}
commit=${commit}
backend_image=${backend_image}
collector_image=${collector_image}
web_image=${web_image}
backend_digest=${backend_digest}
collector_digest=${collector_digest}
web_digest=${web_digest}
chart_package=$(basename "$assets_dir"/*.tgz)
EOF

export GH_TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
if [[ -z "${GH_TOKEN:-}" ]]; then
  echo "GH_TOKEN or GITHUB_TOKEN is required to publish a GitHub Release" >&2
  exit 1
fi

if gh release view "$tag" >/dev/null 2>&1; then
  gh release edit "$tag" --title "$tag" --notes-file "$assets_dir/release-notes.md"
  existing_assets="$(gh release view "$tag" --json assets --jq '.assets[].name' || true)"
  upload_args=()
  for asset_path in "$assets_dir"/*.tgz "$assets_dir"/SHA256SUMS "$assets_dir"/release-manifest.txt; do
    asset_name="$(basename "$asset_path")"
    if printf '%s\n' "$existing_assets" | grep -Fxq "$asset_name"; then
      continue
    fi
    upload_args+=("$asset_path")
  done
  if [[ "${#upload_args[@]}" -gt 0 ]]; then
    gh release upload "$tag" "${upload_args[@]}"
  fi
else
  gh release create "$tag" "$assets_dir"/*.tgz "$assets_dir"/SHA256SUMS "$assets_dir"/release-manifest.txt --title "$tag" --notes-file "$assets_dir/release-notes.md"
fi
