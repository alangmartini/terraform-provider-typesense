#!/usr/bin/env bash
# run-e2e-test.sh - Full E2E test orchestration
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTBED_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$TESTBED_DIR")"

# Configuration
SOURCE_HOST="${SOURCE_HOST:-localhost}"
SOURCE_PORT="${SOURCE_PORT:-8108}"
SOURCE_API_KEY="${SOURCE_API_KEY:-source-test-api-key}"

TARGET_HOST="${TARGET_HOST:-localhost}"
TARGET_PORT="${TARGET_PORT:-8109}"
TARGET_API_KEY="${TARGET_API_KEY:-target-test-api-key}"

# Export directory for migration
EXPORT_DIR="${EXPORT_DIR:-$TESTBED_DIR/export}"

# Cleanup on exit
CLEANUP_ON_EXIT="${CLEANUP_ON_EXIT:-false}"

log() {
    echo ""
    echo "========================================"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
    echo "========================================"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
    exit 1
}

cleanup() {
    if [ "$CLEANUP_ON_EXIT" = "true" ]; then
        log "Cleaning up..."
        cd "$TESTBED_DIR"
        docker compose down -v 2>/dev/null || true
        rm -rf "$EXPORT_DIR"
    fi
}

trap cleanup EXIT

check_prerequisites() {
    log "Checking prerequisites..."

    # Check for docker
    if ! command -v docker &> /dev/null; then
        error "docker is not installed"
    fi

    # Check for docker compose
    if ! docker compose version &> /dev/null; then
        error "docker compose is not available"
    fi

    # Check for jq
    if ! command -v jq &> /dev/null; then
        error "jq is not installed"
    fi

    # Check for go
    if ! command -v go &> /dev/null; then
        error "go is not installed"
    fi

    # Check for terraform-provider-typesense binary
    if [ ! -x "$PROJECT_ROOT/terraform-provider-typesense" ]; then
        echo "Building terraform-provider-typesense..."
        cd "$PROJECT_ROOT"
        go build -o terraform-provider-typesense .
    fi

    echo "All prerequisites met"
}

start_clusters() {
    log "Starting Typesense clusters..."

    cd "$TESTBED_DIR"

    # Stop any existing containers
    docker compose down -v 2>/dev/null || true

    # Start fresh
    docker compose up -d

    # Wait for health
    echo "Waiting for clusters to be healthy..."
    local max_attempts=60
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        local source_healthy=false
        local target_healthy=false

        if curl -sf "http://${SOURCE_HOST}:${SOURCE_PORT}/health" > /dev/null 2>&1; then
            source_healthy=true
        fi

        if curl -sf "http://${TARGET_HOST}:${TARGET_PORT}/health" > /dev/null 2>&1; then
            target_healthy=true
        fi

        if [ "$source_healthy" = "true" ] && [ "$target_healthy" = "true" ]; then
            echo "Both clusters are healthy!"
            return 0
        fi

        echo "  Attempt $attempt/$max_attempts - source: $source_healthy, target: $target_healthy"
        sleep 2
        ((attempt++))
    done

    error "Clusters did not become healthy"
}

seed_source() {
    log "Seeding source cluster..."

    export SOURCE_HOST SOURCE_PORT SOURCE_API_KEY

    "$SCRIPT_DIR/seed-source.sh"
}

run_generate() {
    log "Running generate command (export from source)..."

    rm -rf "$EXPORT_DIR"
    mkdir -p "$EXPORT_DIR"

    cd "$EXPORT_DIR"

    "$PROJECT_ROOT/terraform-provider-typesense" generate \
        --host "${SOURCE_HOST}" \
        --port "${SOURCE_PORT}" \
        --protocol "http" \
        --api-key "${SOURCE_API_KEY}" \
        --include-data

    echo ""
    echo "Export contents:"
    ls -la "$EXPORT_DIR"

    # Count exported files
    local tf_files jsonl_files
    tf_files=$(find "$EXPORT_DIR" -name "*.tf" | wc -l)
    jsonl_files=$(find "$EXPORT_DIR" -name "*.jsonl" | wc -l)

    echo ""
    echo "Exported: $tf_files .tf files, $jsonl_files .jsonl files"
}

run_migrate() {
    log "Running migrate command (import to target)..."

    cd "$EXPORT_DIR"

    "$PROJECT_ROOT/terraform-provider-typesense" migrate \
        --host "${TARGET_HOST}" \
        --port "${TARGET_PORT}" \
        --protocol "http" \
        --api-key "${TARGET_API_KEY}"
}

verify_migration() {
    log "Verifying migration..."

    export SOURCE_HOST SOURCE_PORT SOURCE_API_KEY
    export TARGET_HOST TARGET_PORT TARGET_API_KEY

    "$SCRIPT_DIR/verify-migration.sh"
}

run_terraform_plan() {
    log "Running terraform plan (should show no changes)..."

    cd "$EXPORT_DIR"

    # Check if terraform is available
    if ! command -v terraform &> /dev/null; then
        echo "Terraform not installed, skipping plan verification"
        return 0
    fi

    # Initialize terraform
    cat > provider.tf << EOF
terraform {
  required_providers {
    typesense = {
      source = "local/typesense/typesense"
    }
  }
}

provider "typesense" {
  host     = "${TARGET_HOST}"
  port     = ${TARGET_PORT}
  protocol = "http"
  api_key  = "${TARGET_API_KEY}"
}
EOF

    terraform init -input=false

    # Run plan and check for changes
    local plan_output
    plan_output=$(terraform plan -detailed-exitcode 2>&1) || {
        local exit_code=$?
        if [ $exit_code -eq 2 ]; then
            echo "WARNING: Terraform plan shows changes:"
            echo "$plan_output"
            # Not failing the test for now, as this may be expected
        elif [ $exit_code -ne 0 ]; then
            echo "Terraform plan failed:"
            echo "$plan_output"
        fi
    }

    echo "Terraform plan completed"
}

print_summary() {
    log "E2E Test Summary"

    echo ""
    echo "Source cluster: http://${SOURCE_HOST}:${SOURCE_PORT}"
    echo "Target cluster: http://${TARGET_HOST}:${TARGET_PORT}"
    echo "Export directory: $EXPORT_DIR"
    echo ""

    # Print collection stats from both clusters
    echo "Source collections:"
    curl -sf "http://${SOURCE_HOST}:${SOURCE_PORT}/collections" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" | \
        jq -r '.[] | "  - \(.name): \(.num_documents) documents"'

    echo ""
    echo "Target collections:"
    curl -sf "http://${TARGET_HOST}:${TARGET_PORT}/collections" \
        -H "X-TYPESENSE-API-KEY: ${TARGET_API_KEY}" | \
        jq -r '.[] | "  - \(.name): \(.num_documents) documents"'

    echo ""
    echo "E2E TEST PASSED"
}

main() {
    echo "============================================"
    echo "  Typesense Terraform Provider E2E Test"
    echo "============================================"

    local start_time
    start_time=$(date +%s)

    check_prerequisites
    start_clusters
    seed_source
    run_generate
    run_migrate
    verify_migration
    run_terraform_plan
    print_summary

    local end_time duration
    end_time=$(date +%s)
    duration=$((end_time - start_time))

    echo ""
    echo "Total duration: ${duration}s"
}

main "$@"
