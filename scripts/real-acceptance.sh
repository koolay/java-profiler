#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  KUBECONFIG=/Users/huwl/backup/localk8s.yaml scripts/real-acceptance.sh [options]

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
  --require-full-profiling  Fail if profile/thread/deadlock data is still empty.
  --skip-browser            Skip Playwright UI screenshots/video.
  -h, --help                Show this help.

Environment:
  KUBECONFIG                Required. Example: /Users/huwl/backup/localk8s.yaml.
  BACKEND_IMAGE             Default: ghcr.io/koolay/java-profiler-backend:0.1.0.
  COLLECTOR_IMAGE           Default: ghcr.io/koolay/java-profiler-collector:0.1.0.
  WEB_IMAGE                 Default: ghcr.io/koolay/java-profiler-web:0.1.0.
  CLICKHOUSE_IMAGE          Default: docker.m.daocloud.io/clickhouse/clickhouse-server:24.8.
  JAVA_WORKLOAD_IMAGE       Default: docker.m.daocloud.io/eclipse-temurin:21-jdk.
  JAVA_WORKLOAD_PREBUILT    Set to 1 when JAVA_WORKLOAD_IMAGE already starts a CPU-busy Java app.
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
    --skip-browser) skip_browser="true"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 2 ;;
  esac
done

: "${KUBECONFIG:?KUBECONFIG is required, for example /Users/huwl/backup/localk8s.yaml}"

export NO_PROXY='*'
export no_proxy='*'

backend_image="${BACKEND_IMAGE:-ghcr.io/koolay/java-profiler-backend:0.1.0}"
collector_image="${COLLECTOR_IMAGE:-ghcr.io/koolay/java-profiler-collector:0.1.0}"
web_image="${WEB_IMAGE:-ghcr.io/koolay/java-profiler-web:0.1.0}"
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

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
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
              memory: 2Gi
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
                '        long x = 0;' \
                '        while (true) {' \
                '          byte[] payload = new byte[128 * 1024];' \
                '          for (int i = 0; i < payload.length; i += 4096) payload[i] = (byte)i;' \
                '          sink = payload;' \
                '          for (int i = 0; i < 500000; i++) x += i;' \
                '          if (x == Long.MIN_VALUE) System.out.println(x);' \
                '          synchronized (LOCK) {' \
                '            try { Thread.sleep(3); } catch (InterruptedException e) { Thread.currentThread().interrupt(); }' \
                '          }' \
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

if [[ "$configure_profiler" == "true" && "$install" != "true" ]]; then
  log "## Configure Profiler Target Filters"
  helm upgrade --install "$release" ./deploy/helm \
    --namespace "$profiler_namespace" \
    --reuse-values \
    --set "clusterName=$(kubectl config current-context)" \
    --set "profiling.collectorInterval=${collector_interval}" \
    --set "profiling.targetNamespace=${namespace}" \
    --set "profiling.targetService=${service_name}"
  pass "profiler target filters configured for ${namespace}/${service_name}"
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

kubectl -n "$profiler_namespace" port-forward --address 127.0.0.1 "svc/$release-web" "${web_port}:80" >"$artifact_dir/port-forward-web.log" 2>&1 &
cleanup_pids+=("$!")
kubectl -n "$profiler_namespace" port-forward --address 127.0.0.1 "svc/$release-backend" "${backend_port}:8080" >"$artifact_dir/port-forward-backend.log" 2>&1 &
cleanup_pids+=("$!")
collector_pod="$(kubectl -n "$profiler_namespace" get pod -l app.kubernetes.io/name=java-profiler-collector -o jsonpath='{.items[0].metadata.name}')"
kubectl -n "$profiler_namespace" port-forward --address 127.0.0.1 "pod/$collector_pod" "${collector_port}:9090" >"$artifact_dir/port-forward-collector.log" 2>&1 &
cleanup_pids+=("$!")

wait_http "http://127.0.0.1:${web_port}/" || fail "web port-forward did not become ready"
wait_http "http://127.0.0.1:${backend_port}/metrics" || fail "backend port-forward did not become ready"
wait_http "http://127.0.0.1:${collector_port}/metrics" || fail "collector port-forward did not become ready"
pass "port-forwards are ready"

curl -sS "http://127.0.0.1:${web_port}/api/ui/v1/target-status" >"$artifact_dir/web-target-status.json"
web_status_len="$(jq 'length' "$artifact_dir/web-target-status.json")"
if [[ "$web_status_len" -gt 0 ]]; then
  pass "web proxy returns target status JSON ($web_status_len rows)"
else
  fail "web proxy returned empty target status"
fi

backend_no_auth_code="$(curl -sS -o "$artifact_dir/backend-no-auth.txt" -w '%{http_code}' "http://127.0.0.1:${backend_port}/api/ui/v1/target-status")"
[[ "$backend_no_auth_code" == "401" ]] || fail "backend without UI token returned $backend_no_auth_code, expected 401"
pass "backend direct UI API rejects missing token"

start="$(date -u -v-2H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '2 hours ago' +%Y-%m-%dT%H:%M:%SZ)"
end="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
query="namespace=${namespace}&service=${service_name}&start=${start}&end=${end}"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/target-status?${query}" >"$artifact_dir/backend-target-status.json"
service_status_len="$(jq 'length' "$artifact_dir/backend-target-status.json")"
[[ "$service_status_len" -gt 0 ]] || fail "backend target status has no rows for ${namespace}/${service_name}"
pass "backend target status has rows for ${namespace}/${service_name} ($service_status_len rows)"

curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_cpu_nanoseconds" >"$artifact_dir/backend-flamegraph-cpu.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_allocation_bytes" >"$artifact_dir/backend-flamegraph-alloc-bytes.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/flamegraph?${query}&profile_type=java_lock_delay_nanoseconds" >"$artifact_dir/backend-flamegraph-lock-delay.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/thread-diagnosis?${query}" >"$artifact_dir/backend-thread-diagnosis.json"
curl -sS -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/deadlocks?${query}" >"$artifact_dir/backend-deadlocks.json"
ingestion_code="$(curl -sS -o "$artifact_dir/backend-ingestion.txt" -w '%{http_code}' -H "Authorization: Bearer ${ui_token}" "http://127.0.0.1:${backend_port}/api/ui/v1/ingestion")"

curl -sS "http://127.0.0.1:${collector_port}/metrics" >"$artifact_dir/collector-metrics.txt"
curl -sS "http://127.0.0.1:${backend_port}/metrics" >"$artifact_dir/backend-metrics.txt"

kubectl -n "$profiler_namespace" exec deploy/clickhouse -- clickhouse-client --query \
  "SELECT 'target_status' t, count() FROM java_profiler.java_profiler_target_status WHERE namespace='${namespace}' AND service='${service_name}' UNION ALL SELECT 'ingestion_batches', count() FROM java_profiler.java_profiler_ingestion_batches UNION ALL SELECT 'profile_samples', count() FROM java_profiler.java_profiler_profile_samples WHERE namespace='${namespace}' AND service='${service_name}' UNION ALL SELECT 'profile_stacks', count() FROM java_profiler.java_profiler_profile_stacks WHERE namespace='${namespace}' AND service='${service_name}' UNION ALL SELECT 'thread_snapshots', count() FROM java_profiler.java_profiler_thread_snapshots WHERE namespace='${namespace}' AND service='${service_name}' UNION ALL SELECT 'deadlock_events', count() FROM java_profiler.java_profiler_deadlock_events WHERE namespace='${namespace}' AND service='${service_name}' UNION ALL SELECT 'artifact_index', count() FROM java_profiler.java_profiler_artifact_index" \
  >"$artifact_dir/clickhouse-counts.tsv"

kubectl -n "$profiler_namespace" exec deploy/clickhouse -- clickhouse-client --query \
  "SELECT batch_type, status, retryable, count(), max(received_at), any(message) FROM java_profiler.java_profiler_ingestion_batches GROUP BY batch_type, status, retryable ORDER BY batch_type, status FORMAT Vertical" \
  >"$artifact_dir/clickhouse-ingestion-batches.txt"

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
flamegraph_value="$(jq -r '.root.value // 0' "$artifact_dir/backend-flamegraph-cpu.json")"
alloc_flamegraph_value="$(jq -r '.root.value // 0' "$artifact_dir/backend-flamegraph-alloc-bytes.json")"
lock_flamegraph_value="$(jq -r '.root.value // 0' "$artifact_dir/backend-flamegraph-lock-delay.json")"

[[ "${target_status:-0}" -gt 0 ]] || fail "ClickHouse target_status is empty"
[[ "${ingestion_batches:-0}" -gt 0 ]] || fail "ClickHouse ingestion_batches is empty"
pass "ClickHouse control-plane tables contain target status and ingestion batch rows"

if [[ "${profile_samples:-0}" -gt 0 && "${profile_stacks:-0}" -gt 0 && "${flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty CPU profile path is working"
else
  gap "non-empty CPU profile path is not proven: profile_samples=${profile_samples:-0}, profile_stacks=${profile_stacks:-0}, flamegraph.root.value=${flamegraph_value:-0}"
fi

if [[ "${alloc_flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty allocation stack path is working"
else
  gap "non-empty allocation stack path is not proven: allocation flamegraph.root.value=${alloc_flamegraph_value:-0}"
fi

if [[ "${lock_flamegraph_value:-0}" -gt 0 ]]; then
  pass "non-empty lock-delay stack path is working"
else
  gap "non-empty lock-delay stack path is not proven in this run: lock flamegraph.root.value=${lock_flamegraph_value:-0}"
fi

if [[ "${thread_snapshots:-0}" -gt 0 ]]; then
  pass "thread snapshot path is working"
else
  gap "thread snapshot path is not proven: thread_snapshots=${thread_snapshots:-0}"
fi

if [[ "${deadlock_events:-0}" -gt 0 ]]; then
  pass "deadlock event path has data"
else
  gap "deadlock event path has no data in this run: deadlock_events=${deadlock_events:-0}"
fi

if [[ "$ingestion_code" == "200" ]]; then
  pass "backend ingestion UI API exists"
else
  gap "backend /api/ui/v1/ingestion returned HTTP ${ingestion_code}; UI ingestion view cannot be backed by a real query API yet"
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
