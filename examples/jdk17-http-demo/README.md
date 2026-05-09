# JDK 17 HTTP Demo

Small Java HTTP service for profiling simulations. It uses only JDK APIs and is intended as a Kubernetes target for the node-local Java profiler collector.

## Endpoints

```text
GET /health
GET /work?mode=cpu&durationMs=1000
GET /work?mode=alloc&durationMs=1000
GET /work?mode=lock&durationMs=1000
GET /threads?durationMs=5000
```

`/work` creates CPU, allocation, or monitor-lock activity for async-profiler testing. `/threads` creates short-lived sleeping and blocked threads for thread snapshot checks.

## Run Locally

```bash
mvn test
mvn package
PORT=8080 java -jar target/jdk17-http-demo-0.1.0.jar
curl http://localhost:8080/health
curl "http://localhost:8080/work?mode=cpu&durationMs=3000"
```

## Build Container

The GitHub workflow `.github/workflows/profile-demo-image.yml` builds and publishes the demo image:

```text
ghcr.io/koolay/java-profiler-jdk17-http-demo:latest
ghcr.io/koolay/java-profiler-jdk17-http-demo:sha-<commit>
```

Deploy the published image:

```bash
scripts/deploy-jdk17-demo.sh --image ghcr.io/koolay/java-profiler-jdk17-http-demo:latest --run-load
```

Build locally when iterating:

```bash
docker build -t jdk17-http-demo:local examples/jdk17-http-demo
scripts/deploy-jdk17-demo.sh --image jdk17-http-demo:local --build-image --load-to-node --run-load
```

Or run the helper script:

```bash
scripts/deploy-jdk17-demo.sh --build-image --run-load
```

The Kubernetes manifest opts the pod into temporary profiling with project contract metadata:

```text
java-profiler.io/profile-mode=temporary
java-profiler.io/profile-duration=1h
java-profiler.io/startup-delay=0s
java-profiler.io/snapshot-interval=10s
```
