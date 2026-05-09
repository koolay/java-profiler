#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/deploy-jdk17-demo.sh [options]

Deploy the JDK 17 HTTP demo workload to Kubernetes for java-profiler E2E testing.

Options:
  --namespace NAME          Kubernetes namespace. Default: java-profiler-qa.
  --image IMAGE             Demo image. Default: ghcr.io/koolay/java-profiler-jdk17-http-demo:latest.
  --platform PLATFORM       Docker build platform for --build-image. Default: linux/amd64.
  --build-image             Build examples/jdk17-http-demo before deploying.
  --load-to-node            Import the image into the Kubernetes node containerd runtime.
  --source-mode             Run the demo from a source ConfigMap on a JDK image.
  --source-image IMAGE      JDK image for --source-mode. Default: docker.m.daocloud.io/eclipse-temurin:21-jdk.
  --kind-load               Load the image into a kind cluster after building.
  --kind-cluster NAME       kind cluster name. Default: kind.
  --run-load                Port-forward the service and call CPU/alloc/lock/thread endpoints.
  --duration-ms N           Load duration per endpoint. Default: 30000.
  --local-port PORT         Local port for port-forward. Default: 18080.
  --artifact-dir DIR        Write deployment evidence. Default: /tmp/java-profiler-jdk17-demo-<timestamp>.
  -h, --help                Show this help.

Environment:
  KUBECONFIG                Optional kubectl config path.

Next step after this script:
  KUBECONFIG=/path/to/kubeconfig scripts/real-acceptance.sh \
    --configure-profiler \
    --namespace java-profiler-qa \
    --service jdk17-http-demo \
    --require-full-profiling
USAGE
}

namespace="java-profiler-qa"
image="ghcr.io/koolay/java-profiler-jdk17-http-demo:latest"
platform="linux/amd64"
build_image="false"
load_to_node="false"
source_mode="false"
source_image="docker.m.daocloud.io/eclipse-temurin:21-jdk"
kind_load="false"
kind_cluster="kind"
run_load="false"
duration_ms="30000"
local_port="18080"
artifact_dir="/tmp/java-profiler-jdk17-demo-$(date +%Y%m%d-%H%M%S)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --namespace) namespace="$2"; shift 2 ;;
    --image) image="$2"; shift 2 ;;
    --platform) platform="$2"; shift 2 ;;
    --build-image) build_image="true"; shift ;;
    --load-to-node) load_to_node="true"; shift ;;
    --source-mode) source_mode="true"; shift ;;
    --source-image) source_image="$2"; shift 2 ;;
    --kind-load) kind_load="true"; shift ;;
    --kind-cluster) kind_cluster="$2"; shift 2 ;;
    --run-load) run_load="true"; shift ;;
    --duration-ms) duration_ms="$2"; shift 2 ;;
    --local-port) local_port="$2"; shift 2 ;;
    --artifact-dir) artifact_dir="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 2 ;;
  esac
done

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

cleanup_pids=()
cleanup() {
  for pid in "${cleanup_pids[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT

require_cmd kubectl
require_cmd curl

mkdir -p "$artifact_dir"
summary="$artifact_dir/summary.md"
: > "$summary"

log() {
  printf '%s\n' "$*" | tee -a "$summary"
}

if [[ "$build_image" == "true" ]]; then
  require_cmd docker
  log "## Build Image"
  docker buildx build --platform "$platform" --load -t "$image" examples/jdk17-http-demo | tee "$artifact_dir/docker-build.log"
fi

if [[ "$kind_load" == "true" ]]; then
  require_cmd kind
  log "## Load Image Into kind"
  kind load docker-image "$image" --name "$kind_cluster" | tee "$artifact_dir/kind-load.log"
fi

log "# JDK 17 Demo Kubernetes Deployment"
log ""
log "- namespace: $namespace"
log "- image: $image"
log "- artifact_dir: $artifact_dir"
log "- context: $(kubectl config current-context)"
log ""

kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f - | tee "$artifact_dir/namespace-apply.log"

if [[ "$source_mode" == "true" ]]; then
  log "## Apply Source ConfigMap"
  kubectl -n "$namespace" create configmap jdk17-http-demo-source \
    --from-file=DemoHttpService.java=examples/jdk17-http-demo/src/main/java/com/ebpfjava/examples/httpdemo/DemoHttpService.java \
    --dry-run=client -o yaml | kubectl apply -f - | tee "$artifact_dir/source-configmap.log"
fi

if [[ "$load_to_node" == "true" ]]; then
  require_cmd docker
  log "## Load Image Into Node containerd"
  safe_image="$(printf '%s' "$image" | tr '/:' '__')"
  image_tar="$artifact_dir/${safe_image}.tar"
  docker save "$image" -o "$image_tar"
  cat <<YAML | kubectl -n "$namespace" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: jdk17-demo-image-loader
spec:
  restartPolicy: Never
  hostPID: true
  containers:
    - name: loader
      image: ghcr.io/koolay/library/busybox:1.37.0
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
  kubectl -n "$namespace" wait --for=condition=Ready pod/jdk17-demo-image-loader --timeout=120s | tee "$artifact_dir/image-loader-ready.log"
  kubectl -n "$namespace" cp "$image_tar" "jdk17-demo-image-loader:/host/tmp/${safe_image}.tar"
  kubectl -n "$namespace" exec jdk17-demo-image-loader -- chroot /host /usr/bin/ctr -n k8s.io images import "/tmp/${safe_image}.tar" | tee "$artifact_dir/image-import.log"
  kubectl -n "$namespace" delete pod jdk17-demo-image-loader --ignore-not-found=true --wait=true | tee "$artifact_dir/image-loader-delete.log"
fi

kubectl -n "$namespace" apply -f examples/jdk17-http-demo/k8s.yaml | tee "$artifact_dir/demo-apply.log"
if [[ "$source_mode" == "true" ]]; then
  kubectl -n "$namespace" set image deploy/jdk17-http-demo "app=$source_image" | tee "$artifact_dir/demo-set-image.log"
  kubectl -n "$namespace" patch deploy/jdk17-http-demo --type=json -p='[
    {"op":"add","path":"/spec/template/spec/volumes","value":[{"name":"demo-source","configMap":{"name":"jdk17-http-demo-source"}}]},
    {"op":"add","path":"/spec/template/spec/containers/0/volumeMounts","value":[{"name":"demo-source","mountPath":"/src"}]},
    {"op":"add","path":"/spec/template/spec/containers/0/command","value":["/bin/sh","-lc"]},
    {"op":"add","path":"/spec/template/spec/containers/0/args","value":["mkdir -p /tmp/classes && javac --release 17 -d /tmp/classes /src/DemoHttpService.java && exec java -XX:+UseContainerSupport -cp /tmp/classes com.ebpfjava.examples.httpdemo.DemoHttpService"]}
  ]' | tee "$artifact_dir/demo-source-patch.log"
else
  kubectl -n "$namespace" set image deploy/jdk17-http-demo "app=$image" | tee "$artifact_dir/demo-set-image.log"
fi
kubectl -n "$namespace" rollout status deploy/jdk17-http-demo --timeout=180s | tee "$artifact_dir/demo-rollout.log"

kubectl -n "$namespace" get deploy,svc,pods -l app.kubernetes.io/name=jdk17-http-demo -o wide | tee "$artifact_dir/k8s-objects.txt"
kubectl -n "$namespace" get pods -l app.kubernetes.io/name=jdk17-http-demo -o jsonpath='{range .items[*]}{.metadata.name}{" restartCount="}{.status.containerStatuses[0].restartCount}{"\n"}{end}' | tee "$artifact_dir/restart-counts-before-load.txt"

if [[ "$run_load" == "true" ]]; then
  log ""
  log "## Generate Demo Load"
  kubectl -n "$namespace" port-forward svc/jdk17-http-demo "${local_port}:8080" >"$artifact_dir/port-forward.log" 2>&1 &
  cleanup_pids+=("$!")
  sleep 3

  curl -fsS "http://127.0.0.1:${local_port}/health" | tee "$artifact_dir/health.json"
  curl -fsS "http://127.0.0.1:${local_port}/work?mode=cpu&durationMs=${duration_ms}" | tee "$artifact_dir/work-cpu.json"
  curl -fsS "http://127.0.0.1:${local_port}/work?mode=alloc&durationMs=${duration_ms}" | tee "$artifact_dir/work-alloc.json"
  curl -fsS "http://127.0.0.1:${local_port}/work?mode=lock&durationMs=${duration_ms}" | tee "$artifact_dir/work-lock.json"
  curl -fsS "http://127.0.0.1:${local_port}/threads?durationMs=${duration_ms}" | tee "$artifact_dir/threads.json"

  kubectl -n "$namespace" get pods -l app.kubernetes.io/name=jdk17-http-demo -o jsonpath='{range .items[*]}{.metadata.name}{" restartCount="}{.status.containerStatuses[0].restartCount}{"\n"}{end}' | tee "$artifact_dir/restart-counts-after-load.txt"
fi

log ""
log "## Next E2E Command"
log ""
log "KUBECONFIG=${KUBECONFIG:-/path/to/kubeconfig} scripts/real-acceptance.sh \\"
log "  --configure-profiler \\"
log "  --namespace $namespace \\"
log "  --service jdk17-http-demo \\"
log "  --require-full-profiling \\"
log "  --artifact-dir $artifact_dir/real-acceptance"
