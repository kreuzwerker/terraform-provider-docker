---
name: 1-history-parser-read-the-docker-api-ver
description: >-
  Compare Docker Engine API version history with the Terraform Docker provider
  and create a concrete maintenance issue for API additions, removals, or
  deprecations.
on:
  schedule:
    - cron: "0 9 * * 1-5"
permissions:
  contents: read
  issues: read
engine: copilot
tools:
  github:
    mode: gh-proxy
    toolsets: [repos, issues]
  cache-memory: true
  web-fetch: true
  playwright:
    mode: cli
safe-outputs:
  create-issue:
    labels:
      - enhancement
timeout-minutes: 30
network:
  allowed:
    - defaults
    - "docs.docker.com"
    - playwright
---

# Docker API history and provider compatibility planner

Run as a scheduled maintenance analysis for
`kreuzwerker/terraform-provider-docker`.

## Task

1. Read the Docker Engine API version history at
   <https://docs.docker.com/reference/api/engine/version-history/>. Use
   `web-fetch` first and Playwright CLI if browser rendering or navigation is
   required. Extract every addition, removal, and deprecation relevant to the
   provider into structured JSON. Preserve the API version, endpoint or
   field name, change type, and source URL; do not infer a change that is not
   documented.
2. Inspect this checked-out repository. It is a Go Terraform provider using
   Go 1.25.8, the Docker SDK, and the Terraform Plugin SDK/framework. Focus on
   `internal/provider/`, especially `provider.go`, `resource_*.go`,
   `data_source_*.go`, `action_*.go`, and their `*_funcs.go` companions.
   Map each API element to matching resources, data sources, actions, schema
   attributes, Docker SDK calls, CRUD handlers, tests, and generated docs.
   Include relevant file paths and symbol names, and distinguish confirmed
   mappings from unmapped items.
3. Compare the JSON history with the provider map. Consider the current
   provider's supported surface (containers, images, networks, volumes,
   configs, secrets, services, plugins, tags, Buildx builders, data sources,
   and actions), and identify actionable compatibility gaps, obsolete code, or
   documentation/test updates.
4. Check existing open issues before reporting duplicate work. Use the
   repository's existing `enhancement` label and follow its issue conventions:
   describe affected resources/data sources, concrete references, and
   reproducible validation or follow-up steps.

Maintain a compact JSON snapshot of the source URL, retrieval date, API
changes, provider mappings, and comparison watermark in
`/tmp/gh-aw/cache-memory/`. Reuse it on later runs to highlight newly changed
history, but refresh the source and verify cached conclusions each run.
Timestamped filenames must use `YYYY-MM-DD-HH-MM-SS` with no colons, `T`, or
`Z`.

## Boundaries

- DO NOT modify repository files, commit, push, open a pull request, or run
  arbitrary release or deployment commands.
- DO NOT create an issue for an unchanged comparison, an unverifiable claim,
  or a duplicate of an existing issue.
- DO NOT treat Docker CLI behavior as Docker Engine API history without a
  documented source.
- DO NOT guess resource, data source, schema, handler, SDK, or documentation
  mappings; mark uncertain relationships as `unmapped` or `needs-review`.
- DO NOT include credentials, tokens, private data, or large copied source
  excerpts in the issue.
- Use only the configured safe output for a visible mutation. The only
  permitted visible output is one new GitHub issue; do not use shell `gh`
  commands to create or edit GitHub content.
- If no new, actionable, non-duplicate compatibility work is found, call
  `noop` with a short explanation.

## Issue output

When actionable work exists, create one issue with a concise title and a
meaningful body containing:

- a summary and retrieval date;
- the structured JSON change set (or a compact, complete representation);
- a table/checklist mapping API changes to exact provider files and symbols;
- separate sections for additions, removals, deprecations, unmapped items,
  risks, and recommended tests/docs;
- source links and explicit uncertainty notes.

The issue must be useful to a maintainer without rerunning this workflow.
