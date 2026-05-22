#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/deploy-java-profiler-latest.sh

Deploy the latest java-profiler control plane image set, configure profiling
for the target workload, and expose the Web UI on localhost.

Defaults:
  KUBECONFIG                 $HOME/backup/localk8s.yaml
  PROFILER_NAMESPACE         java-profiler
  TARGET_NAMESPACE           kd-cosmic-xk
  TARGET_DEPLOYMENT          mservice
  TARGET_CONTAINER           mservice
  TARGET_SERVICE             mservice
  LOCAL_UI_PORT              18081
  AUTH_SECRET                java-profiler-auth
  CLICKHOUSE_IMAGE           docker.m.daocloud.io/clickhouse/clickhouse-server:24.8
  CLICKHOUSE_USER            default
  CLICKHOUSE_PASSWORD        qa-clickhouse
  CLICKHOUSE_DSN             tcp://clickhouse:9000/java_profiler
  COLLECTOR_INTERVAL         60s
  PROFILE_MODE               continuous
  PROFILE_DURATION           1h
  STARTUP_DELAY              30s
  SNAPSHOT_INTERVAL          10s
  ENABLE_ALLOCATION_AND_LOCK_JFR true

Optional overrides:
  JAVA_PROFILER_VERSION      Pin a specific release tag instead of auto-detecting the latest.
  BACKEND_IMAGE              Override backend image reference.
  COLLECTOR_IMAGE            Override collector image reference.
  WEB_IMAGE                  Override web image reference.
  CLICKHOUSE_IMAGE           Override ClickHouse image reference.
  CLICKHOUSE_USER            Override ClickHouse user.
  CLICKHOUSE_PASSWORD        Override ClickHouse password.
  PROFILER_NAMESPACE         Override control plane namespace. Auto-detects an existing
                            java-profiler Helm release namespace when unset.
  IMAGE_MODE                 workspace (default) builds current workspace images first;
                            release uses the latest published images.
  UI_TOKEN                   UI token to store in the auth secret.
  COLLECTOR_TOKEN            Collector token to store in the auth secret.
  CLICKHOUSE_DSN             Override the ClickHouse connection string.
  RELEASE_NAME               Helm release name. Default: java-profiler.
  IMAGE_REPO_PREFIX          Image registry prefix. Default: ghcr.io/koolay.

Example:
  export KUBECONFIG=$HOME/backup/localk8s.yaml
  scripts/deploy-java-profiler-latest.sh
USAGE
}

log() {
  printf '%s\n' "$*"
}

fail() {
  log "ERROR: $*"
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

resolve_latest_tag() {
  local repo_slug="koolay/java-profiler"
  local tag=""

  if command -v gh >/dev/null 2>&1; then
    tag="$(gh release list \
      --repo "$repo_slug" \
      --exclude-drafts \
      --exclude-pre-releases \
      --limit 1 \
      --json tagName \
      --jq '.[0].tagName' 2>/dev/null || true)"
  fi

  if [[ -z "$tag" ]]; then
    tag="$(git ls-remote --tags "https://github.com/${repo_slug}.git" 'refs/tags/v*' \
      | awk '{print $2}' \
      | sed 's#refs/tags/##' \
      | sed 's/\^{}//' \
      | sort -V \
      | tail -n 1)"
  fi

  [[ -n "$tag" ]] || fail "unable to determine the latest release tag"
  case "$tag" in
    v*) printf '%s\n' "$tag" ;;
    *) fail "latest release tag does not look like a release tag: $tag" ;;
  esac
}

build_current_workspace_images() {
  require_cmd docker
  require_cmd go
  require_cmd npm

  if [[ -z "${WORKSPACE_IMAGE_TAG:-}" ]]; then
    WORKSPACE_IMAGE_TAG="workspace-$(date +%Y%m%d%H%M%S)"
  fi

  BACKEND_IMAGE="${BACKEND_IMAGE:-java-profiler-backend:${WORKSPACE_IMAGE_TAG}}"
  COLLECTOR_IMAGE="${COLLECTOR_IMAGE:-java-profiler-collector:${WORKSPACE_IMAGE_TAG}}"
  WEB_IMAGE="${WEB_IMAGE:-java-profiler-web:${WORKSPACE_IMAGE_TAG}}"

  log "- building current workspace images: ${BACKEND_IMAGE}, ${COLLECTOR_IMAGE}, ${WEB_IMAGE}"
  BACKEND_IMAGE="$BACKEND_IMAGE" \
  COLLECTOR_IMAGE="$COLLECTOR_IMAGE" \
  WEB_IMAGE="$WEB_IMAGE" \
  WORKSPACE_IMAGE_TAG="$WORKSPACE_IMAGE_TAG" \
    bash scripts/build-real-acceptance-images.sh
}

load_images_into_node() {
  local images=("$BACKEND_IMAGE" "$COLLECTOR_IMAGE" "$WEB_IMAGE")

  cat <<YAML | kubectl -n "$PROFILER_NAMESPACE" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: image-loader
spec:
  restartPolicy: Never
  hostPID: true
  containers:
    - name: loader
      image: ${CLICKHOUSE_IMAGE}
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

  kubectl -n "$PROFILER_NAMESPACE" wait --for=condition=Ready pod/image-loader --timeout=120s >/dev/null
  kubectl -n "$PROFILER_NAMESPACE" exec image-loader -- chroot /host /usr/bin/ctr version >/dev/null

  for image in "${images[@]}"; do
    local safe
    safe="$(printf '%s' "$image" | tr '/:' '__')"
    local tar_path="$ARTIFACT_DIR/$safe.tar"
    log "- loading $image into node containerd"
    docker save "$image" -o "$tar_path"
    kubectl -n "$PROFILER_NAMESPACE" cp "$tar_path" "image-loader:/host/tmp/$safe.tar" >/dev/null
    kubectl -n "$PROFILER_NAMESPACE" exec image-loader -- chroot /host /usr/bin/ctr -n k8s.io images import "/tmp/$safe.tar" >/dev/null
  done

  kubectl -n "$PROFILER_NAMESPACE" delete pod image-loader --ignore-not-found=true --wait=true >/dev/null
}

detect_release_namespace() {
  local release_name="$1"
  local ns=""

  if command -v helm >/dev/null 2>&1; then
    ns="$(helm list -A --filter "^${release_name}$" 2>/dev/null | awk -v release="$release_name" 'NR > 1 && $1 == release {print $2; exit}' || true)"
  fi

  if [[ -z "$ns" ]]; then
    ns="$(kubectl get clusterrole java-profiler-collector -o jsonpath='{.metadata.annotations.meta\.helm\.sh/release-namespace}' 2>/dev/null || true)"
  fi

  printf '%s\n' "$ns"
}

wait_for_rollout() {
  local kind="$1"
  local name="$2"
  local namespace="$3"
  kubectl -n "$namespace" rollout status "$kind/$name" --timeout=10m >/dev/null
}

wait_for_http() {
  local url="$1"
  local deadline=$((SECONDS + 45))
  while (( SECONDS < deadline )); do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

apply_clickhouse() {
  cat <<YAML | kubectl -n "$PROFILER_NAMESPACE" apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clickhouse
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
          image: ${CLICKHOUSE_IMAGE}
          ports:
            - name: native
              containerPort: 9000
            - name: http
              containerPort: 8123
          env:
            - name: CLICKHOUSE_DB
              value: java_profiler
            - name: CLICKHOUSE_USER
              value: ${CLICKHOUSE_USER}
            - name: CLICKHOUSE_PASSWORD
              value: ${CLICKHOUSE_PASSWORD}
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
}

ensure_target_container() {
  local containers
  containers="$(kubectl -n "$TARGET_NAMESPACE" get "deploy/$TARGET_DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[*].name}')"
  case " $containers " in
    *" $TARGET_CONTAINER "*) ;;
    *)
      fail "container '$TARGET_CONTAINER' was not found in deployment '$TARGET_NAMESPACE/$TARGET_DEPLOYMENT' (found: ${containers:-none})"
      ;;
  esac
}

start_ui_port_forward() {
  local pid_file="$ARTIFACT_DIR/ui-port-forward.pid"
  local log_file="$ARTIFACT_DIR/ui-port-forward.log"
  nohup kubectl -n "$PROFILER_NAMESPACE" port-forward --address 127.0.0.1 \
    "svc/${RELEASE_NAME}-web" "${LOCAL_UI_PORT}:80" \
    >"$log_file" 2>&1 </dev/null &
  printf '%s\n' "$!" >"$pid_file"
}

main() {
  require_cmd kubectl
  require_cmd helm
  require_cmd curl
  require_cmd awk
  require_cmd sed
  require_cmd sort
  require_cmd tail

  export KUBECONFIG="${KUBECONFIG:-$HOME/backup/localk8s.yaml}"

  PROFILER_NAMESPACE="${PROFILER_NAMESPACE:-}"
  TARGET_NAMESPACE="${TARGET_NAMESPACE:-kd-cosmic-xk}"
  TARGET_DEPLOYMENT="${TARGET_DEPLOYMENT:-mservice}"
  TARGET_CONTAINER="${TARGET_CONTAINER:-mservice}"
  TARGET_SERVICE="${TARGET_SERVICE:-${TARGET_DEPLOYMENT}}"
  RELEASE_NAME="${RELEASE_NAME:-java-profiler}"
  LOCAL_UI_PORT="${LOCAL_UI_PORT:-18081}"
  AUTH_SECRET="${AUTH_SECRET:-java-profiler-auth}"
  CLICKHOUSE_IMAGE="${CLICKHOUSE_IMAGE:-docker.m.daocloud.io/clickhouse/clickhouse-server:24.8}"
  CLICKHOUSE_USER="${CLICKHOUSE_USER:-default}"
  CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-qa-clickhouse}"
  UI_TOKEN="${UI_TOKEN:-java-profiler-ui-token}"
  COLLECTOR_TOKEN="${COLLECTOR_TOKEN:-java-profiler-collector-token}"
  COLLECTOR_INTERVAL="${COLLECTOR_INTERVAL:-60s}"
  CLICKHOUSE_DSN="${CLICKHOUSE_DSN:-tcp://${CLICKHOUSE_USER}:${CLICKHOUSE_PASSWORD}@clickhouse:9000/java_profiler}"
  PROFILE_MODE="${PROFILE_MODE:-continuous}"
  PROFILE_DURATION="${PROFILE_DURATION:-1h}"
  STARTUP_DELAY="${STARTUP_DELAY:-30s}"
  SNAPSHOT_INTERVAL="${SNAPSHOT_INTERVAL:-10s}"
  ENABLE_ALLOCATION_AND_LOCK_JFR="${ENABLE_ALLOCATION_AND_LOCK_JFR:-true}"
  IMAGE_REPO_PREFIX="${IMAGE_REPO_PREFIX:-ghcr.io/koolay}"
  IMAGE_MODE="${IMAGE_MODE:-workspace}"
  CLUSTER_NAME="$(kubectl config current-context)"
  ARTIFACT_DIR="${ARTIFACT_DIR:-/tmp/java-profiler-deploy-$(date +%Y%m%d-%H%M%S)}"
  mkdir -p "$ARTIFACT_DIR"
  UI_PORT_FORWARD_PID=""

  cleanup() {
    local status=$?
    if [[ $status -ne 0 && -n "${UI_PORT_FORWARD_PID:-}" ]]; then
      kill "$UI_PORT_FORWARD_PID" >/dev/null 2>&1 || true
    fi
  }
  trap cleanup EXIT

  if [[ -z "$PROFILER_NAMESPACE" ]]; then
    PROFILER_NAMESPACE="$(detect_release_namespace "$RELEASE_NAME")"
  fi
  if [[ -z "$PROFILER_NAMESPACE" ]]; then
    PROFILER_NAMESPACE="java-profiler"
  fi

  latest_tag=""
  case "$IMAGE_MODE" in
    workspace)
      build_current_workspace_images
      latest_tag="${WORKSPACE_IMAGE_TAG}"
      ;;
    release)
      latest_tag="${JAVA_PROFILER_VERSION:-$(resolve_latest_tag)}"
      BACKEND_IMAGE="${BACKEND_IMAGE:-${IMAGE_REPO_PREFIX}/java-profiler-backend:${latest_tag}}"
      COLLECTOR_IMAGE="${COLLECTOR_IMAGE:-${IMAGE_REPO_PREFIX}/java-profiler-collector:${latest_tag}}"
      WEB_IMAGE="${WEB_IMAGE:-${IMAGE_REPO_PREFIX}/java-profiler-web:${latest_tag}}"
      ;;
    *)
      fail "unknown IMAGE_MODE: $IMAGE_MODE (expected workspace or release)"
      ;;
  esac

  cat >"$ARTIFACT_DIR/summary.txt" <<EOF
cluster=${CLUSTER_NAME}
profiler_namespace=${PROFILER_NAMESPACE}
target=${TARGET_NAMESPACE}/${TARGET_DEPLOYMENT}
target_container=${TARGET_CONTAINER}
target_service=${TARGET_SERVICE}
release=${RELEASE_NAME}
version=${latest_tag}
image_mode=${IMAGE_MODE}
allocation_and_lock_jfr=${ENABLE_ALLOCATION_AND_LOCK_JFR}
backend_image=${BACKEND_IMAGE}
collector_image=${COLLECTOR_IMAGE}
web_image=${WEB_IMAGE}
ui_port=${LOCAL_UI_PORT}
EOF

  log "# java-profiler deployment"
  log "- kubeconfig: ${KUBECONFIG}"
  log "- version: ${latest_tag}"
  log "- image mode: ${IMAGE_MODE}"
  log "- allocation/lock JFR: ${ENABLE_ALLOCATION_AND_LOCK_JFR}"
  log "- profiler namespace: ${PROFILER_NAMESPACE}"
  log "- target: ${TARGET_NAMESPACE}/${TARGET_DEPLOYMENT} (service ${TARGET_SERVICE}, container ${TARGET_CONTAINER})"
  log "- clickhouse: ${CLICKHOUSE_IMAGE} (user ${CLICKHOUSE_USER})"
  log "- ui: http://127.0.0.1:${LOCAL_UI_PORT}"
  log "- artifact dir: ${ARTIFACT_DIR}"

  ensure_target_container

  kubectl create namespace "$PROFILER_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  kubectl -n "$PROFILER_NAMESPACE" create secret generic "$AUTH_SECRET" \
    --from-literal=collector-token="$COLLECTOR_TOKEN" \
    --from-literal=ui-token="$UI_TOKEN" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null

  log "- applying ClickHouse into ${PROFILER_NAMESPACE}"
  apply_clickhouse
  wait_for_rollout deploy clickhouse "$PROFILER_NAMESPACE"
  log "- ClickHouse ready"

  if [[ "$IMAGE_MODE" == "workspace" ]]; then
    load_images_into_node
  fi

  helm upgrade --install "$RELEASE_NAME" ./deploy/helm \
    --namespace "$PROFILER_NAMESPACE" \
    --create-namespace \
    --wait \
    --timeout 10m \
    --rollback-on-failure \
    --set "clusterName=${CLUSTER_NAME}" \
    --set "image.backend=${BACKEND_IMAGE}" \
    --set "image.collector=${COLLECTOR_IMAGE}" \
    --set "image.web=${WEB_IMAGE}" \
    --set "clickhouse.dsn=${CLICKHOUSE_DSN}" \
    --set "auth.existingSecret=${AUTH_SECRET}" \
    --set "auth.collectorTokenKey=collector-token" \
    --set "auth.uiTokenKey=ui-token" \
    --set "profiling.collectorInterval=${COLLECTOR_INTERVAL}" \
    --set "profiling.enableAllocationAndLockJFR=${ENABLE_ALLOCATION_AND_LOCK_JFR}" \
    --set "profiling.targetNamespace=${TARGET_NAMESPACE}" \
    --set "profiling.targetService=${TARGET_SERVICE}" \
    >/dev/null

  wait_for_rollout deploy clickhouse "$PROFILER_NAMESPACE"
  wait_for_rollout deploy "$RELEASE_NAME-backend" "$PROFILER_NAMESPACE"
  wait_for_rollout deploy "$RELEASE_NAME-web" "$PROFILER_NAMESPACE"
  wait_for_rollout daemonset "$RELEASE_NAME-collector" "$PROFILER_NAMESPACE"

  profile_duration_json=""
  if [[ "$PROFILE_MODE" == "temporary" ]]; then
    profile_duration_json=$',\n  "java-profiler.io/profile-duration":"'"${PROFILE_DURATION}"'"'
  fi
  kubectl -n "$TARGET_NAMESPACE" patch "deploy/$TARGET_DEPLOYMENT" --type merge -p "$(cat <<EOF
{"spec":{"template":{"metadata":{"annotations":{
  "java-profiler.io/acceptance-run":"$(date -u +%Y%m%dT%H%M%SZ)",
  "java-profiler.io/profile-disabled":"false",
  "java-profiler.io/profile-mode":"${PROFILE_MODE}",
  "java-profiler.io/startup-delay":"${STARTUP_DELAY}",
  "java-profiler.io/snapshot-interval":"${SNAPSHOT_INTERVAL}"${profile_duration_json}
}}}}}
EOF
)" >/dev/null

  log "- workload metadata applied to the Pod template"
  kubectl -n "$TARGET_NAMESPACE" rollout restart "deploy/$TARGET_DEPLOYMENT" >/dev/null
  wait_for_rollout deploy "$TARGET_DEPLOYMENT" "$TARGET_NAMESPACE"

  start_ui_port_forward
  UI_PORT_FORWARD_PID="$(cat "$ARTIFACT_DIR/ui-port-forward.pid")"
  if ! wait_for_http "http://127.0.0.1:${LOCAL_UI_PORT}/"; then
    log ""
    log "UI port-forward did not become ready."
    log "Log file: $ARTIFACT_DIR/ui-port-forward.log"
    log "Last 20 lines:"
    tail -n 20 "$ARTIFACT_DIR/ui-port-forward.log" || true
    fail "unable to reach the UI on http://127.0.0.1:${LOCAL_UI_PORT}/"
  fi

  log ""
  log "Deployment complete."
  log "Open the UI at: http://127.0.0.1:${LOCAL_UI_PORT}/"
  log "Port-forward PID: $(cat "$ARTIFACT_DIR/ui-port-forward.pid")"
  log "To stop the UI port-forward: kill $(cat "$ARTIFACT_DIR/ui-port-forward.pid")"
  log "Summary: $ARTIFACT_DIR/summary.txt"
}

main "$@"
