# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Git Workflow

- **Always create a new branch when starting work** on a feature or fix. Use descriptive branch names that reflect the work being done.
- **Commit each atomic change** separately. Each commit should represent a single logical change that can stand on its own.

## Test-Driven Development (TDD)

**Always follow TDD when fixing bugs:**
1. When an error is reported, first create a test that reproduces the error
2. Verify the test fails with the expected error
3. Only then implement the fix
4. Continue until the test passes

This ensures we understand the root cause and have regression coverage.

## Testing Requirements

- **Run E2E tests after implementing new features.** After completing any new feature or significant bug fix, run the E2E testbed to verify the provider still works correctly:
  ```bash
  make testbed-e2e
  ```
- **For quick verification**, use `make testbed-verify` to check current state without full re-seeding.
- **Test with reduced dataset** for faster iteration during development:
  ```bash
  PRODUCTS_COUNT=100 USERS_COUNT=100 ARTICLES_COUNT=100 EVENTS_COUNT=100 EDGE_CASES_COUNT=50 make testbed-seed
  ```

## Consistency Tests

The provider includes a suite of tests to catch "inconsistent result after apply" errors caused by server-side default mismatches. These tests verify that computed attributes properly accept Typesense's server-side defaults.

**Run consistency tests:**
```bash
make testbed-up        # Start testbed first
make test-consistency  # Run consistency test suite
```

**When to add new consistency tests:**
- When adding new field attributes that have server-side defaults
- When modifying how computed values are handled
- When Typesense's API behavior changes

**Key principle:** Test with minimal configurations (only required fields) to expose any mismatch between what Terraform plans and what the API returns.

## E2E Testbed

The `testbed/` directory contains infrastructure for full end-to-end testing:

- `make testbed-up` - Start source (8108) and target (8109) Typesense clusters
- `make testbed-seed` - Populate source with ~50k test documents
- `make testbed-e2e` - Run complete migration test workflow
- `make testbed-verify` - Verify target matches source
- `make testbed-down` - Stop and clean up clusters

## Adding New Resources

When adding a new Terraform resource:
1. Add client methods in `internal/client/server_client.go`
2. Create resource file in `internal/resources/{resource_name}.go`
3. Register in `internal/provider/provider.go` Resources() function
4. Rebuild binary: `go build -o terraform-provider-typesense .`
5. Run `terraform validate` in examples to verify schema is recognized

## Terraform Development Notes

- **Dev override caching**: After adding new resources, must rebuild binary before `terraform validate` picks up changes
- **Sensitive variable transitivity**: Outputs using `count` based on sensitive vars inherit sensitivity; use `length(resource.name) > 0` pattern instead
- **Optional resource pattern**: Use `count = var.x != "" ? 1 : 0` for conditionally created resources
