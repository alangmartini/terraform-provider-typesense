# Implementation Plan: Chinook E2E Test Suite

## Overview

Turn the chinook example into the canonical E2E fixture for the Typesense Terraform provider. Each test scenario lives in its own Go file under `internal/chinooktest/` (build tag `e2e`), spins up its own Typesense container at the version under test, exercises a single concern (apply, update, drift, import-roundtrip, generate-idempotent, migrate, version coverage, edge cases), and tears the container down. Replaces `scripts/verify-chinook-generate.sh` once the Go suite covers its assertions.

## Architecture decisions

1. **Package location:** `internal/chinooktest/`. Build tag `//go:build e2e` so `go test ./...` stays fast; CI gates the suite via `go test -tags e2e ./internal/chinooktest/...`.
2. **One test = one container.** Each test allocates a free port, starts a fresh `typesense/typesense:<version>` container, waits for `/health`, runs its scenario, and stops the container. No shared cluster across tests; isolation cost (~5s/test) is acceptable for ~15 tests.
3. **Helpers as `*_test.go` files.** Test helpers (container starter, terraform runner, mock OpenAI, fixture materializer) live in `_test.go` files in the same package so they compile only under `go test`. Each helper file holds no `Test*` functions.
4. **Terraform CLI invocation.** Tests shell out to the user's installed `terraform` binary via `os/exec`, with `TF_CLI_CONFIG_FILE` pointing at a generated `.terraformrc` that dev-overrides `alanm/typesense` to the freshly-built provider binary. State is isolated to each test's `t.TempDir()`.
5. **Provider binary for tests.** A `TestMain` builds the provider once per `go test` invocation into the temp dir; all tests share that binary.
6. **Version-aware chinook materializer.** Chinook today uses v30-only resources (curation sets, synonym sets) and v28+ stemming, v29+ NL search. The materializer copies `examples/chinook/*.tf` into the test's temp dir and filters out resources unsupported by the target version (driven by the same `version.featureVersions` map the provider already uses).
7. **Mock OpenAI server (not a real one).** For NL search and conversation models, run an in-process Go HTTP server that satisfies Typesense's validation call. Phase 0 confirmed Typesense honors `api_url` for both `vllm/*` AND `openai/*` model names — so chinook keeps its `openai/gpt-4o-mini` defaults and tests just override `api_url` to point at the mock. Protocol details in `docs/testing/mock-openai-protocol.md`: Typesense POSTs a chat-completion request to the literal `api_url` (no `/chat/completions` suffix appended).
8. **Migration tests reuse the same harness.** `migrate_*_test.go` starts two containers (source + target), apply chinook to source, run `migrate` against the source export, assert target equals source.

## Dependency graph

```
                ┌──────────────────────┐
                │  Phase 0: Decisions  │ (resolve open questions)
                └──────────────────────┘
                           │
                ┌──────────▼───────────┐
                │  Phase 1: Foundation │
                │  - container helper  │
                │  - terraform runner  │
                │  - mock OpenAI       │
                │  - chinook fixture   │
                │    materializer      │
                └──────────┬───────────┘
                           │
       ┌───────────────────┼────────────────────────┐
       ▼                   ▼                        ▼
┌────────────┐   ┌────────────────────┐   ┌────────────────┐
│  Phase 2:  │   │  Phase 3:          │   │  Phase 4:      │
│  Tier 1    │   │  Tier 2            │   │  Migration     │
│  (apply,   │   │  (per-version      │   │  (v30→v30)     │
│  update,   │   │  smoke)            │   │                │
│  drift,    │   │                    │   │                │
│  import,   │   │                    │   │                │
│  gen-idem) │   │                    │   │                │
└─────┬──────┘   └─────────┬──────────┘   └────────┬───────┘
      └────────────────────┼───────────────────────┘
                           │
                ┌──────────▼───────────┐
                │  Phase 5: Edge cases │ (optional Tier 3)
                └──────────┬───────────┘
                           │
                ┌──────────▼───────────┐
                │  Phase 6: Wire-up    │
                │  - Makefile target   │
                │  - drop verify .sh   │
                │  - README update     │
                └──────────────────────┘
```

Tasks within Phase 2/3/4 are independent of each other once the foundation is built, so they can be implemented in any order or parallelized.

---

## Phase 0: Resolve open questions (no code, ~30 min)

### Task 0.1: Verify Typesense vLLM API URL override behavior

**Description:** Confirm empirically whether Typesense honors `api_url` for `vllm/*` model names by spinning up Typesense + a logging mock server and posting an `nl_search_model` create. Required before committing to the mock-OpenAI design.

**Acceptance criteria:**
- [ ] Running `POST /nl_search_models` with `model_name: "vllm/test"` and `api_url: "http://<mock>:9999"` causes Typesense to issue a request to the mock (not to api.openai.com).
- [ ] The mock receives the validation request shape (path + method + headers + body skeleton) and we have a recording of it.

**Verification:** Manual smoke run; the mock logs the incoming request. Document the request shape in `docs/testing/mock-openai-protocol.md`.

**Dependencies:** None.

**Files likely touched:** Throwaway in `tmp/`. No commits.

**Estimated scope:** XS.

### Task 0.2: Pick concrete Docker image tags for v27, v28, v29, v30

**Description:** Verify that `typesense/typesense:27.x`, `:28.x`, `:29.x`, `:30.x` images exist on Docker Hub and pin specific tags. Some versions only have `.rcXX` releases — pick the most stable tag for each major version.

**Acceptance criteria:**
- [ ] One pinned tag per major version (e.g., 27.1, 28.0, 29.0, 30.1) listed in the plan.
- [ ] All four images pull successfully on the dev machine.

**Verification:** `docker pull typesense/typesense:<tag>` for each.

**Dependencies:** None.

**Files likely touched:** Notes in this plan.

**Estimated scope:** XS.

### Checkpoint 0
- [ ] Mock-OpenAI strategy validated.
- [ ] Image tag pins captured.
- [ ] No code committed yet.

---

## Phase 1: Foundation

### Task 1.1: Container helper (`harness_test.go`)

**Description:** Add a Go helper that starts a Typesense container at a specified version on a free port, waits for `/health`, returns a `Cluster` value with host/port/api-key/cleanup, and is safe to call concurrently across tests. Uses `os/exec` against the host's `docker` (or `wsl docker` on Windows when native docker is unavailable, matching the Makefile pattern).

**Acceptance criteria:**
- [ ] `StartCluster(t *testing.T, version string) *Cluster` starts a container, waits for health, registers `t.Cleanup` to stop+remove it.
- [ ] `Cluster` exposes `Host`, `Port`, `APIKey`, `BaseURL`, and a `Client()` method returning a `*client.ServerClient`.
- [ ] Free-port selection avoids the 8108/8109 ports used by the testbed.
- [ ] Concurrent `StartCluster` calls in the same test run do not collide.

**Verification:**
- [ ] New `harness_smoke_test.go` calls `StartCluster(t, "30.1")`, asserts `/health` returns 200, and asserts cleanup terminates the container.
- [ ] `go test -tags e2e -run TestHarnessSmoke ./internal/chinooktest/...` passes.

**Dependencies:** None.

**Files likely touched:**
- `internal/chinooktest/harness_test.go`
- `internal/chinooktest/harness_smoke_test.go`

**Estimated scope:** M.

### Task 1.2: Terraform runner helper (extend `harness_test.go`)

**Description:** Add a `Terraform` helper that wraps `os/exec` invocations of the user's `terraform` binary with isolated state, dev-override config, and a `vars` map that gets serialized as `-var=` flags. Captures stdout/stderr and returns a typed error on non-zero exit.

**Acceptance criteria:**
- [ ] `NewTerraform(t, dir)` returns a runner whose `Init`, `Apply`, `Destroy`, `Plan(...)`, and `Show(...)` methods invoke the terraform CLI with isolated state under `dir/terraform.tfstate` and a generated `.terraformrc` dev-override.
- [ ] The provider binary is built once per `go test` run via `TestMain` and reused.
- [ ] `Apply` returns an error containing the captured stderr on failure.
- [ ] On Windows, the runner finds `terraform.exe` via `PATH` or env var `TYPESENSE_E2E_TERRAFORM`.

**Verification:**
- [ ] `harness_smoke_test.go` runs `tf.Apply()` against a one-resource `.tf` file inside a Typesense container, asserts the resource exists via the client.
- [ ] `go test -tags e2e -run TestHarnessSmoke ./internal/chinooktest/...` passes.

**Dependencies:** Task 1.1.

**Files likely touched:**
- `internal/chinooktest/harness_test.go`
- `internal/chinooktest/harness_smoke_test.go`

**Estimated scope:** M.

### Task 1.3: Mock OpenAI server (`mock_openai_test.go`)

**Description:** In-process `httptest.NewServer` that responds to `POST /v1/chat/completions`, `GET /v1/models`, and any other endpoints discovered in Phase 0 with valid OpenAI-shaped JSON. Returns the server URL so tests can pass it as `api_url` to NL/conversation model resources.

**Acceptance criteria:**
- [ ] `StartMockOpenAI(t *testing.T) *MockOpenAI` returns a server with `URL` and `Requests()` (slice of recorded request paths for assertion).
- [ ] `t.Cleanup` shuts the server down.
- [ ] Bound to `0.0.0.0` (or detected host IP) so containers on the same host can reach it.

**Verification:**
- [ ] Smoke test posts a request, verifies it lands in `Requests()` and got a 200.
- [ ] Container-to-host reachability is documented (probably `host.docker.internal` on Docker Desktop, alternative for Linux).

**Dependencies:** Task 1.1 (for container-side reachability test).

**Files likely touched:**
- `internal/chinooktest/mock_openai_test.go`
- `internal/chinooktest/mock_openai_smoke_test.go`

**Estimated scope:** M.

### Task 1.4: Chinook fixture materializer (`chinook_fixture_test.go`)

**Description:** Helper that copies `examples/chinook/*.tf` and `examples/chinook/data/*.jsonl` into a per-test temp directory, then filters resources by version: drops `analytics_rule` for <v28, drops `stemming_dictionary` for <v28, drops `nl_search_model` and `conversation_model` for <v29, swaps `synonym`/`override` between per-collection (≤v29) and system-level (v30+) shapes. Also adds a `mock_openai_url` variable to the chinook example so tests can redirect NL/conversation model `api_url` at the mock without changing model names.

**Acceptance criteria:**
- [ ] `MaterializeChinook(t, version, opts) string` returns the path to a temp directory containing chinook tailored for `version`.
- [ ] Filtering is driven by the same `internal/version` feature gates the provider uses (no duplicated truth).
- [ ] When `opts.MockOpenAIURL` is set, NL/conversation model resources are pointed at it via `var.mock_openai_url` and `var.nl_model_name = "vllm/mock"` etc.

**Verification:**
- [ ] Unit-style helper test: materialize for v27, assert no analytics/stemming/nl/conversation resources remain in the output.
- [ ] Materialize for v30, assert all resources present.

**Dependencies:** None (pure file I/O + parsing; can be developed in parallel with 1.1–1.3).

**Files likely touched:**
- `internal/chinooktest/chinook_fixture_test.go`
- `internal/chinooktest/chinook_fixture_helper_test.go`

**Estimated scope:** M.

### Checkpoint 1
- [ ] All four helpers compile under the `e2e` build tag.
- [ ] Helper smoke tests pass.
- [ ] `go vet ./internal/chinooktest/...` clean.
- [ ] Plan reviewed before scenario tests start.

---

## Phase 2: Tier 1 vertical slices (v30 only)

Each task here is a single test file in its own slice. They are independent and can be parallelized once Phase 1 is done.

### Task 2.1: `apply_test.go` — TestApply

**Description:** Spin up Typesense v30, materialize chinook + mock OpenAI, run `terraform init && apply`, assert via the client that every chinook resource exists with the expected attributes.

**Acceptance criteria:**
- [ ] `terraform apply` exits 0.
- [ ] Client assertions confirm: 11 collections, 6 aliases, 3 stopword sets, 12 presets, 3 analytics rules, 1 stemming dict, 20 synonym set items, 9 curation set items, ≥3 API keys, 1 NL search model, 1 conversation model.
- [ ] `terraform destroy` cleans the cluster (zero collections after).

**Verification:** `go test -tags e2e -run TestApply ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/apply_test.go`.

**Estimated scope:** S.

### Task 2.2: `update_test.go` — TestUpdate

**Description:** Apply chinook, mutate a value in the materialized `.tf` (e.g., change a preset's `value`, add a stopword), re-apply, assert the mutation is reflected via the client.

**Acceptance criteria:**
- [ ] First apply succeeds.
- [ ] After mutation, second apply reports a non-empty plan (≥1 update).
- [ ] After second apply, client assertions confirm the new value(s).

**Verification:** `go test -tags e2e -run TestUpdate ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/update_test.go`.

**Estimated scope:** S.

### Task 2.3: `drift_test.go` — TestDrift

**Description:** Apply chinook, mutate state directly via the client (e.g., delete a stopword set), run `terraform plan`, assert the plan reports drift, run `apply`, assert state is restored.

**Acceptance criteria:**
- [ ] After client-side mutation, `terraform plan` exit code = 2 (changes pending).
- [ ] Apply restores the resource.
- [ ] Final client state matches initial state.

**Verification:** `go test -tags e2e -run TestDrift ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/drift_test.go`.

**Estimated scope:** S.

### Task 2.4: `import_roundtrip_test.go` — TestImportRoundtrip

**Description:** Apply chinook, run `generate --output=<tmp>` against the cluster, then run `terraform init && plan` against the generated dir (with `imports.tf`) using a fresh state. Assert the plan reports zero changes (or only no-op imports).

**Acceptance criteria:**
- [ ] `generate` produces `*.tf` + `imports.tf` + `data/*.jsonl`.
- [ ] `terraform plan` against the generated config + a clean state file produces a plan with zero `+`/`-`/`~` after the imports execute.

**Verification:** `go test -tags e2e -run TestImportRoundtrip ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/import_roundtrip_test.go`.

**Estimated scope:** M.

### Task 2.5: `generate_idempotent_test.go` — TestGenerateIdempotent

**Description:** Apply chinook, run `generate` twice into different output directories, assert the two outputs are byte-identical (after normalizing for any timestamps or non-deterministic IDs).

**Acceptance criteria:**
- [ ] Two `generate` runs produce identical files (modulo a documented normalization function).
- [ ] If non-determinism is found, it's named in a follow-up bug, not papered over silently.

**Verification:** `go test -tags e2e -run TestGenerateIdempotent ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/generate_idempotent_test.go`.

**Estimated scope:** S.

### Checkpoint 2
- [ ] All Tier 1 tests pass.
- [ ] Each test runs in <30s end-to-end on the dev machine.
- [ ] No flakiness in 3 consecutive runs.

---

## Phase 3: Tier 2 — per-version smoke

Single table-driven test or one file per version (the user asked for one file per scenario, so one file per version).

### Task 3.1: `version_v27_test.go` — TestVersionV27

**Description:** Spin up `typesense/typesense:27.x`, materialize chinook subset for v27 (drops analytics, stemming, NL/conversation models, swaps synonym/override to per-collection shape), apply, assert resources, destroy.

**Acceptance criteria:**
- [ ] Apply succeeds with no version-gated errors.
- [ ] Client confirms presence of v27-supported resources (collections, aliases, presets, stopwords, per-collection synonyms, per-collection overrides, API keys).

**Verification:** `go test -tags e2e -run TestVersionV27 ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/version_v27_test.go`.

**Estimated scope:** S.

### Task 3.2: `version_v28_test.go` — TestVersionV28

**Description:** Same shape as 3.1, target v28, includes analytics rules + stemming dictionaries.

**Acceptance criteria:** As 3.1 plus analytics rules and stemming dictionary present.

**Verification:** `go test -tags e2e -run TestVersionV28 ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/version_v28_test.go`.

**Estimated scope:** S.

### Task 3.3: `version_v29_test.go` — TestVersionV29

**Description:** Target v29, includes NL search model (with mock OpenAI). Still per-collection synonyms/overrides (sets are v30+).

**Acceptance criteria:** As 3.2 plus NL search model present and resolvable.

**Verification:** `go test -tags e2e -run TestVersionV29 ./internal/chinooktest/...`.

**Dependencies:** Phase 1, plus mock OpenAI server (Task 1.3).

**Files likely touched:** `internal/chinooktest/version_v29_test.go`.

**Estimated scope:** S.

### Task 3.4: `version_v30_test.go` — TestVersionV30

**Description:** Target v30 (full chinook). Largely overlaps Task 2.1 but exists for symmetry; if redundant, mark as a thin wrapper that calls the same scenario function.

**Acceptance criteria:** As 2.1.

**Verification:** `go test -tags e2e -run TestVersionV30 ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/version_v30_test.go`.

**Estimated scope:** XS.

### Checkpoint 3
- [ ] All four version tests pass.
- [ ] Total Tier 1+2 wall-clock <5 min on the dev machine.

---

## Phase 4: Migration

### Task 4.1: `migrate_v30_test.go` — TestMigrateV30

**Description:** Start two v30 containers (source + target). Apply chinook to source. Run `generate --include-data` against source, then `migrate` against target. Assert target collections + documents match source.

**Acceptance criteria:**
- [ ] Source apply succeeds.
- [ ] `migrate` exits 0.
- [ ] For each chinook collection, target `num_documents` equals source `num_documents` (within fixture row counts).
- [ ] Schema fingerprint (sorted field list) matches between source and target for each collection.

**Verification:** `go test -tags e2e -run TestMigrateV30 ./internal/chinooktest/...`.

**Dependencies:** Phase 1.

**Files likely touched:** `internal/chinooktest/migrate_v30_test.go`.

**Estimated scope:** M.

### Checkpoint 4
- [ ] Migration test passes.
- [ ] Migration time on the dev machine is captured (informational — sets baseline for future regressions).

---

## Phase 5 (optional): Tier 3 edge cases

### Task 5.1: `concurrent_apply_test.go` — TestConcurrentApply

**Description:** Apply a chinook variant that creates synonyms/overrides under a single collection in parallel via Terraform's default 10-resource concurrency. Asserts no items are silently dropped (the regression the synonym/override mutex prevents).

**Acceptance criteria:**
- [ ] Apply succeeds.
- [ ] Item counts match the `.tf` (no dropped items).
- [ ] Test fails if the mutex is removed (regression guard).

**Dependencies:** Phase 1.
**Estimated scope:** S.

### Task 5.2: `escape_chars_test.go` — TestEscapeChars

**Description:** Create a collection whose name contains a space and a slash, plus synonyms/overrides with spaces in IDs. Assert lifecycle works end-to-end (regression guard for the recent `url.PathEscape` work).

**Acceptance criteria:**
- [ ] Apply, update, destroy all succeed.
- [ ] Direct API calls confirm the resource exists at the escaped path.

**Dependencies:** Phase 1.
**Estimated scope:** S.

### Task 5.3: `migrate_v29_to_v30_test.go` — TestMigrateV29ToV30

**Description:** Source = v29 with per-collection synonyms/overrides. Target = v30. Migrate. Assert synonyms land in `synonym_sets`, overrides in `curation_sets`.

**Acceptance criteria:**
- [ ] Migration succeeds.
- [ ] Target has system-level synonym sets and curation sets matching source per-collection content.

**Dependencies:** Phase 1, depends on migrator support for cross-version translation (verify support exists; if not, this task becomes a feature task on migrator first).

**Estimated scope:** M (or L if migrator changes are needed — would split).

---

## Phase 6: Wire-up and cleanup

### Task 6.1: `make chinook-test` switches to `go test -tags e2e`

**Description:** Update the Makefile target to invoke the Go E2E suite instead of the bash verifier. Keep `make start-typesense` for ad-hoc local use.

**Acceptance criteria:**
- [ ] `make chinook-test` runs `go test -tags e2e -timeout 30m ./internal/chinooktest/...` and reports pass/fail.
- [ ] No script orchestration (no bash verifier called).

**Dependencies:** Phases 2 and 4 complete.

**Files likely touched:** `Makefile`.

**Estimated scope:** XS.

### Task 6.2: Remove `scripts/verify-chinook-generate.sh`

**Description:** Delete the bash verifier and any references to it.

**Acceptance criteria:**
- [ ] File deleted.
- [ ] No grep hit for `verify-chinook-generate` in the repo.
- [ ] `make chinook-test` still passes.

**Dependencies:** Task 6.1.

**Files likely touched:** `scripts/verify-chinook-generate.sh` (deleted), maybe `Makefile`, `README.md`.

**Estimated scope:** XS.

### Task 6.3: README + `docs/testing/` update

**Description:** Document how to run the E2E suite, what each test covers, and how to add a new scenario.

**Acceptance criteria:**
- [ ] `README.md` "Testing" section mentions `make chinook-test` and `go test -tags e2e`.
- [ ] `docs/testing/chinook-e2e.md` lists every test scenario, what it asserts, and the steps to add one.

**Dependencies:** All prior phases complete.

**Files likely touched:** `README.md`, `docs/testing/chinook-e2e.md`.

**Estimated scope:** S.

### Checkpoint 6
- [ ] `make chinook-test` is the single command for E2E.
- [ ] `verify-chinook-generate.sh` is gone.
- [ ] Docs reflect the new flow.

---

## Risks and mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Typesense doesn't honor `api_url` for `vllm/*` model names | High — blocks Tier 1+2 NL/conversation tests | Phase 0 Task 0.1 verifies before any code; if it fails, fall back to mocking via DNS override or skipping NL/conversation in v29/v30 unless `TEST_OPENAI_API_KEY` is set. |
| Container startup flakiness (Docker on Windows/WSL) | Medium — flaky tests | `StartCluster` polls health for up to 30s with backoff; failure surfaces container logs in the test failure. |
| `host.docker.internal` not resolvable from container on Linux dev machines | Medium — mock OpenAI unreachable | Detect Linux at runtime, add `--add-host=host.docker.internal:host-gateway`, document the requirement. |
| Old Typesense images (v27, v28) have surprising API differences | Medium — version tests fail | Phase 0 Task 0.2 confirms images exist; per-version task reveals divergences early; documented in `version_vXX_test.go` if any divergence is acceptable. |
| `terraform` binary not on PATH in CI | Medium — CI fails | `TYPESENSE_E2E_TERRAFORM` env var to override path; CI installs terraform before running tests. |
| Per-test container cost (~5s) inflates total time | Low — slower CI | Cap at ~15 tests; total ~80s of pure container time + ~3 min Terraform work. Acceptable. |
| Migrator doesn't currently translate per-collection synonyms → set-level | Variable — Task 5.3 may grow | Investigate during Phase 0 only if Task 5.3 is in-scope; otherwise defer to a separate effort. |

## Open questions for human review

1. **Tier 3 in-scope?** I've planned Tier 3 as optional (Phase 5). Confirm whether to include in this round or defer.
2. **CI integration?** Should `make chinook-test` run on every PR, or only nightly? E2E suites are typically slow/flaky; nightly might be safer initially.
3. **Mock OpenAI: process or container?** Plan currently uses an in-process `httptest.NewServer` reachable via `host.docker.internal`. Alternative: ship the mock as a tiny container. In-process is simpler; container is more isolated. I lean in-process.
4. **Build tag name.** Currently `e2e`. Could also use `chinooke2e` or `acceptance` to avoid collisions if other E2E suites get added later. I lean `e2e`.

