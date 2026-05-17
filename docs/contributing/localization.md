# Localization

English is the source language for `java-profiler` documentation. Chinese pages are localized copies for the paths most users and contributors need first.

## Required bilingual pages

Keep these pages available in both English and Chinese:

| English | Chinese |
| --- | --- |
| `/` | `/zh/` |
| `/getting-started/quickstart` | `/zh/getting-started/quickstart` |
| `/operations/performance-analysis-user-manual` | `/zh/operations/performance-analysis-user-manual` |
| `/contributing/development` | `/zh/contributing/development` |
| `/reference/profiling-contracts` | `/zh/reference/profiling-contracts` |

## English-only pages

Keep implementation-heavy or low-traffic material English-only unless there is a clear user need:

- Architecture details.
- Ingestion architecture review.
- E2E automation details.
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
