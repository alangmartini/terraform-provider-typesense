# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Git Workflow

- **Always create a new branch when starting work** on a feature or fix. Use descriptive branch names that reflect the work being done.
- **Commit each atomic change** separately. Each commit should represent a single logical change that can stand on its own.

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

## E2E Testbed

The `testbed/` directory contains infrastructure for full end-to-end testing:

- `make testbed-up` - Start source (8108) and target (8109) Typesense clusters
- `make testbed-seed` - Populate source with ~50k test documents
- `make testbed-e2e` - Run complete migration test workflow
- `make testbed-verify` - Verify target matches source
- `make testbed-down` - Stop and clean up clusters
