# Contributing

This project needs contributors to preserve one rule: profiling changes must prove real profile data, not just a healthy UI.

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

Build before publishing docs changes:

```bash
cd docs
npm run docs:build
```

The docs site is bilingual. English is the source language, and Chinese covers the core user and contributor paths. See [Localization](./localization.md) before adding or moving public docs pages.

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

Passing means the run produced accepted target status, non-empty CPU/allocation/lock profiles, ClickHouse rows, ingestion evidence, bounded retention, browser UI evidence, and no target workload restart increase.

## Screenshot evidence

Docs screenshots should come from a real UI connected to a real backend.

```bash
export REAL_ACCEPTANCE_BASE_URL=http://127.0.0.1:18081
export REAL_ACCEPTANCE_NAMESPACE=java-profiler-qa
export REAL_ACCEPTANCE_SERVICE=jdk17-http-demo
node scripts/capture-doc-screenshots.mjs
```
