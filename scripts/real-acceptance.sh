#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  KUBECONFIG=$HOME/backup/localk8s.yaml scripts/real-acceptance.sh [options]

Options:
  --install                 Create namespace, auth secret, ClickHouse, Helm release, and Java workload before testing.
  --configure-profiler      Update the Helm release target filters for an existing Java workload without creating it.
  --load-local-images       During --install, docker-save local QA images and import them into the Linux node containerd.
  --skip-workload-rollout-check
                            Do not require deploy/SERVICE rollout status; useful when SERVICE is a label-level filter.
  --namespace NAME          Kubernetes namespace. Default: java-profiler-qa.
  --profiler-namespace NAME Namespace where java-profiler backend/web/collector/ClickHouse run. Default: --namespace value.
  --release NAME            Helm release name. Default: java-profiler.
  --service NAME            Java workload service/name filter. Default: checkout-java.
  --artifact-dir DIR        Evidence directory. Default: /tmp/java-profiler-real-acceptance-<timestamp>.
  --require-full-profiling  Fail if current-run accepted status, profile, thread, or deadlock data is empty.
  --high-volume             Drive a longer allocation/CPU/lock run and verify bounded profile ingestion metadata.
  --skip-browser            Skip Playwright UI screenshots/video.
  -h, --help                Show this help.

  Environment:
  KUBECONFIG                Required. Example: $HOME/backup/localk8s.yaml.
  BACKEND_IMAGE             Default: ghcr.io/koolay/java-profiler-backend:0.1.0.
  COLLECTOR_IMAGE           Default: ghcr.io/koolay/java-profiler-collector:0.1.0.
  WEB_IMAGE                 Default: ghcr.io/koolay/java-profiler-web:0.1.0.
  CLICKHOUSE_IMAGE          Default: docker.m.daocloud.io/clickhouse/clickhouse-server:24.8.
  JAVA_WORKLOAD_IMAGE       Default: docker.m.daocloud.io/eclipse-temurin:21-jdk.
  JAVA_WORKLOAD_PREBUILT    Set to 1 when JAVA_WORKLOAD_IMAGE already starts a CPU-busy Java app.
  JAVA_PROFILER_HIGH_VOLUME_SECONDS
                            Default: 240 when --high-volume is set.
  JAVA_PROFILER_ACCEPTANCE_LOAD_PATHS
                            Optional comma-separated HTTP path suffixes to hammer when the service is not the JDK17 demo.
                            Example: /profile-load/
  UI_TOKEN                  Default: qa-ui-token.
  COLLECTOR_TOKEN           Default: qa-collector-token.
  CLICKHOUSE_USER           Default: default.
  CLICKHOUSE_PASSWORD       Default: qa-clickhouse.
USAGE
}

namespace="java-profiler-qa"
profiler_namespace=""
release="java-profiler"
service_name="checkout-java"
artifact_dir="/tmp/java-profiler-real-acceptance-$(date +%Y%m%d-%H%M%S)"
install="false"
configure_profiler="false"
load_local_images="false"
require_full="false"
high_volume="false"
skip_browser="false"
skip_workload_rollout_check="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install) install="true"; shift ;;
    --configure-profiler) configure_profiler="true"; shift ;;
    --load-local-images) load_local_images="true"; shift ;;
    --skip-workload-rollout-check) skip_workload_rollout_check="true"; shift ;;
    --namespace) namespace="$2"; shift 2 ;;
    --profiler-namespace) profiler_namespace="$2"; shift 2 ;;
    --release) release="$2"; shift 2 ;;
    --service) service_name="$2"; shift 2 ;;
    --artifact-dir) artifact_dir="$2"; shift 2 ;;
    --require-full-profiling) require_full="true"; shift ;;
    --high-volume) high_volume="true"; require_full="true"; shift ;;
    --skip-browser) skip_browser="true"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 2 ;;
  esac
done

: "${KUBECONFIG:?KUBECONFIG is required, for example $HOME/backup/localk8s.yaml}"

export NO_PROXY='*'
export no_proxy='*'

if [[ "$load_local_images" == "true" ]]; then
  backend_image="${BACKEND_IMAGE:-java-profiler-backend:qa-amd64}"
  collector_image="${COLLECTOR_IMAGE:-java-profiler-collector:qa-amd64}"
  web_image="${WEB_IMAGE:-java-profiler-web:qa-amd64}"
else
  backend_image="${BACKEND_IMAGE:-ghcr.io/koolay/java-profiler-backend:0.1.0}"
  collector_image="${COLLECTOR_IMAGE:-ghcr.io/koolay/java-profiler-collector:0.1.0}"
  web_image="${WEB_IMAGE:-ghcr.io/koolay/java-profiler-web:0.1.0}"
fi
clickhouse_image="${CLICKHOUSE_IMAGE:-docker.m.daocloud.io/clickhouse/clickhouse-server:24.8}"
java_workload_image="${JAVA_WORKLOAD_IMAGE:-docker.m.daocloud.io/eclipse-temurin:21-jdk}"
java_workload_prebuilt="${JAVA_WORKLOAD_PREBUILT:-0}"
ui_token="${UI_TOKEN:-qa-ui-token}"
collector_token="${COLLECTOR_TOKEN:-qa-collector-token}"
clickhouse_user="${CLICKHOUSE_USER:-default}"
clickhouse_password="${CLICKHOUSE_PASSWORD:-qa-clickhouse}"
clickhouse_dsn="${CLICKHOUSE_DSN:-tcp://${clickhouse_user}:${clickhouse_password}@clickhouse:9000/java_profiler}"
backend_port="${JAVA_PROFILER_BACKEND_PORT:-18082}"
web_port="${JAVA_PROFILER_WEB_PORT:-18081}"
collector_port="${JAVA_PROFILER_COLLECTOR_PORT:-29090}"
collector_interval="${JAVA_PROFILER_COLLECTOR_INTERVAL:-60s}"
acceptance_started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
if [[ -z "$profiler_namespace" ]]; then
  profiler_namespace="$namespace"
fi

mkdir -p "$artifact_dir"
summary="$artifact_dir/summary.md"
: > "$summary"

log() { printf '%s\n' "$*" | tee -a "$summary"; }
fail() { log "FAIL: $*"; exit 1; }
pass() { log "PASS: $*"; }
gap() {
  log "GAP: $*"
  if [[ "$require_full" == "true" ]]; then
    exit 1
  fi
}
optional_gap() {
  log "GAP: $*"
}

json_root_value() {
  jq --stream -r 'select(.[0] == ["root", "value"]) | .[1]' "$1" | head -n 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

duration_seconds() {
  case "$1" in
    *ms) echo 1 ;;
    *s) echo "${1%s}" ;;
    *m) echo "$(( ${1%m} * 60 ))" ;;
    *h) echo "$(( ${1%h} * 3600 ))" ;;
    *) echo "$1" ;;
  esac
}

drive_workload_load() {
  local local_port="${JAVA_PROFILER_WORKLOAD_PORT:-18182}"
  local duration="${1:-30}"
  local cpu_alloc_parallel="${JAVA_PROFILER_LOAD_PARALLELISM:-1}"
  local lock_parallel="${JAVA_PROFILER_LOCK_PARALLELISM:-3}"
  if [[ "$high_volume" == "true" ]]; then
    cpu_alloc_parallel="${JAVA_PROFILER_LOAD_PARALLELISM:-4}"
    lock_parallel="${JAVA_PROFILER_LOCK_PARALLELISM:-8}"
  fi
  local service_port
  local load_paths_csv="${JAVA_PROFILER_ACCEPTANCE_LOAD_PATHS:-}"
  local -a load_paths=()
  if [[ -n "$load_paths_csv" ]]; then
    IFS=',' read -r -a load_paths <<<"$load_paths_csv"
  fi
  service_port="$(kubectl -n "$namespace" get "svc/$service_name" -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || true)"
  if [[ -z "$service_port" ]]; then
    log "- workload service ${namespace}/${service_name} has no Service; assuming in-pod load is already active for ${duration}s"
    sleep "$duration"
    return 0
  fi
  kubectl -n "$namespace" port-forward --address 127.0.0.1 "svc/$service_name" "${local_port}:${service_port}" >"$artifact_dir/port-forward-workload.log" 2>&1 &
  cleanup_pids+=("$!")
  local readiness_url="http://127.0.0.1:${local_port}/health"
  if [[ "${#load_paths[@]}" -gt 0 ]]; then
    if [[ "${load_paths[0]}" == http://* || "${load_paths[0]}" == https://* ]]; then
      readiness_url="${load_paths[0]}"
    else
      readiness_url="http://127.0.0.1:${local_port}${load_paths[0]}"
    fi
  fi
  if [[ "${#load_paths[@]}" -gt 0 && "$service_name" != "jdk17-http-demo" ]]; then
    if ! wait_http "$readiness_url"; then
      log "- workload service did not respond on configured load path ${readiness_url}; skipping generic load driver"
      return 0
    fi
    log "- workload service is not the JDK17 HTTP demo; driving generic HTTP load for ${duration}s on ${#load_paths[@]} configured path(s)"
    local request_parallel="$cpu_alloc_parallel"
    local path_target
    local path_request_pids=()
    local deadline=$(( $(date +%s) + duration ))
    while [[ "$(date +%s)" -lt "$deadline" ]]; do
      for path_target in "${load_paths[@]}"; do
        path_request_pids=()
        for _ in $(seq 1 "$request_parallel"); do
          if [[ "$path_target" == http://* || "$path_target" == https://* ]]; then
            curl -fsS "$path_target" >>"$artifact_dir/workload-generic-load.log" 2>&1 &
          else
            curl -fsS "http://127.0.0.1:${local_port}${path_target}" >>"$artifact_dir/workload-generic-load.log" 2>&1 &
          fi
          path_request_pids+=("$!")
        done
        for load_pid in "${path_request_pids[@]}"; do
          wait "$load_pid" || true
        done
      done
    done
    return 0
  fi
  if ! wait_http "http://127.0.0.1:${local_port}/health"; then
    log "- workload service is not the JDK17 HTTP demo; skipping endpoint load driver"
    return 0
  fi
  log "- driving JDK17 demo load for ${duration}s during profiling window"
  local deadline=$(( $(date +%s) + duration ))
  local iteration=0
  while [[ "$(date +%s)" -lt "$deadline" ]]; do
    iteration=$((iteration + 1))
    for mode in cpu alloc gc io wall; do
      load_pids=()
      for _ in $(seq 1 "$cpu_alloc_parallel"); do
        curl -fsS "http://127.0.0.1:${local_port}/work?mode=${mode}&durationMs=3000" >>"$artifact_dir/workload-${mode}-load.log" 2>&1 &
        load_pids+=("$!")
      done
      for load_pid in "${load_pids[@]}"; do
        wait "$load_pid" || true
      done
    done
    # Lock profiling needs real contention. A single request can hold and
    # release the monitor without ever blocking another Java thread.
    lock_pids=()
    for _ in $(seq 1 "$lock_parallel"); do
      curl -fsS "http://127.0.0.1:${local_port}/work?mode=lock&durationMs=3000" >>"$artifact_dir/workload-lock-load.log" 2>&1 &
      lock_pids+=("$!")
    done
    for lock_pid in "${lock_pids[@]}"; do
      wait "$lock_pid" || true
    done
  done
}

wait_http() {
  local url="$1"
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

cleanup_pids=()
cleanup() {
  for pid in "${cleanup_pids[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT

require_cmd kubectl
require_cmd helm
require_cmd curl
require_cmd jq

image_pull_hint() {
  cat <<EOF | tee -a "$summary"

Image pull fallback:
  If a Linux node cannot pull an image directly, prefetch it on a machine with registry access and load it into the node runtime.
  Example:
    crane pull <image> /tmp/image.tar
    # then load /tmp/image.tar into the Kubernetes node containerd runtime, for example:
    # sudo ctr -n k8s.io images import /tmp/image.tar
  This script keeps every image configurable through BACKEND_IMAGE, COLLECTOR_IMAGE, WEB_IMAGE, CLICKHOUSE_IMAGE, and JAVA_WORKLOAD_IMAGE.
EOF
}

load_images_into_node() {
  require_cmd docker
  log "## Load Local Images"
  local images=("$backend_image" "$collector_image" "$web_image")
  if [[ "$java_workload_prebuilt" == "1" ]]; then
    images+=("$java_workload_image")
  fi

  cat <<YAML | kubectl -n "$profiler_namespace" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: image-loader
spec:
  restartPolicy: Never
  hostPID: true
  containers:
    - name: loader
      image: ${clickhouse_image}
      command: ["/bin/sh", "-lc", "sleep 3600"]
      securityContext:
        privileged: true
        runAsUser: 0
      volumeMounts:
        - name: host-root
          mountPath: /host
  volumes:
    - name: host-root
      hostPath:
        path: /
        type: Directory
YAML
  kubectl -n "$profiler_namespace" wait --for=condition=Ready pod/image-loader --timeout=120s
  kubectl -n "$profiler_namespace" exec image-loader -- chroot /host /usr/bin/ctr version >"$artifact_dir/image-loader-ctr-version.txt"

  for image in "${images[@]}"; do
    local safe
    safe="$(printf '%s' "$image" | tr '/:' '__')"
    log "- loading $image"
    docker save "$image" -o "$artifact_dir/$safe.tar"
    kubectl -n "$profiler_namespace" cp "$artifact_dir/$safe.tar" "image-loader:/host/tmp/$safe.tar"
    kubectl -n "$profiler_namespace" exec image-loader -- chroot /host /usr/bin/ctr -n k8s.io images import "/tmp/$safe.tar" >"$artifact_dir/image-import-$safe.log"
  done
  kubectl -n "$profiler_namespace" delete pod image-loader --ignore-not-found=true --wait=true
  pass "local images imported into node containerd"
}

log "# Java Profiler Real Acceptance"
log ""
log "- kubeconfig: $KUBECONFIG"
log "- context: $(kubectl config current-context)"
log "- namespace: $namespace"
log "- profiler_namespace: $profiler_namespace"
log "- release: $release"
log "- service: $service_name"
log "- artifact_dir: $artifact_dir"
log "- configure_profiler: $configure_profiler"
log "- skip_workload_rollout_check: $skip_workload_rollout_check"
log "- high_volume: $high_volume"
log "- acceptance_started_at: $acceptance_started_at"
log ""

if [[ "$install" == "true" ]]; then
  log "## Install"
  kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -
  kubectl create namespace "$profiler_namespace" --dry-run=client -o yaml | kubectl apply -f -
  kubectl -n "$profiler_namespace" create secret generic java-profiler-auth \
    --from-literal=collector-token="$collector_token" \
    --from-literal=ui-token="$ui_token" \
    --dry-run=client -o yaml | kubectl apply -f -

  cat <<YAML | kubectl -n "$profiler_namespace" apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clickhouse
  labels:
    app: clickhouse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: clickhouse
  template:
    metadata:
      labels:
        app: clickhouse
    spec:
      containers:
        - name: clickhouse
          image: ${clickhouse_image}
          ports:
            - name: native
              containerPort: 9000
            - name: http
              containerPort: 8123
          env:
            - name: CLICKHOUSE_DB
              value: java_profiler
            - name: CLICKHOUSE_USER
              value: ${clickhouse_user}
            - name: CLICKHOUSE_PASSWORD
              value: ${clickhouse_password}
          resources:
            requests:
              cpu: 100m
              memory: 512Mi
            limits:
              cpu: "1"
              memory: 4Gi
---
apiVersion: v1
kind: Service
metadata:
  name: clickhouse
spec:
  selector:
    app: clickhouse
  ports:
    - name: native
      port: 9000
      targetPort: 9000
    - name: http
      port: 8123
      targetPort: 8123
YAML

  if ! kubectl -n "$profiler_namespace" rollout status deploy/clickhouse --timeout=180s; then
    kubectl -n "$profiler_namespace" get pods -o wide | tee "$artifact_dir/install-pods-after-clickhouse-failure.txt" | tee -a "$summary" >/dev/null || true
    kubectl -n "$profiler_namespace" describe deploy/clickhouse >"$artifact_dir/clickhouse-deployment-describe.txt" 2>&1 || true
    kubectl -n "$profiler_namespace" describe pod -l app=clickhouse >"$artifact_dir/clickhouse-pod-describe.txt" 2>&1 || true
    image_pull_hint
    fail "ClickHouse did not become ready. Check $artifact_dir/clickhouse-pod-describe.txt for ImagePullBackOff or startup errors."
  fi

  if [[ "$load_local_images" == "true" ]]; then
    load_images_into_node
  fi

  helm upgrade --install "$release" ./deploy/helm \
    --namespace "$profiler_namespace" \
    --values deploy/helm/values.yaml \
    --set "clusterName=$(kubectl config current-context)" \
    --set "image.backend=$backend_image" \
    --set "image.collector=$collector_image" \
    --set "image.web=$web_image" \
    --set "clickhouse.dsn=${clickhouse_dsn}" \
    --set "profiling.collectorInterval=${collector_interval}" \
    --set "profiling.enableAllocationAndLockJFR=${require_full}" \
    --set "profiling.targetNamespace=${namespace}" \
    --set "profiling.targetService=${service_name}" \
    --set "auth.existingSecret=java-profiler-auth" \
    --set "auth.collectorTokenKey=collector-token" \
    --set "auth.uiTokenKey=ui-token"

  if [[ "$java_workload_prebuilt" == "1" ]]; then
    cat <<YAML | kubectl -n "$namespace" apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${service_name}
  labels:
    app: ${service_name}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${service_name}
  template:
    metadata:
      labels:
        app: ${service_name}
        java-profiler.io/profile-mode: temporary
      annotations:
        java-profiler.io/acceptance-run: "${acceptance_started_at}"
        java-profiler.io/profile-disabled: "false"
        java-profiler.io/profile-mode: temporary
        java-profiler.io/profile-duration: 1h
        java-profiler.io/startup-delay: 0s
        java-profiler.io/snapshot-interval: 10s
    spec:
      containers:
        - name: app
          image: ${java_workload_image}
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: "1"
              memory: 512Mi
YAML
  else
    cat <<YAML | kubectl -n "$namespace" apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${service_name}
  labels:
    app: ${service_name}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${service_name}
  template:
    metadata:
      labels:
        app: ${service_name}
        java-profiler.io/profile-mode: temporary
      annotations:
        java-profiler.io/acceptance-run: "${acceptance_started_at}"
        java-profiler.io/profile-disabled: "false"
        java-profiler.io/profile-mode: temporary
        java-profiler.io/profile-duration: 1h
        java-profiler.io/startup-delay: 0s
        java-profiler.io/snapshot-interval: 10s
    spec:
      containers:
        - name: app
          image: ${java_workload_image}
          command: ["/bin/sh", "-lc"]
          args:
            - |
              printf '%s\n' \
                'public class BusyApp {' \
                '  private static final Object LOCK = new Object();' \
                '  private static volatile Object sink;' \
                '  public static void main(String[] args) throws Exception {' \
                '    for (int t = 0; t < 4; t++) {' \
                '      Thread worker = new Thread(() -> {' \
                '        java.nio.file.Path ioFile = null;' \
                '        try {' \
                '          ioFile = java.nio.file.Files.createTempFile("busy-", ".bin");' \
                '          long x = 0;' \
                '          while (true) {' \
                '            byte[] payload = new byte[64 * 1024];' \
                '            for (int i = 0; i < payload.length; i += 4096) payload[i] = (byte)i;' \
                '            java.nio.file.Files.write(ioFile, payload);' \
                '            java.nio.file.Files.readAllBytes(ioFile);' \
                '            sink = payload;' \
                '            for (int i = 0; i < 500000; i++) x += i;' \
                '            if ((x & 0x3fff) == 0) System.gc();' \
                '            if (x == Long.MIN_VALUE) System.out.println(x);' \
                '            synchronized (LOCK) {' \
                '              try { Thread.sleep(3); } catch (InterruptedException e) { Thread.currentThread().interrupt(); }' \
                '            }' \
                '          }' \
                '        } catch (Exception e) {' \
                '          throw new RuntimeException(e);' \
                '        } finally {' \
                '          if (ioFile != null) try { java.nio.file.Files.deleteIfExists(ioFile); } catch (Exception ignored) {}' \
                '        }' \
                '      }, "busy-worker-" + t);' \
                '      worker.setDaemon(false);' \
                '      worker.start();' \
                '    }' \
                '    while (true) {' \
                '      Thread.sleep(1000);' \
                '    }' \
                '  }' \
                '}' > /tmp/BusyApp.java
              javac /tmp/BusyApp.java
              java -cp /tmp BusyApp
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: "1"
              memory: 512Mi
YAML
  fi

  if ! kubectl -n "$profiler_namespace" rollout status "deploy/$release-backend" --timeout=180s; then
    kubectl -n "$profiler_namespace" describe "deploy/$release-backend" >"$artifact_dir/backend-deployment-describe.txt" 2>&1 || true
    image_pull_hint
    fail "backend did not become ready"
  fi
  if ! kubectl -n "$profiler_namespace" rollout status "deploy/$release-web" --timeout=180s; then
    kubectl -n "$profiler_namespace" describe "deploy/$release-web" >"$artifact_dir/web-deployment-describe.txt" 2>&1 || true
    image_pull_hint
    fail "web did not become ready"
  fi
  if ! kubectl -n "$namespace" rollout status "deploy/$service_name" --timeout=180s; then
    kubectl -n "$namespace" describe "deploy/$service_name" >"$artifact_dir/workload-deployment-describe.txt" 2>&1 || true
    image_pull_hint
    fail "Java workload did not become ready"
  fi
  if ! kubectl -n "$profiler_namespace" rollout status "daemonset/$release-collector" --timeout=180s; then
    kubectl -n "$profiler_namespace" describe "daemonset/$release-collector" >"$artifact_dir/collector-daemonset-describe.txt" 2>&1 || true
    image_pull_hint
    fail "collector did not become ready"
  fi
  pass "install phase completed"
fi

target_selector=""
if kubectl -n "$namespace" get "deploy/$service_name" >/dev/null 2>&1; then
  target_selector="$(kubectl -n "$namespace" get "deploy/$service_name" -o json | jq -r '.spec.selector.matchLabels // {} | to_entries | map("\(.key)=\(.value)") | join(",")')"
fi
if [[ -z "$target_selector" ]]; then
  target_selector="app=${service_name}"
fi

capture_target_state() {
  local phase="$1"
  local pods_json="$artifact_dir/target-${phase}-pods.json"
  local deploy_txt="$artifact_dir/target-${phase}-deployment.txt"
  kubectl -n "$namespace" get pods -l "$target_selector" -o wide | tee "$artifact_dir/target-${phase}-pods.txt" | tee -a "$summary" >/dev/null || true
  kubectl -n "$namespace" get pods -l "$target_selector" -o json >"$pods_json" 2>/dev/null || printf '{"items":[]}\n' >"$pods_json"
  kubectl -n "$namespace" get "deploy/$service_name" -o yaml >"$deploy_txt" 2>/dev/null || true
  jq '[.items[] | {pod: .metadata.name, phase: .status.phase, node: .spec.nodeName, restarts: ([.status.containerStatuses[]?.restartCount] | add // 0), containers: [.status.containerStatuses[]? | {name, restartCount, ready, image}]}]' "$pods_json" >"$artifact_dir/target-${phase}-restarts.json"
}

compare_target_restarts() {
  local before="$artifact_dir/target-before-restarts.json"
  local after="$artifact_dir/target-after-restarts.json"
  local report="$artifact_dir/target-restart-comparison.json"
  jq -n --slurpfile before "$before" --slurpfile after "$after" '
    ($before[0] // []) as $b |
    ($after[0] // []) as $a |
    [ $a[] as $afterPod |
      ($b[] | select(.pod == $afterPod.pod)) as $beforePod |
      select($afterPod.restarts > $beforePod.restarts) |
      {pod: $afterPod.pod, before: $beforePod.restarts, after: $afterPod.restarts}
    ]' >"$report"
  local increased
  increased="$(jq 'length' "$report")"
  if [[ "$increased" -gt 0 ]]; then
    jq -r '.[] | "- \(.pod): \(.before) -> \(.after)"' "$report" | tee -a "$summary" >/dev/null
    fail "target workload restart count increased during acceptance"
  fi
  pass "target workload restart counts did not increase"
}

log "## Target Baseline"
log "- target_selector: $target_selector"
capture_target_state before

capture_clickhouse_state() {
  local phase="$1"
  kubectl -n "$profiler_namespace" get pods -l app=clickhouse -o json >"$artifact_dir/clickhouse-${phase}-pods.json" 2>/dev/null || printf '{"items":[]}\n' >"$artifact_dir/clickhouse-${phase}-pods.json"
  jq '[.items[] | {pod: .metadata.name, uid: .metadata.uid, phase: .status.phase, restarts: ([.status.containerStatuses[]?.restartCount] | add // 0), oom_killed: ([.status.containerStatuses[]?.lastState.terminated.reason | select(. == "OOMKilled")] | length)}]' "$artifact_dir/clickhouse-${phase}-pods.json" >"$artifact_dir/clickhouse-${phase}-state.json"
}

compare_clickhouse_state() {
  local before="$artifact_dir/clickhouse-before-state.json"
  local after="$artifact_dir/clickhouse-after-state.json"
  local report="$artifact_dir/clickhouse-restart-comparison.json"
  jq -n --slurpfile before "$before" --slurpfile after "$after" '
    ($before[0] // []) as $b |
    ($after[0] // []) as $a |
    {
      replaced_pods: [ $a[] as $afterPod | select([ $b[].uid ] | index($afterPod.uid) | not) | {pod: $afterPod.pod, uid: $afterPod.uid} ],
      restart_increases: [ $a[] as $afterPod | ($b[] | select(.uid == $afterPod.uid)) as $beforePod | select($afterPod.restarts > $beforePod.restarts) | {pod: $afterPod.pod, before: $beforePod.restarts, after: $afterPod.restarts} ],
      oom_killed: [ $a[] | select(.oom_killed > 0) | {pod, oom_killed} ]
    }' >"$report"
  local issue_count
  issue_count="$(jq '(.replaced_pods | length) + (.restart_increases | length) + (.oom_killed | length)' "$report")"
  if [[ "$issue_count" -gt 0 ]]; then
    jq . "$report" | tee -a "$summary" >/dev/null
    fail "ClickHouse restarted, was replaced, or reported OOMKilled during high-volume run"
  fi
  pass "ClickHouse stayed running without OOM during high-volume profile ingestion"
}

if [[ "$configure_profiler" == "true" && "$install" != "true" ]]; then
  log "## Configure Profiler Target Filters"
  if [[ "$load_local_images" == "true" ]]; then
    load_images_into_node
  fi
  helm upgrade --install "$release" ./deploy/helm \
    --namespace "$profiler_namespace" \
    --reuse-values \
    --set "clusterName=$(kubectl config current-context)" \
    --set "profiling.collectorInterval=${collector_interval}" \
    --set "profiling.enableAllocationAndLockJFR=${require_full}" \
    --set "profiling.targetNamespace=${namespace}" \
    --set "profiling.targetService=${service_name}"
  pass "profiler target filters configured for ${namespace}/${service_name}"
  if [[ "$load_local_images" == "true" ]]; then
    kubectl -n "$profiler_namespace" rollout restart "deploy/$release-backend" >/dev/null
    kubectl -n "$profiler_namespace" rollout restart "deploy/$release-web" >/dev/null
    kubectl -n "$profiler_namespace" rollout restart "daemonset/$release-collector" >/dev/null
    kubectl -n "$profiler_namespace" rollout status "deploy/$release-backend" --timeout=180s
    kubectl -n "$profiler_namespace" rollout status "deploy/$release-web" --timeout=180s
    kubectl -n "$profiler_namespace" rollout status "daemonset/$release-collector" --timeout=180s
    pass "profiler runtime restarted after loading local images"
  fi
  if kubectl -n "$namespace" get "deploy/$service_name" >/dev/null 2>&1; then
    kubectl -n "$namespace" annotate "deploy/$service_name" \
      "java-profiler.io/acceptance-run=${acceptance_started_at}" \
      "java-profiler.io/profile-disabled=false" \
      "java-profiler.io/profile-mode=temporary" \
      "java-profiler.io/profile-duration=1h" \
      "java-profiler.io/startup-delay=0s" \
      "java-profiler.io/snapshot-interval=10s" \
      --overwrite >/dev/null
    kubectl -n "$namespace" rollout restart "deploy/$service_name" >/dev/null
    kubectl -n "$namespace" rollout status "deploy/$service_name" --timeout=180s
    pass "target workload restarted after profiler configuration to avoid stale async-profiler conflict"
  fi
fi

log "## Cluster State"
kubectl -n "$namespace" get pods,svc,deploy,ds -o wide | tee "$artifact_dir/kubernetes-resources.txt" | tee -a "$summary" >/dev/null
if [[ "$profiler_namespace" != "$namespace" ]]; then
  kubectl -n "$profiler_namespace" get pods,svc,deploy,ds -o wide | tee "$artifact_dir/profiler-kubernetes-resources.txt" | tee -a "$summary" >/dev/null
fi

kubectl -n "$profiler_namespace" get deploy/clickhouse >/dev/null 2>&1 || fail "ClickHouse deployment not found in $profiler_namespace. Re-run with --install or deploy prerequisites."
kubectl -n "$profiler_namespace" get "deploy/$release-backend" >/dev/null 2>&1 || fail "backend deployment not found in $profiler_namespace. Re-run with --install or deploy Helm release."
kubectl -n "$profiler_namespace" get "deploy/$release-web" >/dev/null 2>&1 || fail "web deployment not found in $profiler_namespace. Re-run with --install or deploy Helm release."
kubectl -n "$profiler_namespace" get "daemonset/$release-collector" >/dev/null 2>&1 || fail "collector daemonset not found in $profiler_namespace. Re-run with --install or deploy Helm release."
if [[ "$skip_workload_rollout_check" != "true" ]]; then
  kubectl -n "$namespace" get "deploy/$service_name" >/dev/null 2>&1 || fail "Java workload deployment $service_name not found. Use --skip-workload-rollout-check when SERVICE is only a selector filter."
fi

kubectl -n "$profiler_namespace" rollout status deploy/clickhouse --timeout=120s
kubectl -n "$profiler_namespace" rollout status "deploy/$release-backend" --timeout=120s
kubectl -n "$profiler_namespace" rollout status "deploy/$release-web" --timeout=120s
kubectl -n "$profiler_namespace" rollout status "daemonset/$release-collector" --timeout=120s
if [[ "$skip_workload_rollout_check" != "true" ]]; then
  kubectl -n "$namespace" rollout status "deploy/$service_name" --timeout=120s
fi
pass "all required workloads are rolled out"

# The current chart has no readiness probes, so Kubernetes can report rollout
# success before the backend/web processes have opened their ports.
sleep "${JAVA_PROFILER_ACCEPTANCE_SETTLE_SECONDS:-10}"

web_pod="$(kubectl -n "$profiler_namespace" get pod -l app.kubernetes.io/name=java-profiler-web -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' | awk '{print $1}')"
backend_pod="$(kubectl -n "$profiler_namespace" get pod -l app.kubernetes.io/name=java-profiler-backend -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}' | awk '{print $1}')"
[[ -n "$web_pod" ]] || fail "web pod not found for port-forward"
[[ -n "$backend_pod" ]] || fail "backend pod not found for port-forward"
kubectl -n "$profiler_namespace" port-forward --address 127.0.0.1 "pod/$web_pod" "${web_port}:80" >"$artifact_dir/port-forward-web.log" 2>&1 &
cleanup_pids+=("$!")
kubectl -n "$profiler_namespace" port-forward --address 127.0.0.1 "pod/$backend_pod" "${backend_port}:8080" >"$artifact_dir/port-forward-backend.log" 2>&1 &
cleanup_pids+=("$!")
collector_pod="$(kubectl -n "$profiler_namespace" get pod -l app.kubernetes.io/name=java-profiler-collector -o jsonpath='{.items[0].metadata.name}')"
kubectl -n "$profiler_namespace" port-forward --address 127.0.0.1 "pod/$collector_pod" "${collector_port}:9090" >"$artifact_dir/port-forward-collector.log" 2>&1 &
cleanup_pids+=("$!")

wait_http "http://127.0.0.1:${web_port}/" || fail "web port-forward did not become ready"
wait_http "http://127.0.0.1:${backend_port}/metrics" || fail "backend port-forward did not become ready"
wait_http "http://127.0.0.1:${collector_port}/metrics" || fail "collector port-forward did not become ready"
pass "port-forwards are ready"

web_status_len="0"
for _ in $(seq 1 90); do
  curl -sS "http://127.0.0.1:${web_port}/api/ui/v1/target-status" >"$artifact_dir/web-target-status.json" || true
  web_status_len="$(jq 'length' "$artifact_dir/web-target-status.json" 2>/dev/null || echo 0)"
  if [[ "$web_status_len" -gt 0 ]]; then
    break
  fi
  sleep 1
done
if [[ "$web_status_len" -gt 0 ]]; then
  pass "web proxy returns target status JSON ($web_status_len rows)"
else
  fail "web proxy returned empty target status"
fi

backend_no_auth_code="$(curl -sS -o "$artifact_dir/backend-no-auth.txt" -w '%{http_code}' "http://127.0.0.1:${backend_port}/api/ui/v1/target-status")"
[[ "$backend_no_auth_code" == "401" ]] || fail "backend without UI token returned $backend_no_auth_code, expected 401"
pass "backend direct UI API rejects missing token"

if [[ "$high_volume" == "true" ]]; then
  capture_clickhouse_state before
fi

profile_wait_seconds="${JAVA_PROFILER_ACCEPTANCE_PROFILE_WAIT_SECONDS:-$(( $(duration_seconds "$collector_interval") * 2 + 10 ))}"
if [[ "$high_volume" == "true" ]]; then
  high_volume_seconds="${JAVA_PROFILER_HIGH_VOLUME_SECONDS:-240}"
  if [[ "$profile_wait_seconds" -lt "$high_volume_seconds" ]]; then
    profile_wait_seconds="$high_volume_seconds"
  fi
fi
drive_workload_load "$profile_wait_seconds" &
load_pid="$!"
cleanup_pids+=("$load_pid")
log "- waiting ${profile_wait_seconds}s for async-profiler start/stop/read cycles"
wait "$load_pid" || true

if [[ "$high_volume" == "true" ]]; then
  capture_clickhouse_state after
fi

start="${JAVA_PROFILER_ACCEPTANCE_START:-$acceptance_started_at}"
end="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
query="namespace=${namespace}&service=${service_name}&start=${start}&end=${end}"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/target-status?${query}" >"$artifact_dir/backend-target-status.json"
service_status_len="$(jq 'length' "$artifact_dir/backend-target-status.json")"
[[ "$service_status_len" -gt 0 ]] || fail "backend target status has no rows for ${namespace}/${service_name}"
pass "backend target status has rows for ${namespace}/${service_name} ($service_status_len rows)"
accepted_status_len="$(jq '[.[] | select(.reason == "accepted")] | length' "$artifact_dir/backend-target-status.json")"
if [[ "$accepted_status_len" -gt 0 ]]; then
  pass "backend target status has accepted rows for this acceptance window ($accepted_status_len rows)"
else
  gap "backend target status has no accepted rows for this acceptance window"
fi

curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_cpu_nanoseconds" >"$artifact_dir/backend-flamegraph-cpu.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_wall_clock_nanoseconds" >"$artifact_dir/backend-flamegraph-wall.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_io_wait_nanoseconds" >"$artifact_dir/backend-flamegraph-io-wait.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_allocation_bytes" >"$artifact_dir/backend-flamegraph-alloc-bytes.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/allocation-summary?${query}&profile_type=java_allocation_bytes" >"$artifact_dir/backend-allocation-summary.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_lock_delay_nanoseconds" >"$artifact_dir/backend-flamegraph-lock-delay.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/jvm-events?${query}&event_type=gc_pause" >"$artifact_dir/backend-jvm-events-gc.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/thread-diagnosis?${query}" >"$artifact_dir/backend-thread-diagnosis.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/deadlocks?${query}" >"$artifact_dir/backend-deadlocks.json"
ingestion_code="$(curl -sS -o "$artifact_dir/backend-ingestion.txt" -w '%{http_code}' -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/ingestion")"

curl -sS "http://127.0.0.1:${collector_port}/metrics" >"$artifact_dir/collector-metrics.txt"
curl -sS "http://127.0.0.1:${backend_port}/metrics" >"$artifact_dir/backend-metrics.txt"

kubectl -n "$profiler_namespace" exec deploy/clickhouse -- clickhouse-client --query \
  "WITH parseDateTime64BestEffort('${start}', 9, 'UTC') AS run_start SELECT 'target_status' t, count() FROM java_profiler.java_profiler_target_status WHERE namespace='${namespace}' AND service='${service_name}' AND status_at >= run_start UNION ALL SELECT 'ingestion_batches', count() FROM java_profiler.java_profiler_ingestion_batches WHERE received_at >= run_start UNION ALL SELECT 'profile_samples', count() FROM java_profiler.java_profiler_profile_samples WHERE namespace='${namespace}' AND service='${service_name}' AND created_at >= run_start UNION ALL SELECT 'profile_stacks', count() FROM java_profiler.java_profiler_profile_stacks WHERE namespace='${namespace}' AND service='${service_name}' AND created_at >= run_start UNION ALL SELECT 'jvm_events', count() FROM java_profiler.java_profiler_jvm_events WHERE namespace='${namespace}' AND service='${service_name}' AND created_at >= run_start UNION ALL SELECT 'thread_snapshots', count() FROM java_profiler.java_profiler_thread_snapshots WHERE namespace='${namespace}' AND service='${service_name}' AND created_at >= run_start UNION ALL SELECT 'deadlock_events', count() FROM java_profiler.java_profiler_deadlock_events WHERE namespace='${namespace}' AND service='${service_name}' AND created_at >= run_start UNION ALL SELECT 'artifact_index', count() FROM java_profiler.java_profiler_artifact_index WHERE created_at >= run_start" \
  >"$artifact_dir/clickhouse-counts.tsv"

kubectl -n "$profiler_namespace" exec deploy/clickhouse -- clickhouse-client --query \
  "WITH parseDateTime64BestEffort('${start}', 9, 'UTC') AS run_start SELECT batch_type, status, retryable, count(), max(received_at), any(message) FROM java_profiler.java_profiler_ingestion_batches WHERE received_at >= run_start GROUP BY batch_type, status, retryable ORDER BY batch_type, status FORMAT Vertical" \
  >"$artifact_dir/clickhouse-ingestion-batches.txt"

kubectl -n "$profiler_namespace" exec deploy/clickhouse -- clickhouse-client --query \
  "WITH parseDateTime64BestEffort('${start}', 9, 'UTC') AS run_start SELECT ifNull(sum(dropped_sample_count), 0), ifNull(sum(dropped_stack_count), 0), ifNull(max(truncated), 0), ifNull(max(batch_sample_count), 0), countIf(batch_type='profile' AND status='accepted'), countIf(batch_type='profile' AND status='rejected') FROM java_profiler.java_profiler_ingestion_batches WHERE received_at >= run_start FORMAT TSV" \
  >"$artifact_dir/clickhouse-profile-ingestion-metadata.tsv"

kubectl -n "$profiler_namespace" exec deploy/clickhouse -- clickhouse-client --query \
  "SHOW CREATE TABLE java_profiler.java_profiler_profile_samples FORMAT TSVRaw" \
  >"$artifact_dir/clickhouse-profile-samples-ddl.sql"
kubectl -n "$profiler_namespace" exec deploy/clickhouse -- clickhouse-client --query \
  "SHOW CREATE TABLE java_profiler.java_profiler_thread_snapshots FORMAT TSVRaw" \
  >"$artifact_dir/clickhouse-thread-snapshots-ddl.sql"

profile_samples="$(awk '$1=="profile_samples"{print $2}' "$artifact_dir/clickhouse-counts.tsv")"
profile_stacks="$(awk '$1=="profile_stacks"{print $2}' "$artifact_dir/clickhouse-counts.tsv")"
thread_snapshots="$(awk '$1=="thread_snapshots"{print $2}' "$artifact_dir/clickhouse-counts.tsv")"
deadlock_events="$(awk '$1=="deadlock_events"{print $2}' "$artifact_dir/clickhouse-counts.tsv")"
target_status="$(awk '$1=="target_status"{print $2}' "$artifact_dir/clickhouse-counts.tsv")"
ingestion_batches="$(awk '$1=="ingestion_batches"{print $2}' "$artifact_dir/clickhouse-counts.tsv")"
flamegraph_value="$(json_root_value "$artifact_dir/backend-flamegraph-cpu.json")"
alloc_flamegraph_value="$(json_root_value "$artifact_dir/backend-flamegraph-alloc-bytes.json")"
alloc_summary_has_data="$(jq -r '.coverage.has_data // false' "$artifact_dir/backend-allocation-summary.json" 2>/dev/null || echo false)"
alloc_summary_total="$(jq -r '.coverage.total_value // 0' "$artifact_dir/backend-allocation-summary.json" 2>/dev/null || echo 0)"
alloc_summary_paths="$(jq -r '.top_paths | length' "$artifact_dir/backend-allocation-summary.json" 2>/dev/null || echo 0)"
alloc_summary_self_frames="$(jq -r '.top_self_frames | length' "$artifact_dir/backend-allocation-summary.json" 2>/dev/null || echo 0)"
alloc_summary_insights="$(jq -r '.insights | length' "$artifact_dir/backend-allocation-summary.json" 2>/dev/null || echo 0)"
lock_flamegraph_value="$(json_root_value "$artifact_dir/backend-flamegraph-lock-delay.json")"
wall_flamegraph_value="$(json_root_value "$artifact_dir/backend-flamegraph-wall.json")"
io_flamegraph_value="$(json_root_value "$artifact_dir/backend-flamegraph-io-wait.json")"
gc_event_count="$(jq -r '.events | length' "$artifact_dir/backend-jvm-events-gc.json")"
jvm_events="$(awk '$1=="jvm_events"{print $2}' "$artifact_dir/clickhouse-counts.tsv")"
read -r dropped_sample_count dropped_stack_count max_truncated max_batch_sample_count accepted_profile_batches rejected_profile_batches <"$artifact_dir/clickhouse-profile-ingestion-metadata.tsv"

[[ "${target_status:-0}" -gt 0 ]] || fail "ClickHouse target_status is empty"
[[ "${ingestion_batches:-0}" -gt 0 ]] || fail "ClickHouse ingestion_batches is empty"
pass "ClickHouse control-plane tables contain target status and ingestion batch rows"

if [[ "${profile_samples:-0}" -gt 0 && "${profile_stacks:-0}" -gt 0 && "${flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty CPU profile path is working"
else
  gap "non-empty CPU profile path is not proven: profile_samples=${profile_samples:-0}, profile_stacks=${profile_stacks:-0}, flamegraph.root.value=${flamegraph_value:-0}"
fi

if [[ "${wall_flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty Wall Clock stack path is working"
else
  gap "non-empty Wall Clock stack path is not proven: wall flamegraph.root.value=${wall_flamegraph_value:-0}"
fi

if [[ "${io_flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty Java I/O wait stack path is working"
else
  gap "non-empty Java I/O wait stack path is not proven: io flamegraph.root.value=${io_flamegraph_value:-0}"
fi

if [[ "${gc_event_count:-0}" -gt 0 && "${jvm_events:-0}" -gt 0 ]]; then
  pass "non-empty JVM GC event path is working"
else
  gap "non-empty JVM GC event path is not proven: jvm_events=${jvm_events:-0}, gc_events=${gc_event_count:-0}"
fi

if [[ "${alloc_flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty allocation stack path is working"
else
  gap "non-empty allocation stack path is not proven: allocation flamegraph.root.value=${alloc_flamegraph_value:-0}"
fi

if [[ "$alloc_summary_has_data" == "true" && "${alloc_summary_total:-0}" -gt 0 && "${alloc_summary_paths:-0}" -gt 0 && "${alloc_summary_self_frames:-0}" -gt 0 && "${alloc_summary_insights:-0}" -gt 0 ]]; then
  pass "non-empty allocation summary API is working"
else
  gap "non-empty allocation summary API is not proven: has_data=${alloc_summary_has_data}, total=${alloc_summary_total:-0}, paths=${alloc_summary_paths:-0}, self_frames=${alloc_summary_self_frames:-0}, insights=${alloc_summary_insights:-0}"
fi

if [[ "${lock_flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty lock-delay stack path is working"
else
  gap "non-empty lock-delay stack path is not proven in this run: lock flamegraph.root.value=${lock_flamegraph_value:-0}"
fi

if [[ "${thread_snapshots:-0}" -gt 0 ]]; then
  pass "thread snapshot path is working"
else
  optional_gap "thread snapshot path is not proven in this run: thread_snapshots=${thread_snapshots:-0}"
fi

if [[ "${deadlock_events:-0}" -gt 0 ]]; then
  pass "deadlock event path has data"
else
  optional_gap "deadlock event path has no data in this run: deadlock_events=${deadlock_events:-0}"
fi

if [[ "$ingestion_code" == "200" ]]; then
  pass "backend ingestion UI API exists"
else
  gap "backend /api/ui/v1/ingestion returned HTTP ${ingestion_code}; UI ingestion view cannot be backed by a real query API yet"
fi

if [[ "$high_volume" == "true" ]]; then
  log "## High Volume Ingestion"
  log "- dropped_sample_count: ${dropped_sample_count:-0}"
  log "- dropped_stack_count: ${dropped_stack_count:-0}"
  log "- max_truncated: ${max_truncated:-0}"
  log "- max_batch_sample_count: ${max_batch_sample_count:-0}"
  log "- accepted_profile_batches: ${accepted_profile_batches:-0}"
  log "- rejected_profile_batches: ${rejected_profile_batches:-0}"

  [[ "${accepted_profile_batches:-0}" -gt 0 ]] || fail "high-volume run did not accept any profile batches"
  [[ "${rejected_profile_batches:-0}" -eq 0 ]] || fail "high-volume run rejected profile batches; inspect clickhouse-ingestion-batches.txt"
  [[ "${max_batch_sample_count:-0}" -le 10000 ]] || fail "profile batch exceeded collector max samples per batch: ${max_batch_sample_count}"
  if [[ "${max_truncated:-0}" -gt 0 || "${dropped_sample_count:-0}" -gt 0 || "${dropped_stack_count:-0}" -gt 0 ]]; then
    pass "high-volume run exercised bounded profile ingestion metadata"
  else
    pass "high-volume run stayed below limits with accepted profile batches"
  fi

  compare_clickhouse_state
fi

if grep -q 'TTL expires_at' "$artifact_dir/clickhouse-profile-samples-ddl.sql" && grep -q 'toIntervalDay(7)' "$artifact_dir/clickhouse-profile-samples-ddl.sql"; then
  pass "profile sample TTL is bounded to 7 days"
else
  fail "profile sample TTL is missing or not bounded to 7 days"
fi

if [[ "$skip_browser" != "true" ]]; then
  require_cmd node
  log "## Browser UI"
  (
    cd web
    REAL_ACCEPTANCE=1 \
    REAL_ACCEPTANCE_BASE_URL="http://127.0.0.1:${web_port}" \
    REAL_ACCEPTANCE_NAMESPACE="$namespace" \
    REAL_ACCEPTANCE_SERVICE="$service_name" \
    REAL_ACCEPTANCE_ARTIFACT_DIR="$artifact_dir" \
    npx playwright test tests/real-acceptance.spec.ts --config=playwright.config.ts --reporter=list --output="$artifact_dir/playwright-output"
  ) | tee "$artifact_dir/playwright.log" | tee -a "$summary" >/dev/null
  pass "browser UI acceptance completed"
fi

log "## Target After"
capture_target_state after
compare_target_restarts

log ""
log "Evidence written to: $artifact_dir"
