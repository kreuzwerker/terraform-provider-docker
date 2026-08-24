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
  copilot-requests: write 
engine: copilot
tools:
  github:
    mode: gh-proxy
    toolsets: [repos, issues]
  cache-memory: true
  web-fetch:
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

Work through these four stages in order. Do not collapse them into an
immediate code-change suggestion.

### Stage 1 — Read version history (no code analysis)

Read the Docker Engine API version history at
<https://docs.docker.com/reference/api/engine/version-history/>. Use
`web-fetch` first and Playwright CLI only if browser rendering or navigation
is required. Before looking at repository code, extract documented changes into
structured JSON such as:

```json
{
  "api_version": "1.51",
  "changes": [
    {
      "type": "request_field",
      "endpoint": "POST /containers/create",
      "field": "HostConfig.CgroupnsMode",
      "source_url": "https://docs.docker.com/reference/api/engine/version-history/"
    }
  ]
}
```

Include additions, removals, and deprecations, with API version, change type,
endpoint, field or operation, and source URL. Do not infer a change that is
not documented, and explicitly record changes that are outside the provider's
scope. This stage must contain no Terraform resource or source-code mapping.

### Stage 2 — Find relevant Terraform resources

Now inspect the repository's provider registration and establish an
API-to-provider map. Start with `internal/provider/provider.go`, then inspect
the registered resources, data sources, and actions. Use confirmed mappings
such as:

```text
POST /containers/create  -> docker_container
GET /containers/json     -> docker_container data source (if registered)
POST /networks/create    -> docker_network
```

Search the Docker SDK usage and provider code to verify each relationship.
If a mapping is not readily derivable, maintain it as structured YAML or JSON
in the report (do not add a repository file) and mark it `needs-review`; never
guess. Include endpoint, provider kind/name, and the evidence file/symbol.

### Stage 3 — Inspect implementation coverage

For every relevant mapped API change, inspect the actual implementation under
`internal/provider/`. This is a Go provider using the Docker SDK dependency
declared in `go.mod` and both Terraform Plugin SDK/framework APIs. Read the
current dependency versions from `go.mod` rather than assuming a fixed SDK
version. For each resource, data source, or action, check:

- schema declaration and the exact Terraform attribute type/name;
- expand or request-building functions, including nested `HostConfig` data;
- flatten or response-reading functions;
- CRUD or action handlers and Docker SDK calls;
- unit/acceptance tests and test helper coverage;
- generated documentation under `docs/` and its source templates.

For each missing piece, report a separate item such as `schema`, `expand`,
`flatten`, `handler`, `test`, or `docs`. For example, determine whether
`docker_container` exposes `HostConfig.CgroupnsMode`, rather than assuming
that it does.

### Stage 4 — Produce an implementation plan

Compare the history JSON, verified provider map, and implementation coverage.
Check existing open issues before reporting duplicate work. Use the
repository's existing `enhancement` label and issue conventions. Produce a
maintainer-ready report (not code) for each actionable change, for example:

```text
Docker API 1.51
New attribute: HostConfig.CgroupnsMode
Terraform resource: docker_container
Changes required:
- [ ] schema: internal/provider/resource_docker_container.go
- [ ] expand: relevant HostConfig request builder
- [ ] flatten: relevant response reader, if applicable
- [ ] acceptance test: matching internal/provider/*_test.go
- [ ] documentation: generated docs and source template
Estimated complexity: Low
```

Estimate complexity as Low, Medium, or High and explain the estimate. Include
additions, removals, deprecations, unmapped items, risks, exact file paths and
symbols, recommended tests/docs, source links, and uncertainty notes.
Consider the provider's current surface: containers, images, networks,
volumes, configs, secrets, services, plugins, tags, Buildx builders, data
sources, and actions.

Maintain a compact JSON snapshot of the source URL, retrieval date, API
changes, provider mappings, and comparison watermark in
`/tmp/gh-aw/cache-memory/`. Reuse it on later runs to highlight newly changed
history, but refresh the source and verify cached conclusions each run.
Timestamped filenames must use `YYYY-MM-DD-HH-MM-SS` with no colons, `T`, or
`Z`.

## Boundaries

- DO NOT analyze provider code during Stage 1; keep history extraction
  independent and auditable.
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
