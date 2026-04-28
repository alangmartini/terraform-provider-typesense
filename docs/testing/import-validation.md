# Import / Resync Validation Test Suite

Manual acceptance tests for the `generate` command (imports.tf HCL workflow). Each test states a goal, the steps to execute, and the expected outcome. Run against a real Typesense Cloud cluster.

## Test Environment

| Item | Value |
|------|-------|
| Branch under test | `feat/imports-tf-hcl` |
| Provider binary | `./terraform-provider-typesense.exe` (built at repo root) |
| Terraform | `>= 1.5` (HCL import blocks required) |
| Cloud cluster | Test cluster in Typesense Cloud (clone, safe to mutate) |
| Cloud management API key | Short-lived test key (rotate after run) |
| Server admin API key | Read from cluster admin console |
| Work directory | `tmp/validation/<test-name>/` (all generated output) |

Credentials never committed. Provide via env vars at run time:

```bash
export TYPESENSE_CLOUD_MANAGEMENT_API_KEY="..."
export TEST_CLUSTER_HOST="xyz.a1.typesense.net"
export TEST_CLUSTER_API_KEY="..."
export TEST_CLUSTER_ID="xyz"
```

## Preconditions (run once)

1. Build the provider binary.
   ```bash
   go build -o terraform-provider-typesense.exe .
   ```
2. Confirm `~/.terraformrc` has a `dev_overrides` entry for `alanm/typesense` pointing at the repo root.
3. Create an empty scratch directory.
   ```bash
   mkdir -p tmp/validation
   ```

---

## Test 1: Import from cluster produces local files for migration

**Goal.** Given a live source cluster, `generate` must write self-contained HCL plus `imports.tf` on disk so the operator can move them to a target environment.

**Steps.**

1. Run generator in schema-only mode.
   ```bash
   ./terraform-provider-typesense.exe generate \
     --host="$TEST_CLUSTER_HOST" --port=443 --protocol=https \
     --api-key="$TEST_CLUSTER_API_KEY" \
     --cloud-api-key="$TYPESENSE_CLOUD_MANAGEMENT_API_KEY" \
     --output=./tmp/validation/test1-import
   ```
2. List the output directory.
3. Open `imports.tf` and confirm it contains HCL `import { to = ...; id = ... }` blocks.
4. Open `main.tf` and confirm `terraform.required_providers` plus a `provider "typesense"` stub.
5. Open at least one per-resource file (e.g. `collections.tf`) and confirm resource bodies were generated.

**Expected.**

- Exit status 0.
- Files written: `main.tf`, `imports.tf`, `cluster.tf` (because `--cloud-api-key` was passed), and one or more resource-type files (`collections.tf`, `api_keys.tf`, etc.).
- `imports.tf` is valid HCL and lists every imported resource.
- No secrets leak into any file (API key placeholders only).

---

## Test 2: Imported state works as infra-as-code

**Goal.** Operator can take the generated directory, point Terraform at the same cluster, and end up with a clean plan (no diff). This is the "adopt existing cluster into IaC" flow.

**Steps.**

1. `cd tmp/validation/test1-import` (reuse output from Test 1).
2. Replace the API-key placeholder in `main.tf` with a variable reference or the real key via `TF_VAR_*` (never commit).
3. Because `dev_overrides` is in effect, skip `terraform init`.
4. Apply to perform the import.
   ```bash
   terraform apply -auto-approve
   ```
5. Plan against the same cluster.
   ```bash
   terraform plan -detailed-exitcode
   ```
6. Delete `imports.tf` as recommended by README.
7. Re-plan to confirm it still reports no changes.

**Expected.**

- `terraform apply` reports "N imported" and zero creations/updates.
- `terraform plan` exits with code `0` (no changes, no errors). Exit code `2` means drift and is a failure.
- After removing `imports.tf`, plan still exits `0`.

---

## Test 3: A single local change appears as a single diff

**Goal.** After adoption, a one-line edit to the HCL must surface as exactly one resource change in plan. Confirms that the generated file captured every attribute Typesense returns by default — otherwise spurious diffs would appear.

**Steps.**

1. Copy Test 2's directory so the run is isolated.
   ```bash
   cp -r tmp/validation/test1-import tmp/validation/test3-single-change
   cd tmp/validation/test3-single-change
   ```
2. Pick a low-risk field. Options in order of preference:
   - Edit an `typesense_override` rule's `includes` block (add one pinned doc).
   - Rename a synonym's `synonyms` list (add one term).
   - Change a preset's `value` JSON (toggle a per_page default).
3. Run `terraform plan -detailed-exitcode`.
4. Inspect the diff output.
5. Revert the edit and plan again.

**Expected.**

- Plan shows **exactly one** resource with `~ update in-place` and **zero** other changes.
- Exit code `2` (changes pending) on the modified plan, `0` after revert.
- No unrelated `forces replacement`, `-/+`, or computed-attribute churn.

Failure modes that indicate a bug in the generator, not in the test:
- Multiple resources show as drifting after a single edit.
- Computed fields (e.g. `created_at`, server-default fields) show as pending changes.

---

## Test 4: Re-import (resync) picks up out-of-band changes

**Goal.** After a cluster has already been adopted into Terraform, mutations made via the Typesense API (outside Terraform) must be re-importable by re-running `generate` and merging the result — without clobbering the existing setup.

**Steps.**

1. Start from Test 2's working directory with a clean plan.
2. Make an out-of-band mutation against the live cluster using the server API directly (NOT Terraform). Example:
   ```bash
   curl -X POST "https://$TEST_CLUSTER_HOST/collections/products/synonyms/test-resync-1" \
     -H "X-TYPESENSE-API-KEY: $TEST_CLUSTER_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"synonyms": ["alpha", "bravo"]}'
   ```
3. Run `terraform plan` — this should show drift (the new synonym is absent from state).
4. Re-run `generate` into a new output directory.
   ```bash
   ./terraform-provider-typesense.exe generate \
     --host="$TEST_CLUSTER_HOST" --port=443 --protocol=https \
     --api-key="$TEST_CLUSTER_API_KEY" \
     --cloud-api-key="$TYPESENSE_CLOUD_MANAGEMENT_API_KEY" \
     --output=./tmp/validation/test4-resync
   ```
5. Diff the new `imports.tf` and resource files against the originals.
6. Merge the new synonym's HCL block and `import` block into the original project.
7. Run `terraform apply` then `terraform plan`.

**Expected.**

- Step 3 plan shows drift for the new out-of-band resource (proves drift is detected, not silently swallowed).
- Step 5 diff shows the new synonym is present in the fresh generation.
- After merge + apply, plan is clean.

Known limitation to verify: today there is **no automatic merge** — the README documents `terraform refresh/plan/apply` as the resync path, with `generate` used only for first-time adoption or as a source for hand-merging. Confirm whether re-running `generate` against an existing output directory overwrites or merges.

---

## Test 5: Generator scopes to the configured cluster only

**Goal.** When the operator passes `--host` plus `--cloud-api-key` for an account that owns multiple clusters, `generate` must export resources only from the matching cluster. A cross-cluster leak here is a data-segmentation bug.

**Steps.**

1. Confirm the cloud management key's account owns at least one cluster besides the test cluster. (If not, temporarily create a throwaway second cluster — or skip to the code-level verification below.)
2. Run generator with `--host` pointing at the test cluster.
   ```bash
   ./terraform-provider-typesense.exe generate \
     --host="$TEST_CLUSTER_HOST" --port=443 --protocol=https \
     --api-key="$TEST_CLUSTER_API_KEY" \
     --cloud-api-key="$TYPESENSE_CLOUD_MANAGEMENT_API_KEY" \
     --output=./tmp/validation/test5-filter
   ```
3. Inspect `cluster.tf`.
4. Count resources and compare against the cluster's actual inventory via the admin UI.
5. Also run generator WITHOUT `--host` (cloud-only mode) into a separate directory and confirm it writes resources for all clusters in the account.

**Expected.**

- `cluster.tf` from step 2 contains exactly one `typesense_cluster` resource, matching `$TEST_CLUSTER_ID`.
- Server resources (collections, synonyms, etc.) reflect the one cluster that `--host` pointed at.
- Step 5 demonstrates the difference: cloud-only mode exports all clusters (expected, documented).

Code anchor: `internal/generator/generator.go:336` (`generateClusters`) filters clusters by hostname match when `g.config.Host != ""`; falls back to fingerprint probing at `internal/generator/generator.go:375`; writes a warning and exports all clusters if both fail at `internal/generator/generator.go:400`.

---

## Test 6: Automated test feasibility check

**Goal.** Determine whether the above tests can be expressed as automated tests that run in CI or locally, and identify what infrastructure is needed.

**Steps.**

1. Run the existing generator unit tests.
   ```bash
   go test ./internal/generator/...
   ```
2. Read `internal/generator/generator_test.go` and `internal/generator/imports_test.go` to see what's covered.
3. Inspect the `testbed/` directory for existing end-to-end harness.
4. Record which of Tests 1-5 could be automated with:
   - The local Typesense container (covers Tests 1, 2, 3 — no cloud required).
   - The Typesense Cloud Management API (needed for Tests 4 when re-import touches cluster-level resources, and Test 5 for multi-cluster filtering).
5. Identify missing pieces (e.g. test helpers for running `terraform apply` against a temp directory, cleanup between runs).

**Expected output.** A short list of "automatable now" vs "needs new harness" vs "manual only", with file paths for where new automation should live.

---

## Cleanup

```bash
rm -rf tmp/validation
```

If any test modified the cloud cluster (Test 4's out-of-band synonym, Test 5's throwaway cluster), delete those mutations before ending the run. Rotate the cloud management API key and server admin key after the session.

## Run Log: 2026-04-15

| Test | Result | Notes |
|------|--------|-------|
| 1. Import produces local files | PASS | 36 import blocks across 8 files (`main.tf`, `cluster.tf`, `collections.tf`, `api_keys.tf`, `stopwords.tf`, `synonyms.tf`, `analytics.tf`, `imports.tf`). Exit 0. |
| 2. Imported state as IaC | PASS* | `terraform apply` imported 34 resources cleanly. Subsequent `terraform plan` reports "No changes." *2 analytics rules excluded due to bug (see below). |
| 3. Single change, single diff | PASS | Added `"validation-probe"` to `typesense_stopwords_set.test2`. Plan showed exactly 1 `~ update in-place` with only the `stopwords` field changed. Exit 2. |
| 4. Re-import resync | PASS | Out-of-band `tf-validation-resync` stopwords set created via direct API. Re-running `generate` into a fresh dir produced a new `imports.tf` containing the new resource. Merged into existing project; subsequent apply imported 1 more resource; plan was clean. |
| 5. Single-cluster filter | PASS | Account has 3 clusters; `cluster.tf` from Test 1 contained exactly 1 (`jl8ka6qtdc5gx4f0p`), matching `--host`. Code path: `internal/generator/generator.go:336` scopes to matching cluster when `--host` is set. |
| 6. Automated test feasibility | PARTIAL | 19 unit tests exist in `internal/generator/` covering cluster matching, import block generation, and per-resource import-ID formats. No end-to-end test that runs `generate → terraform apply → plan` against a real cluster. See recommendations below. |

### Bug found during Test 2

**Symptom.** `terraform apply` failed to import `typesense_analytics_rule.docs_popular_queries` and `typesense_analytics_rule.docs_no_hits` with "Cannot import non-existent remote object."

**Root cause.** The rule names contain spaces and a literal `/` (e.g. `docs / popular queries`). `ServerClient.GetAnalyticsRule` at `internal/client/server_client.go:966` builds the request URL with `fmt.Sprintf("%s/analytics/rules/%s", c.baseURL, name)`, passing the raw name without URL-encoding. The server returns 404 for the malformed path. Verified: `GET /analytics/rules/docs%20%2F%20popular%20queries` returns 200 with the correct rule.

**Same issue elsewhere.** `CreateAnalyticsRule` at `:879` and `DeleteAnalyticsRule` at `:999` build URLs the same way. Likely affects every resource whose import ID can contain spaces or slashes (analytics rules, synonyms with certain characters, etc.).

**Suggested fix.** Wrap path segments with `url.PathEscape(name)` before substituting into the URL.

### Automated test recommendations (Test 6)

**Already automated (unit tests):**
- Cluster hostname matching with multi-cluster fixtures: `internal/generator/generator_test.go` (`TestClusterMatchesHost`, `TestFindClustersByServerProbe`). Covers Test 5 at the unit level.
- Import block HCL emission: `internal/generator/imports_test.go` (`TestGenerateImportBlocks`, per-resource `Test*ImportID`). Covers Test 1's HCL shape.
- `terraform validate` against generated HCL: `internal/generator/terraform_validate_test.go`. Covers HCL syntactic correctness.

**Gap: end-to-end generate → apply → plan loop.** The testbed at `testbed/scripts/run-e2e-test.sh` exercises `generate`/`migrate` for data migration but does not then run `terraform apply` against the generator's output and assert a clean plan. This is the scenario Tests 2, 3, and 4 cover manually.

**Recommended automation.** Add a new script `testbed/scripts/run-import-validation.sh` that:
1. Seeds the source testbed cluster with a known fixture set (include one analytics rule and one synonym whose names contain spaces, to regression-test URL-encoding).
2. Runs `./terraform-provider-typesense generate --host=... --output=./tmp`.
3. Runs `terraform apply` then `terraform plan -detailed-exitcode` and fails if exit != 0.
4. Edits one field in one resource file; runs `plan -detailed-exitcode` and fails if exit != 2 or if more than one resource appears in the diff.
5. Creates one out-of-band resource via direct API; re-runs `generate` into a second directory; asserts the new resource is present in the fresh `imports.tf`.
6. Tears down.

This keeps the existing unit tests for fast feedback and adds one long-running integration job for release gating. It does not require Typesense Cloud Management API access as long as Test 5's cluster filtering stays covered by unit tests.

