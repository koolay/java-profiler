# Localization

English is the original version of the `java-profiler` documentation. Chinese pages cover the paths most users and contributors need first.

## Required bilingual pages

Keep these pages available in both English and Chinese:

| English | Chinese |
| --- | --- |
| `/` | `/zh/` |
| `/getting-started/quickstart` | `/zh/getting-started/quickstart` |
| `/operations/performance-analysis-user-manual` | `/zh/operations/performance-analysis-user-manual` |
| `/contributing/development` | `/zh/contributing/development` |
| `/reference/profiling-contracts` | `/zh/reference/profiling-contracts` |

Operational pages with a localized full manual:

| English entry point | Chinese full manual |
| --- | --- |
| `/operations/deployment-operations-admin-manual` | `/zh/operations/deployment-operations-admin-manual` |
| `/operations/e2e-automation-test-guide` | `/zh/operations/e2e-automation-test-guide` |

## English-only pages

Keep implementation-heavy or low-traffic material English-only unless there is a clear user need:

- Architecture details.
- Ingestion architecture review.
- The detailed E2E automation procedure is localized under `/zh/operations`; the English route remains an overview and acceptance boundary.
- Real profiling acceptance standard.
- Research notes.
- Brainstorms and project-history material.

Do not put research, brainstorms, or plans in the public navigation.

## Translation workflow

When changing a required bilingual page:

1. Update the English page first.
2. Update the matching Chinese page in the same change.
3. Keep the same route shape under `/zh/`.
4. Reuse screenshots unless the image itself contains language-specific UI text that makes the page confusing.
5. Run the docs build before publishing.

```bash
cd docs
npm run docs:build
```

## Style

- Prefer clear technical Chinese over literal translation.
- Keep product names, API keys, annotation names, profile types, and file paths unchanged.
- Use English terms when they are the terms users see in the UI, for example `Top Table`, `Flame Graph`, `Self CPU`, and `Total CPU`.
- Avoid adding new product claims only to the Chinese page. If the claim matters, add it to English first.
