# Contributing

The fastest way to break confidence in this project is to make the UI look healthy while the collector produces no real data. Changes to the profiling path therefore need real-data acceptance, not only unit tests or a page-load check.

## Development setup

Run commands from the repository root.

```bash
go test ./...
javac --release 11 java-helper/thread-diagnostics/src/main/java/com/ebpfjava/threads/*.java
cd examples/jdk17-http-demo && mvn test
cd ../../web && npm ci && npm test && npm run build
```

## Docs site

```bash
cd docs
npm install
npm run docs:dev
```

Build before publishing documentation changes:

```bash
cd docs
npm run docs:build
```

The docs site is bilingual. English is the original version, and Chinese covers the core user and contributor paths. Read [Localization](./localization.md) before adding or moving a public page.

## Real acceptance

Use real Kubernetes acceptance for changes touching collector profiling, ingestion, ClickHouse storage, backend query APIs, deployment, the demo service, or the profile UI.

```bash
export KUBECONFIG=$HOME/backup/localk8s.yaml

scripts/real-acceptance.sh \
  --service jdk17-http-demo \
  --configure-profiler \
  --require-full-profiling \
  --high-volume \
  --artifact-dir /tmp/java-profiler-real-acceptance-$(date +%Y%m%d%H%M%S)
```

The run is successful only when it produces an accepted target, non-empty CPU/allocation/lock profiles, ClickHouse rows, ingestion information, bounded retention, browser evidence, and no increase in the target workload's restart count.

## Screenshot evidence

Docs screenshots should come from a real UI connected to a real backend.

```bash
export REAL_ACCEPTANCE_BASE_URL=http://127.0.0.1:18081
export REAL_ACCEPTANCE_NAMESPACE=java-profiler-qa
export REAL_ACCEPTANCE_SERVICE=jdk17-http-demo
node scripts/capture-doc-screenshots.mjs
```
