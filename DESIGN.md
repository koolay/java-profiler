# Design System - Java Profiler

## Product Context

- **What this is:** A Java-focused production profiling workbench for Kubernetes services. It helps engineers inspect real async-profiler evidence from a selected Java process during production performance incidents.
- **Who it's for:** Senior Java developers, SREs, platform engineers, and incident responders handling P0/P1 CPU, allocation, lock, and JVM performance problems.
- **Space/industry:** Kubernetes Java performance profiling and incident diagnostics.
- **Project type:** Data-dense operational web app, not a marketing site or general observability dashboard.

## Aesthetic Direction

- **Direction:** Industrial / Utilitarian / Forensic.
- **Decoration level:** Minimal.
- **Mood:** Serious, precise, and evidence-first. The UI should feel like a production incident workbench, not a dashboard built for screenshots.
- **Reference products:** Datadog Continuous Profiler, Dynatrace CPU profiling, Grafana Pyroscope. These are references for profiler interaction patterns only; this project must keep its Java/Kubernetes scope and avoid becoming a general observability suite.

## Core Design Principles

1. **Evidence before decoration.** Freshness, drop rate, sampling frequency, Pod/JVM scope, and CPU quota baseline must be visible near the data they qualify.
2. **Java semantics stay visible.** Use JVM, HotSpot, JIT, allocation, lock, stack frame, and method-signature language directly. Do not flatten the product into a generic multi-language profiler.
3. **MVP stays narrow.** The first UI should optimize one expert path: choose a Java Pod, inspect CPU profile, find top methods, select a frame, copy/share the evidence.
4. **Numbers must carry units.** Avoid raw sample counts in primary UI. Convert samples into time, cores, percentages with explicit baseline, or rates.
5. **Noise is optional.** Native/system frames must be hideable. Expert users should be able to return visual focus to Java application frames quickly.
6. **Collaboration is part of incident response.** Share, copy stack, and permalink actions are core workbench actions, not polish.

## MVP Screen Scope

The initial UI should be a **single Java Pod CPU profile view**.

### Include in MVP

- Top context bar with Namespace, Service, Pod, time window, and evidence health.
- Explicit evidence health: freshness lag, drop rate, sampling frequency, and collection status.
- CPU profile type as the primary view.
- Flame Graph and Top Methods views, with a combined "Both" mode.
- Self CPU and Total CPU with time/cores conversion.
- Selected Frame detail drawer with FQCN, method signature, line number when available, Self/Total, baseline, JIT status when known, and stack path.
- Hide Native/System Frames toggle.
- Search by class, method, and line number.
- Copy Stack, Share/Permalink, Focus, Back, and Reset actions.
- Light and dark mode compatibility.

### Defer from MVP

- A/B Comparison as a primary panel.
- Wall Clock, GC, and I/O detail views.
- JVM event timeline correlation.
- Multi-Pod service rollup.
- Release/version comparison.
- AI-generated interpretation blocks.
- Code viewer integration beyond copy/permalink actions.

Deferred features may appear as disabled navigation items or roadmap notes only when doing so does not distract from the core CPU profile workflow.

## Future Evidence Views

These features are part of the product direction, but should not expand the first UI implementation unless the active plan explicitly includes them.

- **A/B Comparison:** Compare equivalent evidence across two contexts, such as normal Pod versus anomalous Pod, baseline time window versus incident window, or release A versus release B. The first comparison mode should work without release metadata by comparing two time windows.
- **Wall Clock:** Analyze runnable plus blocked time for Java services where request latency is not explained by CPU.
- **GC:** Show GC pause, allocation pressure, and JVM event evidence correlated to the selected profile window.
- **I/O:** Show network and disk blocking time when supported by the collected evidence.
- **Service Rollup:** Show Pod variance and outlier detection, then let users drill down to a single Java Pod.

## Typography

- **UI / Body:** IBM Plex Sans. It is readable, neutral, technical, and works well in dense operational interfaces.
- **Data / Tables:** IBM Plex Mono with tabular numbers. Use for CPU cores, percentages, timestamps, Pod names, sample rates, and profile IDs.
- **Code / Stack Frames:** JetBrains Mono. Use for method names, stack paths, FQCNs, and snippets.
- **Loading strategy:** Prefer self-hosted fonts in production. Google Fonts or Bunny Fonts are acceptable for prototypes only.

## Type Scale

- **Page title:** 20px / 28px, 600.
- **Panel title:** 14px / 20px, 600.
- **Body:** 13px / 20px, 400.
- **Table body:** 12px / 18px, 400.
- **Labels / headers:** 11px / 16px, 700, uppercase only for compact labels.
- **Stack frames:** 11px / 16px, JetBrains Mono.
- **Metric values:** 18-20px / 24px, IBM Plex Mono, 600.

Do not scale font size with viewport width. Keep letter spacing at `0` except compact uppercase labels, where `0.04em` is acceptable.

## Color

- **Approach:** Restrained semantic palette. Color exists to communicate evidence type, state, and severity.
- **Background:** `#F7F8FA`
- **Surface:** `#FFFFFF`
- **Surface muted:** `#EEF1F4`
- **Border:** `#D5DAE1`
- **Strong border:** `#B6BEC8`
- **Text:** `#172026`
- **Muted text:** `#5D6975`
- **Primary:** `#0F766E`
- **CPU:** `#C2410C`
- **Wall Clock:** `#0F766E`
- **Allocation:** `#2563EB`
- **GC:** `#7C2D12`
- **I/O:** `#6D5D00`
- **Lock:** `#854D0E`
- **Success:** `#15803D`
- **Warning:** `#B45309`
- **Error:** `#B42318`
- **Info:** `#2563EB`

## Color Usage Rules

- Use strong semantic colors for labels, icons, active borders, legends, and selected states.
- Use low-saturation or translucent variants for large filled areas such as flame graph frames.
- Flame graph fill examples:
  - CPU: `rgba(194, 65, 12, 0.12-0.28)`
  - Wall Clock: `rgba(15, 118, 110, 0.12-0.24)`
  - Allocation: `rgba(37, 99, 235, 0.10-0.22)`
  - GC: `rgba(124, 45, 18, 0.10-0.22)`
  - I/O: `rgba(109, 93, 0, 0.10-0.22)`
- Do not use gradients, decorative color blobs, or large saturated background bands.
- Validate contrast for every foreground/background pair used in table rows, flame frames, tooltips, badges, and controls.

## Dark Mode

Dark mode should be redesigned, not merely inverted.

- **Background:** `#101214`
- **Surface:** `#171B1F`
- **Surface muted:** `#20262B`
- **Border:** `#2B3238`
- **Strong border:** `#3B454D`
- **Text:** `#E7EAEE`
- **Muted text:** `#A4ACB5`

Reduce large-area saturation in dark mode. Keep selected states legible without glowing effects.

## Spacing

- **Base unit:** 4px.
- **Density:** Compact.
- **Scale:** 2px, 4px, 8px, 12px, 16px, 24px, 32px, 48px, 64px.
- **Table rows:** 34px default.
- **Toolbar controls:** 32px height.
- **Tabs:** 26-28px height.
- **Panel padding:** 10-14px.
- **Page padding:** 14-16px.

Dense does not mean cramped. Preserve enough row height for scanning long method names and numeric columns without vertical jitter.

## Layout

- **Approach:** Grid-disciplined workbench.
- **Primary shell:** top context bar, left evidence/scope navigation, central profile workspace, right selected-frame detail drawer.
- **MVP layout:** three columns on desktop: left navigation around 180-200px, central workspace fluid, detail drawer around 300-340px.
- **Responsive behavior:** collapse the detail drawer below the main workspace on medium screens, and stack all regions on narrow screens.
- **Max content width:** None for the workbench. Use full available width because flame graphs and tables need horizontal space.
- **Border radius:** 4px for small controls, 6px for buttons/chips, 8px maximum for panels and cards, `9999px` only for compact badges.

Avoid nested cards and decorative card mosaics. Use panels only for actual work surfaces: flame graph, table, selected frame, event detail, or controls.

## Interaction

- **Search:** `Cmd/Ctrl + K` opens class/method search.
- **Flame graph traversal:** Arrow keys move between frames, `Enter` focuses, `Esc` backs out or resets.
- **Table sorting:** Column headers must sort, especially Self CPU and Total CPU.
- **Tooltips:** Hovering a method/frame should show rich detail: FQCN, method signature, line number, Self/Total, converted time/cores, baseline, JIT/inlining state when known, and copy/focus actions.
- **Share:** Permalinks should preserve namespace, service, Pod, time range, profile type, search term, hide-native state, and focused frame when possible.
- **Copy Stack:** Must produce a complete stack path suitable for incident notes, Jira, Slack, or ticket systems.

## Motion

- **Approach:** Minimal-functional.
- **Durations:** 80-100ms for micro feedback, 150-200ms for drawer/tab transitions, 240ms maximum for larger layout changes.
- **Easing:** ease-out for entering, ease-in for exiting, ease-in-out for movement.
- **Avoid:** Decorative motion, looping effects, animated backgrounds, or refresh animations that make data appear unstable.

## Accessibility

- Maintain keyboard parity for search, focus, back, reset, and table sorting.
- Keep focus rings visible and consistent.
- Do not encode state by color alone. Pair color with text, icon, or badge labels.
- Use zebra striping or subtle row separators for dense tables.
- Keep method names accessible via tooltip or expandable detail when truncated.

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-05-17 | Adopt Industrial / Utilitarian / Forensic direction | Production profiling is an incident-response workflow; precision and trust matter more than decorative appeal. |
| 2026-05-17 | Scope MVP to single Java Pod CPU profile | This preserves the core expert workflow while avoiding premature complexity from A/B diff, event correlation, and multi-Pod aggregation. |
| 2026-05-17 | Use IBM Plex Sans, IBM Plex Mono, and JetBrains Mono | The product is data-heavy and Java-code-heavy; these fonts support dense scanning and developer trust. |
| 2026-05-17 | Use low-saturation flame graph fills | Large saturated profiling blocks reduce readability and can break contrast; strong colors are reserved for labels, legends, and selected states. |
