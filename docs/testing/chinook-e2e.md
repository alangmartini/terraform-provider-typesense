# Chinook end-to-end test suite

The `internal/chinooktest/` package exercises the provider against real
Typesense containers via the chinook example. Each test owns its
container lifecycle, so the suite is hermetic and parallel-friendly.

## Running

```bash
make chinook-test                            # Full suite, ~6 min wall-clock
make chinook-e2e RUN=TestApply               # Single scenario
make chinook-e2e RUN='TestVersionV3.|Migrate' # Regex filter
```

`make chinook-e2e` pre-compiles the test binary to
`bin/chinooktest/chinooktest.test.exe` (and `terraform-provider-typesense.exe`
in the same directory) so Windows Firewall remembers the allow rule
across runs. To pre-allow both binaries non-interactively:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\setup-windows-firewall.ps1
```

The build tag `e2e` keeps these tests out of the default `go test ./...`
run.

## Prerequisites

- Docker (Docker Desktop / WSL2 / Linux native).
- `terraform` on `PATH`, or `TYPESENSE_E2E_TERRAFORM` pointing at the binary.
- Go toolchain.

The suite builds the provider binary once per `go test` invocation and
uses Terraform CLI dev-overrides so no `terraform init` round-trip is
needed. The mock OpenAI server runs in-process and binds to `0.0.0.0`;
the cluster reaches it via `host.docker.internal` (Docker Desktop) or
the host-gateway alias (Linux).

## Scenarios

### Tier 1 — vertical slices (v30)

| File | Test | What it asserts |
|------|------|-----------------|
| `apply_test.go` | `TestApply` | Full chinook apply, expected resource cardinality on cluster, clean destroy. |
| `update_test.go` | `TestUpdate` | Mutating an existing resource (stopwords list) and re-applying. |
| `drift_test.go` | `TestDrift` | Server-side deletion produces a non-zero plan; apply restores. |
| `import_roundtrip_test.go` | `TestImportRoundtrip` | apply -> generate -> reapply against fresh state -> plan reports zero changes. |
| `generate_idempotent_test.go` | `TestGenerateIdempotent` | Two `generate` runs produce byte-identical output (modulo timestamp). |

### Tier 2 — per-version smoke

`version_v27_test.go` through `version_v30_test.go` apply the
materialized chinook against the matching Typesense image. The
materializer drops feature files unsupported by the target version
(`analytics.tf`, `stemming.tf`, `nl_search_model.tf`,
`conversation_model.tf`, `outputs.tf` when its references break).

### Phase 4 — migration

`migrate_v30_test.go::TestMigrateV30` stands up two v30 clusters,
applies chinook to source, seeds a few documents, runs
`generate --include-data` then `migrate --include-documents`, and
asserts collection set, doc counts, and the tracks schema fingerprint
match across clusters.

## Helpers

- `StartCluster(t, version)` — runs a Typesense Docker container, waits
  for `/health`, registers cleanup. Returns a `*Cluster` with a
  configured `Client()`.
- `StartMockOpenAI(t)` — in-process OpenAI-compatible server reachable
  from containers via `host.docker.internal`.
- `MaterializeChinook(t, version, opts)` — copies `examples/chinook` to
  a temp dir, drops version-incompatible files, returns
  `*Materialized{Dir, Vars}`.
- `NewTerraform(t, workDir)` — wraps Terraform CLI invocation with a
  per-test `.terraformrc` containing the dev override.

## Adding a new scenario

1. Create `internal/chinooktest/<name>_test.go` with `//go:build e2e`.
2. Use `StartCluster`, `StartMockOpenAI`, `MaterializeChinook`,
   `NewTerraform`, and the existing `expectCount` / `runChinookVersion`
   helpers wherever they fit.
3. Run `make chinook-e2e RUN=YourTestName` until green and stable.
4. Add a row to the table above.

If the scenario needs a fixture beyond chinook, prefer extending the
chinook example (it stays the canonical source of truth) over creating
a parallel fixture, unless the new scenario is specifically about
behavior chinook cannot express (escape characters in collection names,
deliberately broken inputs, etc.).
