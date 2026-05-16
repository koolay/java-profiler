---
date: 2026-05-16
topic: tag-triggered-release-pipeline
---

# Tag-Triggered Release Pipeline

## Summary

Creating a release tag should automatically trigger CI to build the current repository state and publish a release for that tag. The release should include the repository's release-grade artifacts so the tag corresponds to something a user can actually install or consume, not just a version label.

---

## Problem Frame

Today the repository already has CI for pushes and pull requests, plus a separate image-oriented workflow, but there is no dedicated tag-driven release path. That leaves release creation as a manual or implicit step instead of a repeatable part of the delivery process.

The gap matters because a release tag is usually the point where maintainers want to say, "this commit is shippable." If tag creation does not also produce the distributable outputs and a release record, consumers have to guess which commit or artifact set is authoritative.

---

## Actors

- A1. Maintainer / release engineer: creates the release tag and expects the project to publish a usable release from it.
- A2. CI system: detects the tag, runs the release pipeline, and publishes the release if validation passes.
- A3. Release consumer: downloads the published release artifacts or uses the tag page as the source of truth for a versioned delivery.

---

## Key Flows

- F1. Tag creation to release publication
  - **Trigger:** A maintainer pushes a release tag.
  - **Actors:** A1, A2
  - **Steps:** CI recognizes the tag, builds the repository from that tagged commit, runs the release checks, and publishes the release only if the pipeline succeeds.
  - **Outcome:** A release exists for that tag and represents the tagged source state.

- F2. Consumer retrieves a release
  - **Trigger:** A consumer opens the release page for a published tag.
  - **Actors:** A3
  - **Steps:** The consumer finds the release, downloads the relevant artifact, and uses the tag as the version reference.
  - **Outcome:** The tag points to a concrete, consumable delivery.

- F3. Failed release attempt
  - **Trigger:** The release pipeline fails during build or validation.
  - **Actors:** A2
  - **Steps:** CI stops the publication flow and surfaces the failure.
  - **Outcome:** No release is published for an invalid build.

---

## Requirements

**Triggering and release identity**
- R1. The system must start a release pipeline automatically when a release tag is pushed.
- R2. The pipeline must treat the tagged commit as the source of truth for everything published under that release.
- R3. The release must be identifiable by both the tag and the source commit that produced it.
- R4. A release run must not publish a release if the build or validation steps fail.

**Release outputs**
- R5. The release must publish the repository's release-grade artifacts for that tag.
- R6. The published artifacts must be the concrete deliverables a consumer would use to install, run, or consume the version.
- R7. A repeated run for the same tag must not create a second independent release entry.

**Operator experience**
- R8. Normal release creation should be automatic after tag push, without requiring a separate manual dispatch step.
- R9. The release experience should make it obvious which artifacts belong to which tag.

---

## Acceptance Examples

- AE1. **Covers R1, R2, R3, R5, R8, R9.** Given a maintainer pushes a valid release tag, when the pipeline completes successfully, then a release is published for that tag and the release page exposes the tag, the source commit, and the release-grade artifacts.
- AE2. **Covers R4.** Given a release tag is pushed but build or validation fails, when the pipeline ends, then no release is published for that tag.
- AE3. **Covers R7.** Given the same release tag is processed again, when the pipeline reruns, then it does not create a duplicate release entry for that tag.

---

## Success Criteria

- Maintainers can cut a release by creating a tag, without a separate manual packaging step.
- Consumers can reliably use the tag as the authoritative version marker and find the corresponding deliverables from the release.
- The handoff to planning is clear enough that ce-plan does not need to invent trigger conditions, release identity, or the basic release shape.

---

## Scope Boundaries

- Release tags should trigger the release pipeline automatically; ad hoc manual release creation is not the v1 path.
- The release should cover only the repository's shippable artifacts, not docs, test fixtures, or unrelated build outputs.
- Changelog generation, release-note curation, and semantic version policy are not required to solve the core release trigger problem.
- Release signing, SBOM generation, provenance metadata, and other supply-chain extras are optional enhancements unless explicitly added to the scope later.
- Multi-channel promotion, staged rollout, and broader release orchestration are out of scope for this first pass.

---

## Key Decisions

- Tag push is the release trigger: this matches the desired operator workflow and keeps releases tied to an explicit repository state.
- The release should be more than a tag marker: it must include the artifacts a consumer can actually use.
- The release should fail closed: if the pipeline does not validate the tag's build, nothing is published.

---

## Dependencies / Assumptions

- The repository already has CI infrastructure capable of building the current workspace from a tagged commit.
- The exact artifact set is assumed to come from the repository's existing release-grade outputs; planning will confirm the final list.
- A tag naming convention or tag eligibility rule will be needed if the repository wants to distinguish release tags from ordinary tags.
- The publishing workflow has permission to create and update releases in the repository hosting platform.

---

## Outstanding Questions

### Resolve Before Planning

- What tag pattern or release rule should qualify a tag for automatic release publication?
- Which artifacts are the canonical v1 release deliverables for this repository?

### Deferred to Planning

- Should release notes be generated automatically or authored separately?
- Should supply-chain extras like signing, SBOMs, or provenance be part of the first release cut or a later enhancement?
