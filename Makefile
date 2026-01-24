.PHONY: test test-acc start-typesense stop-typesense build clean

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

# Clean up build artifacts and test data
clean:
	@echo "Cleaning up..."
	@rm -f terraform-provider-typesense
	@rm -rf typesense-test-data
	@echo "✓ Clean complete!"
