# Chinook E2E Test Suite — Todo

Flat checklist tracking implementation progress. Source of truth is `tasks/plan.md`.

## Phase 0: Decisions
- [x] 0.1 Verify Typesense honors `api_url` for `vllm/*` model names — PASS (also works for `openai/*`); see `docs/testing/mock-openai-protocol.md`
- [x] 0.2 Pin Docker image tags for v27, v28, v29, v30 — `27.1`, `28.0`, `29.0`, `30.1`
- [x] Checkpoint 0

## Phase 1: Foundation
- [x] 1.1 Container helper (`harness_test.go`) + `harness_smoke_test.go`
- [x] 1.2 Terraform runner helper — split into `provider_build_test.go` (TestMain + binary build) and `terraform_runner_test.go` (Apply/Destroy/Plan)
- [ ] 1.3 Mock OpenAI server (`mock_openai_test.go`) + smoke test
- [ ] 1.4 Chinook fixture materializer (`chinook_fixture_test.go`)
- [ ] Checkpoint 1: helpers green, plan reviewed

## Phase 2: Tier 1 vertical slices (v30)
- [ ] 2.1 `apply_test.go` — TestApply
- [ ] 2.2 `update_test.go` — TestUpdate
- [ ] 2.3 `drift_test.go` — TestDrift
- [ ] 2.4 `import_roundtrip_test.go` — TestImportRoundtrip
- [ ] 2.5 `generate_idempotent_test.go` — TestGenerateIdempotent
- [ ] Checkpoint 2: Tier 1 stable across 3 runs

## Phase 3: Tier 2 — per-version smoke
- [ ] 3.1 `version_v27_test.go` — TestVersionV27
- [ ] 3.2 `version_v28_test.go` — TestVersionV28
- [ ] 3.3 `version_v29_test.go` — TestVersionV29
- [ ] 3.4 `version_v30_test.go` — TestVersionV30
- [ ] Checkpoint 3: total wall-clock <5 min on dev machine

## Phase 4: Migration
- [ ] 4.1 `migrate_v30_test.go` — TestMigrateV30
- [ ] Checkpoint 4: migration timing baseline captured

## Phase 5: Edge cases (optional)
- [ ] 5.1 `concurrent_apply_test.go` — TestConcurrentApply
- [ ] 5.2 `escape_chars_test.go` — TestEscapeChars
- [ ] 5.3 `migrate_v29_to_v30_test.go` — TestMigrateV29ToV30 (only if migrator already supports translation)

## Phase 6: Wire-up and cleanup
- [ ] 6.1 `make chinook-test` runs `go test -tags e2e`
- [ ] 6.2 Remove `scripts/verify-chinook-generate.sh`
- [ ] 6.3 Update `README.md` + add `docs/testing/chinook-e2e.md`
- [ ] Checkpoint 6: single-command E2E, .sh gone, docs current

## Decisions still pending (block start)
- [ ] Tier 3 in-scope or deferred?
- [ ] CI: every PR vs. nightly?
- [ ] Mock OpenAI: in-process vs. container?
- [ ] Build tag name (`e2e` vs. alternative)?
