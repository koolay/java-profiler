---
title: Fix Flamegraph Code Readability
status: active
created: 2026-05-10
origin: user request
---

# Fix Flamegraph Code Readability

## Problem Frame

The real Kubernetes profiling acceptance path now produces valid Java CPU flamegraph data for the JDK HTTP demo, but the UI does not make code frames easy enough to inspect. Narrow bars truncate long Java method names, dense stacks can overlap visually, and the user needs to clearly see the profiled code segment, not just prove that the application ran without errors.

## Scope Boundaries

- Keep this focused on the existing web flamegraph UI and real acceptance verification.
- Do not change the profiling ingestion API or backend data model.
- Do not add a new backend dependency or expand into non-Java profiling.
- Keep the Kubernetes real test path based on the existing JDK17 HTTP demo service and collector setup.

## Requirements

- R1: The CPU flamegraph UI must make long Java frame names such as `DemoHttpService.burnCpu:188` readable after real profiling data is ingested.
- R2: The UI must expose a selected-frame detail area with full frame text, sample value, and percent of the current rendered root so users can inspect code segments without relying on hover-only tooltips.
- R3: Search and zoom interactions must keep working with the improved presentation.
- R4: Automated tests must cover the readability behavior.
- R5: Real E2E validation must confirm the Kubernetes demo still produces real `DemoHttpService` stacks and that the UI screenshot visibly contains readable code frames.

## Existing Patterns

- Flamegraph rendering lives in `web/src/visualization/flamegraph.tsx`.
- Flamegraph unit coverage lives in `web/src/visualization/flamegraph.test.tsx`.
- Shared UI styles live in `web/src/styles.css`.
- Real browser acceptance lives in `web/tests/real-acceptance.spec.ts`.
- Kubernetes real test deployment uses `scripts/deploy-jdk17-demo.sh`.

## Implementation Units

### U1: Add Inspectable Flamegraph Selection

Files:

- Modify: `web/src/visualization/flamegraph.tsx`
- Modify: `web/src/styles.css`
- Test: `web/src/visualization/flamegraph.test.tsx`

Approach:

- Track a selected frame path separately from zoom path.
- Select a frame when clicked and keep double-click or a small explicit control for zoom, so inspection and navigation do not fight each other.
- Render a frame detail panel below the flamegraph containing the full frame name, raw value, percent of current root, and depth.
- Use monospace styling and `overflow-wrap: anywhere` for full Java method paths.

Test Scenarios:

- Renders the selected frame detail with a long Java method name after clicking a frame.
- Keeps title attributes for hover access.
- Search still marks matching frames.
- Reset clears zoom back to root without losing basic frame rendering.

Verification:

- `cd web && npm test -- --run flamegraph`

### U2: Improve Flamegraph Label Readability

Files:

- Modify: `web/src/styles.css`
- Test: `web/src/visualization/flamegraph.test.tsx`

Approach:

- Increase row height slightly and stabilize frame layout dimensions.
- Use darker text on lighter frame backgrounds or stronger contrast per depth so text remains readable in screenshots.
- Ensure narrow frames hide inline text instead of overlapping neighboring frames, while full text remains available in the detail panel.
- Keep mobile layout from overflowing incoherently.

Test Scenarios:

- Very long labels remain present in accessible text/title and are inspectable in the detail panel.
- Tiny frames get the compact class and do not rely on visible inline text.

Verification:

- `cd web && npm test -- --run flamegraph`

### U3: Real Acceptance Re-Test

Files:

- Existing test: `web/tests/real-acceptance.spec.ts`
- Existing script: `scripts/deploy-jdk17-demo.sh`

Approach:

- Run unit tests and web build/test checks.
- Use `export KUBECONFIG=$HOME/backup/localk8s.yaml`.
- Re-run real Kubernetes profiling against `jdk17-http-demo`.
- Query backend flamegraph data for `DemoHttpService` stack frames.
- Run Playwright real acceptance and inspect screenshot evidence for readable code frame text.
- Iterate U1/U2 if the screenshot still fails readability.

Test Scenarios:

- Backend flamegraph has `root.value > 0`.
- Backend flamegraph includes `DemoHttpService.burnCpu` or `DemoHttpService.handleWork`.
- Demo Pod restart count remains zero.
- UI screenshot clearly exposes the selected/full frame code segment.

Verification:

- `go test ./collector/internal/profiler ./collector/runtime`
- `helm lint deploy/helm -f deploy/helm/values_test.yaml`
- `cd web && npm test -- --run flamegraph`
- `cd web && npm run build`
- `REAL_ACCEPTANCE=1 ... npx playwright test --config=playwright.config.ts tests/real-acceptance.spec.ts --reporter=line`

## Risks

- Real cluster port-forwarding may fail intermittently on kubelet connectivity; if so, keep the backend query and UI evidence from successful port-forwards as the acceptance record.
- Visual readability is screenshot-sensitive; acceptance should inspect actual screenshot output, not only DOM assertions.
