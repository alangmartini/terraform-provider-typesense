.PHONY: test test-acc test-acc-ci test-consistency test-conversation-model \
	start-typesense stop-typesense build clean \
	testbed-up testbed-down testbed-seed testbed-e2e testbed-verify testbed-clean

# Configuration
TYPESENSE_API_KEY := test-api-key-for-acceptance-tests
PORT := 8108
CONTAINER_NAME := typesense-test
TYPESENSE_HOST := localhost
TYPESENSE_PROTOCOL := http

# Build the provider binary
build:
	@echo "Building provider binary..."

# Run unit tests only (no acceptance tests)
test:
	@echo "Running unit tests..."
	go test ./... -v -short

# Start local Typesense container for development
start-typesense:
	@echo "Setting up local Typesense instance..."
	@docker stop $(CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(CONTAINER_NAME) 2>/dev/null || true
	@rm -rf "$(PWD)/typesense-test-data"
	@mkdir -p "$(PWD)/typesense-test-data"
	@echo "Starting Typesense container..."
	@docker run -d -p $(PORT):$(PORT) --name $(CONTAINER_NAME) \
		-v "$(PWD)/typesense-test-data:/data" \
		typesense/typesense:29.0.rc30 \
		--data-dir /data \
		--api-key=$(TYPESENSE_API_KEY) \
		--enable-cors
	@echo "Waiting for Typesense to be ready..."
	@until curl -s -o /dev/null -w "%{http_code}" "http://localhost:$(PORT)/health" \
		-H "X-TYPESENSE-API-KEY: $(TYPESENSE_API_KEY)" | grep -q "200"; do \
		echo "  Still waiting..."; \
		sleep 2; \
	done
	@echo ""
	@echo "✓ Typesense is ready!"
	@echo ""
	@echo "Environment variables:"
	@echo "  TYPESENSE_HOST=$(TYPESENSE_HOST)"
	@echo "  TYPESENSE_PORT=$(PORT)"
	@echo "  TYPESENSE_PROTOCOL=$(TYPESENSE_PROTOCOL)"
	@echo "  TYPESENSE_API_KEY=$(TYPESENSE_API_KEY)"

# Stop and remove Typesense container
stop-typesense:
	@echo "Stopping and removing Typesense test container..."
	@docker stop $(CONTAINER_NAME) 2>/dev/null || true
	@docker rm $(CONTAINER_NAME) 2>/dev/null || true
	@echo "Removing test data directory..."
	@rm -rf "$(PWD)/typesense-test-data"
	@echo "✓ Cleanup complete!"

# Run acceptance tests (starts Typesense, runs tests, cleans up)
test-acc:
	@echo "Starting acceptance test run..."
	@$(MAKE) start-typesense
	@echo ""
	@echo "Running acceptance tests..."
	@export TYPESENSE_HOST=$(TYPESENSE_HOST) && \
	export TYPESENSE_PORT=$(PORT) && \
	export TYPESENSE_PROTOCOL=$(TYPESENSE_PROTOCOL) && \
	export TYPESENSE_API_KEY=$(TYPESENSE_API_KEY) && \
	export TF_ACC=1 && \
	go test ./... -v || ($(MAKE) stop-typesense && exit 1)
	@echo ""
	@$(MAKE) stop-typesense
	@echo ""
	@echo "✓ Acceptance tests complete!"

# Run all acceptance tests (CI-friendly, assumes Typesense is already running)
# Used in GitHub Actions with Typesense service container
test-acc-ci:
	@echo "Running all acceptance tests..."
	@export TYPESENSE_HOST=$${TYPESENSE_HOST:-localhost} && \
	export TYPESENSE_PORT=$${TYPESENSE_PORT:-8108} && \
	export TYPESENSE_PROTOCOL=$${TYPESENSE_PROTOCOL:-http} && \
	export TYPESENSE_API_KEY=$${TYPESENSE_API_KEY:-source-test-api-key} && \
	export TF_ACC=1 && \
	go test -v -timeout 30m ./internal/resources/...
	@echo ""
	@echo "✓ All acceptance tests complete!"

# Run conversation model validation tests
# These tests verify Typesense properly validates history collection schema
test-conversation-model:
	@echo "Running conversation model validation tests..."
	@if ! curl -sf http://localhost:8108/health > /dev/null 2>&1; then \
		echo "Error: Testbed not running. Run 'make testbed-up' first."; \
		exit 1; \
	fi
	@echo ""
	@export TYPESENSE_HOST=localhost && \
	export TYPESENSE_PORT=8108 && \
	export TYPESENSE_PROTOCOL=http && \
	export TYPESENSE_API_KEY=source-test-api-key && \
	export TF_ACC=1 && \
	go test -v -run "TestAccConversationModelResource" ./internal/resources
	@echo ""
	@echo "✓ Conversation model tests complete!"

# Run consistency tests against the testbed
# These tests catch "inconsistent result after apply" errors from server-side default mismatches
test-consistency:
	@echo "Running consistency tests against testbed..."
	@if ! curl -sf http://localhost:8108/health > /dev/null 2>&1; then \
		echo "Error: Testbed not running. Run 'make testbed-up' first."; \
		exit 1; \
	fi
	@echo ""
	@export TYPESENSE_HOST=localhost && \
	export TYPESENSE_PORT=8108 && \
	export TYPESENSE_PROTOCOL=http && \
	export TYPESENSE_API_KEY=source-test-api-key && \
	export TF_ACC=1 && \
	go test -v -run "TestAccCollectionResource_(minimal|numeric|string|explicit|allField|mixed|object|geopoint)" ./internal/resources
	@echo ""
	@echo "✓ Consistency tests complete!"

# Clean up build artifacts and test data
clean:
	@echo "Cleaning up..."
	@rm -f terraform-provider-typesense
	@rm -rf typesense-test-data
	@echo "✓ Clean complete!"

# ==============================================================================
# E2E Testbed Targets
# ==============================================================================

# Testbed configuration
TESTBED_DIR := $(PWD)/testbed
TESTBED_COMPOSE := $(TESTBED_DIR)/docker-compose.yml

# Start both source and target Typesense clusters
testbed-up:
	@echo "Starting E2E testbed clusters..."
	@docker compose -f $(TESTBED_COMPOSE) up -d
	@echo "Waiting for clusters to be healthy..."
	@until curl -sf http://localhost:8108/health > /dev/null 2>&1; do sleep 2; done
	@until curl -sf http://localhost:8109/health > /dev/null 2>&1; do sleep 2; done
	@echo ""
	@echo "✓ Testbed clusters are ready!"
	@echo "  Source: http://localhost:8108 (API key: source-test-api-key)"
	@echo "  Target: http://localhost:8109 (API key: target-test-api-key)"

# Stop and remove testbed clusters with volumes
testbed-down:
	@echo "Stopping E2E testbed clusters..."
	@docker compose -f $(TESTBED_COMPOSE) down -v
	@echo "✓ Testbed stopped and cleaned!"

# Seed the source cluster with test fixtures
testbed-seed:
	@echo "Seeding source cluster with test data..."
	@$(TESTBED_DIR)/scripts/seed-source.sh

# Run complete E2E test workflow
testbed-e2e:
	@echo "Running complete E2E test..."
	@$(TESTBED_DIR)/scripts/run-e2e-test.sh

# Verify migration between source and target
testbed-verify:
	@echo "Verifying migration..."
	@$(TESTBED_DIR)/scripts/verify-migration.sh

# Clean all testbed data (containers, volumes, exports)
testbed-clean:
	@echo "Cleaning all testbed data..."
	@docker compose -f $(TESTBED_COMPOSE) down -v 2>/dev/null || true
	@rm -rf $(TESTBED_DIR)/export
	@echo "✓ Testbed cleaned!"
