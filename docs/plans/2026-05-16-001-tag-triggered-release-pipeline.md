---
title: "feat: Tag-triggered release pipeline"
type: feat
status: active
date: 2026-05-16
origin: docs/brainstorms/2026-05-16-tag-triggered-release-pipeline.md
---

# feat: Tag-triggered release pipeline

## Summary

Add an automated release path that runs when a release tag is pushed, builds the tagged revision, publishes the product's versioned artifacts, and creates a GitHub Release for that tag. The release boundary is the product deliverables, not the demo workflow: backend, collector, web, and the Helm install package.

## Problem Frame

The repository already validates pushes and pull requests, and it already has a separate demo-image workflow, but it does not yet have a canonical tag-driven release path. That means a release tag does not currently guarantee that the matching product artifacts were built and published together.

This plan closes that gap so a tag can act as the authoritative handoff from source state to consumable release state. The result should be a release that is easy for maintainers to cut and easy for consumers to trust.

## Requirements Traceability

- R1-R4 from `docs/brainstorms/2026-05-16-tag-triggered-release-pipeline.md`: tag push triggers the pipeline, the tagged commit is the source of truth, the release is tied to one tag/commit pair, and failed builds do not publish.
- R5-R7: publish the release-grade artifacts, keep them attached to the right tag, and avoid duplicate release entries for the same tag.
- R8-R9: release creation should be automatic for tag pushes and the release page should make artifact/tag relationships obvious.

## Context & Research

- `.github/workflows/ci.yml` already runs Go, Java helper, web, and Helm validation on push and pull request events.
- `.github/workflows/profile-demo-image.yml` is a demo-specific workflow with manual dispatch support and should remain separate from the product release path.
- `deploy/helm/Chart.yaml` and `deploy/helm/values.yaml` currently pin version and image tags to `0.1.0`, so release packaging needs a version-alignment strategy instead of assuming those defaults are already release-shaped.
- `deploy/helm/templates/` is the current install surface for the product.
- `scripts/build-real-acceptance-images.sh` and `scripts/deploy-jdk17-demo.sh` are verification and demo helpers, not the canonical product release path.

## Key Technical Decisions

- Use a dedicated tag-triggered workflow rather than expanding the existing PR/main CI workflow into a release publisher.
- Treat the product's GHCR images and Helm package as the canonical release outputs.
- Keep the demo image workflow out of the product release path.
- Fail closed: if validation fails, nothing is published for that tag.
- Make release versioning explicit enough that the chart package and image tags stay aligned with the release tag.

## Implementation Units

### U1. Add the tag-triggered release workflow

Files:

- `.github/workflows/release.yml`

Scope:

- Trigger on release-intent tags only.
- Run the minimum validation required before publication.
- Publish nothing when the tagged build fails.
- Prevent duplicate publication for the same tag.

Validation scenarios:

- A valid release tag triggers the workflow and reaches publication.
- A non-tag push does not create a release.
- A failing build or validation step does not publish a release.
- Reprocessing the same tag does not create a second release entry.

### U2. Package and publish the product artifacts with tag-aligned versioning

Files:

- `.github/workflows/release.yml`
- `deploy/helm/Chart.yaml`
- `deploy/helm/values.yaml`
- `deploy/helm/templates/backend-deployment.yaml`
- `deploy/helm/templates/collector-daemonset.yaml`
- `deploy/helm/templates/web-deployment.yaml`

Scope:

- Build and publish the backend, collector, and web images for the release tag.
- Package the Helm chart so the published package is version-aligned with the tag.
- Keep the chart and image references consistent so consumers get one coherent release set.
- Leave the demo image out of the product release output.

Validation scenarios:

- The published images carry the release tag, not a stale development tag.
- The packaged chart advertises the same release identity as the tag.
- A chart install path resolves to the release-aligned image references.
- The product release output does not include the demo image.

### U3. Make the release path discoverable for maintainers

Files:

- `docs/index.md`
- `docs/operations/deployment-operations-admin-manual.md`

Scope:

- Document the tag-driven release path where maintainers already look for operational guidance.
- Add the new plan to the docs index so the release work is easy to find later.

Validation scenarios:

- A maintainer reading the operational docs can tell how the tag release path is supposed to work.
- The documentation index points to the release plan.

## Dependencies / Assumptions

- Release tags will follow a release-intent naming convention, likely `vX.Y.Z`, rather than arbitrary tags.
- GitHub Actions will have permission to push images to GHCR and create GitHub Releases.
- The product release will include backend, collector, and web images plus the Helm chart package; the demo workflow remains separate.
- Supply-chain extras such as signing, SBOMs, and provenance are optional hardening items unless we decide to make them part of v1.

## Open Questions

### Deferred to Implementation

- Should the workflow generate release notes automatically, or should maintainers write the release notes separately?
- Should provenance/SBOM generation ship in the first release workflow, or follow as a later hardening pass?
