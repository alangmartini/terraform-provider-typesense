---
name: chinook-framework
description: Author or modify e2e tests under `internal/chinooktest/` that exercise the Typesense provider against real Docker containers via the chinook example fixture. Use when adding a new test scenario, debugging a chinook test failure, extending coverage to a new Typesense version or resource type, or stress-testing generator/migrator round-trips.
---

# Chinook Framework

The chinook framework is the Typesense provider's end-to-end test suite. It lives under `internal/chinooktest/`, gated by `//go:build e2e`, and runs against real `typesense/typesense:<tag>` Docker containers. The chinook example (`examples/chinook/`) is the canonical fixture; tests materialize a per-version copy, apply via Terraform, exercise one concern, and tear the container down.

This skill is the working knowledge needed to extend the framework. Follow it; do not improvise.

## When this skill applies

- "Add an e2e test for X" / "test that Y works end-to-end".
- "The provider broke when …" and the bug needs a regression guard.
- Verifying generator output, migrator behavior, or a new Typesense version.
- Investigating drift, idempotency, or import round-trip behavior.
- Anything that touches `internal/chinooktest/`, `examples/chinook/`, or the `chinook-test` / `chinook-e2e` Make targets.

## Authoritative references

Read these once when the skill loads:

- `tasks/plan.md` — the original phased plan with acceptance criteria per scenario.
- `tasks/todo.md` — what's done, what's deferred, and *why* (especially Tier 3 skip notes).
- `docs/testing/chinook-e2e.md` — the user-facing test catalog and helper inventory.
- `docs/testing/mock-openai-protocol.md` — Phase 0 findings on what Typesense sends to LLM endpoints.
- `internal/chinooktest/harness_test.go` — `StartCluster`, `Cluster.Client()`, `freePort`, `randomSuffix`.
- `internal/chinooktest/terraform_runner_test.go` — `Terraform`, `Apply`, `Destroy`, `Plan`, `PlanWithOutput`.
- `internal/chinooktest/mock_openai_test.go` — `StartMockOpenAI`, `mock.URL` vs `mock.Local`.
- `internal/chinooktest/chinook_fixture_test.go` — `MaterializeChinook`, `chinookSkip` (per-version file dropping).
- `internal/chinooktest/version_lifecycle_test.go` — `runChinookVersion`, `expectPerCollectionSynonyms/Overrides`.

## Running

```bash
make chinook-test                          # full suite, ~5 min
make chinook-e2e RUN=TestName              # one scenario
make chinook-e2e RUN='TestVersion|Migrate' # regex filter
```

Both binaries (provider and test runner) compile to `bin/chinooktest/` so Windows Firewall only prompts once. On Windows, run `scripts/setup-windows-firewall.ps1` once as Administrator to pre-allow them.

The build tag `e2e` keeps these tests out of the default `go test ./...` run. If you forget the tag, the file you're editing will appear empty to the compiler.

## Architecture (the four pillars)

Every test composes these four helpers. Memorize them.

### 1. `StartCluster(t, version) *Cluster`

Spins up `typesense/typesense:<version>` on a free port. Returns `Cluster{Host, Port, APIKey, BaseURL, Name}` and registers `t.Cleanup` to remove the container and its data volume. Use `cluster.Client()` for typed API calls.

The container is started with `--add-host=host.docker.internal:host-gateway` so it can reach the host. On Windows the runner sets `MSYS_NO_PATHCONV=1` to stop Git Bash from rewriting Linux paths in the Docker args.

### 2. `StartMockOpenAI(t) *MockOpenAI`

In-process httptest server bound to `0.0.0.0:<free-port>` so a Typesense container can reach it via `host.docker.internal`. Two URLs:

- `mock.URL` → `http://host.docker.internal:<port>`, pass to chinook as `mock_openai_url`.
- `mock.Local` → `http://127.0.0.1:<port>`, use from the host process for assertions.

Per `docs/testing/mock-openai-protocol.md`, Typesense honors `api_url` for both `vllm/*` and `openai/*` model names. The mock currently only handles `nl_search_models` validation; `conversation_model.tf` is dropped from materialized fixtures across all versions until the mock covers that path.

### 3. `MaterializeChinook(t, version, opts) *Materialized`

Copies `examples/chinook/` to `t.TempDir()` and drops version-incompatible files via `chinookSkip(version)`:

| File | Dropped when |
|------|--------------|
| `analytics.tf` | feature `FeatureAnalyticsRules` not supported (< v28) |
| `stemming.tf` | feature `FeatureStemmingDictionaries` not supported (< v28) |
| `nl_search_model.tf` | feature `FeatureNLSearchModels` not supported (< v29) |
| `outputs.tf` | when either `analytics.tf` or `nl_search_model.tf` is dropped (its references would dangle) |
| `conversation_model.tf` | always (mock not yet supporting this validation) |
| `terraform.tfstate*`, `.terraform/`, lock files, tfvars | always (stale dev artifacts) |

Returns `Materialized{Dir, Vars}`. Pass `opts.MockOpenAIURL` to redirect `nl_search_model` validation. Use `chinookVars(cluster, m.Vars)` to assemble the full Terraform var map.

### 4. `NewTerraform(t, workDir) *Terraform`

Wraps `terraform` CLI with a per-test `.terraformrc` containing the `dev_overrides` block pointing at the provider binary. State is isolated under `<workDir>/terraform.tfstate`. Methods:

- `Apply(vars map[string]string) error`
- `Destroy(vars map[string]string) error`
- `Plan(vars map[string]string) (int, error)` — exit 0 = no changes, 2 = changes pending
- `PlanWithOutput(vars) (int, string, error)` — same as Plan but returns combined output (use this when you need to surface the diff in test failures)

The provider binary is built once per `go test` invocation by `TestMain` (in `provider_build_test.go`) and lands in `bin/chinooktest/` so the Windows Firewall rule is stable.

## How to add a new test

```go
//go:build e2e

package chinooktest

import (
	"context"
	"testing"
)

// TestXxx <single-sentence statement of what failure mode this guards against>.
func TestXxx(t *testing.T) {
	cluster := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)             // omit if the scenario doesn't touch nl_search_model
	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mock.URL})

	tf := NewTerraform(t, m.Dir)
	vars := chinookVars(cluster, m.Vars)

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// 1. Exercise the concern (mutate state, run subcommand, drift, etc.).
	// 2. Assert via cluster.Client() OR direct HTTP, not via Terraform output.
	// 3. Where the assertion is a plan diff, use tf.PlanWithOutput so the diff
	//    appears in the failure message — not just an exit code.

	if err := tf.Destroy(vars); err != nil {
		t.Fatalf("destroy: %v", err)
	}
}
```

Then:

1. Run with `make chinook-e2e RUN=TestXxx` until green.
2. Run twice in a row to confirm it isn't flaky.
3. Add a row to the table in `docs/testing/chinook-e2e.md`.
4. If the test surfaced a provider/generator/migrator bug, fix it as a *separate atomic commit* (red-green-refactor: the test is the red).

One file per scenario — that's the project convention. Don't lump scenarios.

## Common patterns (with worked references)

### Apply + verify counts (Tier 1, full chinook)

See `apply_test.go::TestApply`. Use `expectCount(t, label, want, fetcher)` for one-line cardinality assertions.

### Per-version smoke

See `version_lifecycle_test.go::runChinookVersion`. New versions become a four-line file that calls the helper:

```go
func TestVersionV31(t *testing.T) {
    runChinookVersion(t, versionScenario{
        Image: "31.0",
        Verify: func(t *testing.T, cli *client.ServerClient) { /* counts */ },
    })
}
```

Per-version expectations differ — be careful with v27 (no analytics destinations → 7 collections, not 10).

### Mutation + re-apply

See `update_test.go::TestUpdate`. Edit the materialized HCL file directly (`os.WriteFile(filepath.Join(m.Dir, "stopwords.tf"), …)`), re-apply, assert the change landed via the typed client.

### Drift recovery

See `drift_test.go::TestDrift`. Mutate state out-of-band via `cluster.Client()`, assert `tf.Plan` returns exit code 2, run `tf.Apply` to recover, verify.

### Import round-trip

See `import_roundtrip_test.go::TestImportRoundtrip`. Apply, run the `generate` subcommand into a fresh dir, drop `nl_search_models.tf` (its `api_key` is unrecoverable from the API), apply the generated config, plan, expect exit 0. The local helpers `runGenerate` and `dropImportsForResourceType` live alongside the test.

### Generator idempotency

See `generate_idempotent_test.go::TestGenerateIdempotent`. Run `generate` twice into separate dirs and compare files byte-by-byte. The only documented non-determinism is the `# Generated at:` header timestamp — `normalizeGeneratedFile` strips it. If a new non-determinism surfaces, fix it in the generator; do NOT add it to the normalizer silently.

### Migration

See `migrate_v30_test.go::TestMigrateV30`. Two clusters via two `StartCluster` calls; seed source via `seedTracks` (raw HTTP POST to `/collections/tracks/documents/import?action=upsert`); run `generate --include-data` then the `migrate --include-documents` subcommand; assert collection set, doc counts, and tracks schema fingerprint.

### Custom (non-chinook) fixture

See `escape_chars_test.go::TestEscapeChars`. When the scenario can't be expressed in chinook (special characters, intentionally broken inputs), inline the HCL as a `const` and write it via `os.WriteFile(filepath.Join(dir, "main.tf"), …, 0o600)`. Keep it minimal — one collection plus the resource(s) under test.

## Adding to / changing the chinook example itself

Chinook is the canonical fixture for *all* e2e tests. Changes there ripple. Rules:

1. **A resource that depends on a feature file's contents must live in that file.** Example: `analytics_reader` was moved into `analytics.tf` because it references collections defined there; otherwise dropping `analytics.tf` for v27 left a dangling reference. Apply the same logic for any new resource that depends on a version-gated collection or model.

2. **`outputs.tf` is fragile.** Any output that references a v28+/v29+ resource will break the whole module on v27. The materializer drops `outputs.tf` whenever `analytics.tf` or `nl_search_model.tf` is dropped — extend `chinookSkip` if you add references to other version-gated resources.

3. **Adjust expected counts in version tests when chinook grows.** v27 expects 7 collections (10 minus the 3 analytics destinations). v28+/v29+/v30 expect 10. The synonym/curation/preset/analytics counts are hard-coded in `apply_test.go` and the `version_vXX_test.go` files — search and update.

4. **Don't add documents to chinook.** It's a schema-only fixture. If a test needs documents, seed them in the test (see `migrate_v30_test.go::seedTracks`).

## Pitfalls (every one of these has bitten us)

### `omitempty` on Go structs hides user intent

`json:"…,omitempty"` drops `false`, `""`, `0`. If the user can set the field to a non-zero value AND a zero value with different semantics, you have a bug. The `CurationItem.RemoveMatchedTokens` field had to be migrated to `*bool` because Typesense's server-side default for absent `remove_matched_tokens` is `true`, so an explicit `false` was being silently overridden.

When designing or auditing a request struct, ask: *if I send the zero value, does the server interpret "absent" as the same thing?* If not, use a pointer.

### Typesense API shape changes between versions

`/analytics/rules` returns `{"rules": [...]}` on v28-v29 and a bare `[...]` on v30. `ListAnalyticsRules` decodes both. Any new client list method should be probed against multiple versions before being considered correct.

### Generated HCL must round-trip

When the generator omits a field (because it's the zero value or a default), but the schema's `Default:` populates it differently, you get permanent drift. Two fixes already in the codebase:

- `remove_matched_tokens` is emitted explicitly when `replace_query` is set (`internal/generator/hcl.go`).
- `rule.query` and `rule.match` are read as `null` instead of `""` (`internal/resources/override.go`).

When adding a new generator emitter, write a `TestImportRoundtrip`-style test for it. If the test plan reports drift after import, the generator is lying about state.

### URL path segments need escaping

Synonym IDs, override IDs, and collection names can contain spaces and slashes. Always route REST URLs through `client.serverPath(baseURL, segments...)`, which calls `url.PathEscape` on each segment. `TestEscapeChars` is the regression guard.

### Mock OpenAI binding requires `0.0.0.0`

The mock must bind to all interfaces (`net.Listen("tcp", "0.0.0.0:0")`) so the Docker container can reach it via `host.docker.internal`. Binding to `127.0.0.1` looks fine in unit tests but the container can't reach it. This is also why Windows Firewall prompts — the `0.0.0.0` listen on a fresh exe path triggers it. The stable bin path mitigates the prompt; binding to localhost would not work.

### Schema defaults vs server defaults vs user intent

Three sources of truth that must agree:

- **Schema default** (`Default: booldefault.StaticBool(true)`) — what plan resolves to when user omits the field.
- **Server default** — what the cluster stores when the request omits the field.
- **User config** — what the user wrote.

When these disagree you get drift. Audit them whenever you add a `Default:` to a resource attribute.

### v30 mutual-exclusion: `replace_query` + `remove_matched_tokens=true`

Typesense v30 rejects this combination with HTTP 400. `replace_query` + `remove_matched_tokens=false` is fine. `replace_query` alone is fine. Only the explicit `=true` combination fails. The override resource handles this by leaving the pointer nil only in that one case.

### Container start flakiness

`StartCluster` polls `/health` for up to 30 s. If a test fails with a startup error, the test prints `docker logs` to give you the cause. Common causes: port collision (rare with `freePort`), Docker Desktop not running, WSL2 VM idle-shutdown on Windows (`wsl sleep infinity &` in another terminal).

### `MSYS_NO_PATHCONV=1` is required for Docker on Git Bash

Without it, Git Bash rewrites `/data` (a Linux path) to `C:/Program Files/Git/data` (Windows nonsense) before Docker sees it. The harness sets it; if you call `docker` directly from a probe, set it yourself.

### Don't `t.Parallel()` chinook tests by default

Each test starts a container, which takes ~5 s. They already overlap I/O via Docker's daemon, so parallelism within the test process gains less than you'd think and complicates failure isolation. Stick with sequential unless you specifically measure a speedup.

## Anti-patterns

- **Mocking the Typesense client.** The whole point of this suite is that it talks to a real cluster. Mocks belong in unit tests.
- **Asserting via `terraform output`.** State assertions go through `cluster.Client()` or raw HTTP. Outputs are diagnostic only and chinook drops the file on older versions anyway.
- **Sleeping to "wait for things to happen".** If you need to wait, poll an observable condition (the cluster's response, a count, etc.) with a deadline.
- **Adding non-determinism to fix a flaky test.** If a generated file isn't byte-identical, find the source (map iteration, time.Now, default ordering) and fix it. Do not extend `normalizeGeneratedFile` to paper over it.
- **Skipping `Destroy` "because the cleanup will handle it".** `t.Cleanup` removes the container, but in a multi-step test, leaving resources behind hides bugs in the destroy path.

## Bug-discovery pattern

Every new e2e test added in this codebase has surfaced at least one provider, generator, or migrator bug. Expect this. When the test fails the first time, the failure is rarely in the test — it's in the system under test.

Workflow:

1. Write the test, run it, see it fail.
2. *Resist* tweaking the test to pass. Ask whether the failure is the system telling you something.
3. If yes: fix the bug as a separate commit *before* the test commit (or alongside, with the test as the regression guard).
4. The test then becomes a permanent guard against the bug returning.

Examples shipped via this pattern: the `remove_matched_tokens` *bool fix, the empty `rule.query` -> null fix, the generator `replace_query` emitter fix, the analytics rules wrapped-shape fix.

## Test catalog (current state)

Always update `docs/testing/chinook-e2e.md` *and* `tasks/todo.md` when you add or skip a scenario. The catalog there is authoritative; the entries below are a snapshot for quick reference, not the source of truth.

- Smoke (Phase 1): `TestHarnessSmoke`, `TestTerraformSmoke`, `TestMockOpenAISmoke`, `TestMaterializeChinook*`.
- Tier 1 (Phase 2, v30): `TestApply`, `TestUpdate`, `TestDrift`, `TestImportRoundtrip`, `TestGenerateIdempotent`.
- Tier 2 (Phase 3, per-version): `TestVersionV27`, `TestVersionV28`, `TestVersionV29`, `TestVersionV30`.
- Phase 4: `TestMigrateV30`.
- Tier 3 (Phase 5): `TestEscapeChars`. `TestConcurrentApply` and `TestMigrateV29ToV30` are deliberately deferred — see `tasks/todo.md` for reasons.

## Verification before merging a new test

- [ ] Test passes via `make chinook-e2e RUN=TestName`.
- [ ] Test passes a *second* time without state from the first run interfering.
- [ ] Test fails when the bug it guards is reintroduced (exercise this once if practical).
- [ ] Cardinality / fingerprint / count assertions are stable across runs (no maps-as-strings, no time-based comparisons).
- [ ] No new entries needed in `normalizeGeneratedFile`. If there are, justify them in the commit message.
- [ ] `docs/testing/chinook-e2e.md` and `tasks/todo.md` updated.
- [ ] If a provider/generator/migrator bug was fixed: separate atomic commit with the test as the regression guard.
