#!/usr/bin/env bash
set -euo pipefail

platform="${REAL_ACCEPTANCE_PLATFORM:-linux/amd64}"
arch="${platform#linux/}"
out_dir="${REAL_ACCEPTANCE_BUILD_DIR:-.tmp/real-acceptance-images}"

backend_image="${BACKEND_IMAGE:-java-profiler-backend:qa-amd64}"
collector_image="${COLLECTOR_IMAGE:-java-profiler-collector:qa-amd64}"
web_image="${WEB_IMAGE:-java-profiler-web:qa-amd64}"
async_profiler_version="${ASYNC_PROFILER_VERSION:-4.2.1}"
async_profiler_sha256="${ASYNC_PROFILER_SHA256:-}"
collector_base_image="${COLLECTOR_BASE_IMAGE:-docker.m.daocloud.io/library/golang:1.21-alpine}"

case "$arch" in
  amd64|arm64) ;;
  *) echo "Unsupported arch in REAL_ACCEPTANCE_PLATFORM=$platform" >&2; exit 2 ;;
esac

mkdir -p "$out_dir/bin" "$out_dir/docker"

echo "Building linux/${arch} backend and collector binaries"
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -o "$out_dir/bin/java-profiler-backend" ./cmd/backend
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -o "$out_dir/bin/java-profiler-collector" ./cmd/collector

ap_arch="$arch"
if [[ "$arch" == "amd64" ]]; then
  ap_arch="x64"
  async_profiler_sha256="${async_profiler_sha256:-e4d764f27d06a1d339d13df4f2e1599558b69fcfb01d4c811d13b8c895d7ea63}"
elif [[ "$arch" == "arm64" ]]; then
  async_profiler_sha256="${async_profiler_sha256:-b7f58eead5973d5b04a920380f278e75cf190b49435974c3569869d298639664}"
fi
ap_archive="$out_dir/async-profiler-${async_profiler_version}-linux-${ap_arch}.tar.gz"
ap_extract="$out_dir/async-profiler-${async_profiler_version}-linux-${ap_arch}"
ap_dir="$out_dir/docker/async-profiler"
if [[ ! -s "$ap_archive" ]]; then
  echo "Downloading async-profiler ${async_profiler_version} linux/${ap_arch}"
  curl --http1.1 --retry 3 --retry-all-errors --retry-delay 2 -fsSL -o "$ap_archive" "https://github.com/async-profiler/async-profiler/releases/download/v${async_profiler_version}/async-profiler-${async_profiler_version}-linux-${ap_arch}.tar.gz"
fi
if [[ -n "$async_profiler_sha256" ]]; then
  echo "${async_profiler_sha256}  ${ap_archive}" | sha256sum -c -
fi
rm -rf "$ap_extract" "$ap_dir"
tar -xzf "$ap_archive" -C "$out_dir"
mkdir -p "$ap_dir"
cp "$ap_extract/bin/asprof" "$ap_dir/asprof"
cp "$ap_extract/lib/libasyncProfiler.so" "$ap_dir/libasyncProfiler.so"

cat >"$out_dir/docker/Dockerfile.backend" <<EOF
FROM scratch
COPY bin/java-profiler-backend /java-profiler-backend
ENTRYPOINT ["/java-profiler-backend"]
EOF

cat >"$out_dir/docker/Dockerfile.collector" <<EOF
FROM ${collector_base_image}
COPY bin/java-profiler-collector /java-profiler-collector
COPY async-profiler/asprof /var/lib/java-profiler/assets/asprof
COPY async-profiler/libasyncProfiler.so /var/lib/java-profiler/assets/libasyncProfiler.so
ENTRYPOINT ["/java-profiler-collector"]
EOF

echo "Building web dist and linux/${arch} static web proxy"
(
  cd web
  npm run build
)

cat >"$out_dir/web-server.go" <<'EOF'
package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	backend := strings.TrimRight(os.Getenv("JAVA_PROFILER_BACKEND_URL"), "/")
	if backend == "" {
		backend = "http://java-profiler-backend:8080"
	}
	token := os.Getenv("JAVA_PROFILER_UI_TOKEN")
	root := "/dist"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			proxyAPI(w, r, backend, token)
			return
		}
		serveStatic(w, r, root)
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}

func proxyAPI(w http.ResponseWriter, r *http.Request, backend, token string) {
	target, err := url.Parse(backend + r.URL.RequestURI())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req.Header = r.Header.Clone()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func serveStatic(w http.ResponseWriter, r *http.Request, root string) {
	name := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
	if name == "." || name == "" {
		name = "index.html"
	}
	path := filepath.Join(root, name)
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(root, "index.html")
	}
	http.ServeFile(w, r, path)
}
EOF

CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -o "$out_dir/bin/java-profiler-web" "$out_dir/web-server.go"

cat >"$out_dir/docker/Dockerfile.web" <<EOF
FROM scratch
COPY bin/java-profiler-web /java-profiler-web
COPY web-dist /dist
ENTRYPOINT ["/java-profiler-web"]
EOF

rm -rf "$out_dir/docker/web-dist"
cp -R web/dist "$out_dir/docker/web-dist"

cp "$out_dir/bin/java-profiler-backend" "$out_dir/docker/java-profiler-backend" 2>/dev/null || true
cp "$out_dir/bin/java-profiler-collector" "$out_dir/docker/java-profiler-collector" 2>/dev/null || true
cp "$out_dir/bin/java-profiler-web" "$out_dir/docker/java-profiler-web" 2>/dev/null || true
rm -rf "$out_dir/docker/bin"
cp -R "$out_dir/bin" "$out_dir/docker/bin"

docker buildx build --platform "$platform" --load -t "$backend_image" -f "$out_dir/docker/Dockerfile.backend" "$out_dir/docker"
docker buildx build --platform "$platform" --load -t "$collector_image" -f "$out_dir/docker/Dockerfile.collector" "$out_dir/docker"
docker buildx build --platform "$platform" --load -t "$web_image" -f "$out_dir/docker/Dockerfile.web" "$out_dir/docker"

docker image inspect "$backend_image" "$collector_image" "$web_image" --format '{{.RepoTags}} {{.Os}}/{{.Architecture}}'
