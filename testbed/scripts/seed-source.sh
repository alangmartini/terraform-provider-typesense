#!/usr/bin/env bash
# seed-source.sh - Populate source Typesense cluster with test fixtures
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTBED_DIR="$(dirname "$SCRIPT_DIR")"
FIXTURES_DIR="$TESTBED_DIR/fixtures"
SCHEMAS_DIR="$FIXTURES_DIR/schemas"

# Configuration
SOURCE_HOST="${SOURCE_HOST:-localhost}"
SOURCE_PORT="${SOURCE_PORT:-8108}"
SOURCE_API_KEY="${SOURCE_API_KEY:-source-test-api-key}"
SOURCE_URL="http://${SOURCE_HOST}:${SOURCE_PORT}"

# Document counts per collection
PRODUCTS_COUNT="${PRODUCTS_COUNT:-10000}"
USERS_COUNT="${USERS_COUNT:-10000}"
ARTICLES_COUNT="${ARTICLES_COUNT:-10000}"
EVENTS_COUNT="${EVENTS_COUNT:-10000}"
EDGE_CASES_COUNT="${EDGE_CASES_COUNT:-1000}"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    log "ERROR: $*" >&2
    exit 1
}

wait_for_health() {
    local max_attempts=30
    local attempt=1

    log "Waiting for source cluster to be healthy..."
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "${SOURCE_URL}/health" -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" > /dev/null 2>&1; then
            log "Source cluster is healthy"
            return 0
        fi
        log "  Attempt $attempt/$max_attempts - waiting..."
        sleep 2
        ((attempt++))
    done

    error "Source cluster did not become healthy after $max_attempts attempts"
}

delete_collection_if_exists() {
    local collection=$1
    local response

    response=$(curl -sf "${SOURCE_URL}/collections/${collection}" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" 2>/dev/null || echo "")

    if [ -n "$response" ] && [ "$response" != "null" ]; then
        log "Deleting existing collection: $collection"
        curl -sf -X DELETE "${SOURCE_URL}/collections/${collection}" \
            -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" > /dev/null
    fi
}

create_collection() {
    local schema_file=$1
    local collection_name
    collection_name=$(jq -r '.name' "$schema_file")

    log "Creating collection: $collection_name"

    if ! curl -sf -X POST "${SOURCE_URL}/collections" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" \
        -H "Content-Type: application/json" \
        -d @"$schema_file" > /dev/null; then
        error "Failed to create collection: $collection_name"
    fi
}

import_documents() {
    local collection=$1
    local count=$2

    log "Generating and importing $count documents to $collection..."

    # Generate fixtures and pipe directly to import API
    cd "$FIXTURES_DIR"
    go run generate-fixtures.go "$collection" "$count" | \
        curl -sf -X POST "${SOURCE_URL}/collections/${collection}/documents/import?action=create" \
            -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" \
            -H "Content-Type: text/plain" \
            --data-binary @- > /dev/null

    # Verify count
    local actual_count
    actual_count=$(curl -sf "${SOURCE_URL}/collections/${collection}" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" | jq -r '.num_documents')

    log "  Imported $actual_count documents to $collection"
}

create_synonyms() {
    local collection=$1

    log "Creating synonyms for $collection..."

    # Example synonyms
    curl -sf -X PUT "${SOURCE_URL}/collections/${collection}/synonyms/laptop-notebook" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" \
        -H "Content-Type: application/json" \
        -d '{"synonyms": ["laptop", "notebook", "portable computer"]}' > /dev/null || true

    curl -sf -X PUT "${SOURCE_URL}/collections/${collection}/synonyms/phone-mobile" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" \
        -H "Content-Type: application/json" \
        -d '{"synonyms": ["phone", "mobile", "smartphone", "cell phone"]}' > /dev/null || true
}

create_overrides() {
    local collection=$1

    log "Creating overrides for $collection..."

    # Example override - pin specific results
    curl -sf -X PUT "${SOURCE_URL}/collections/${collection}/overrides/featured-products" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" \
        -H "Content-Type: application/json" \
        -d '{
            "rule": {"query": "featured", "match": "contains"},
            "includes": [{"id": "prod_000001", "position": 1}],
            "filter_by": "in_stock:true"
        }' > /dev/null || true
}

create_stopwords() {
    log "Creating stopwords set..."

    curl -sf -X PUT "${SOURCE_URL}/stopwords/common-words" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" \
        -H "Content-Type: application/json" \
        -d '{
            "stopwords": ["the", "a", "an", "and", "or", "but", "is", "are", "was", "were"],
            "locale": "en"
        }' > /dev/null || true
}

main() {
    log "Starting source cluster seeding..."
    log "Configuration:"
    log "  Source URL: $SOURCE_URL"
    log "  Products: $PRODUCTS_COUNT"
    log "  Users: $USERS_COUNT"
    log "  Articles: $ARTICLES_COUNT"
    log "  Events: $EVENTS_COUNT"
    log "  Edge Cases: $EDGE_CASES_COUNT"

    # Wait for cluster
    wait_for_health

    # Delete existing collections
    log "Clearing existing data..."
    for schema in "$SCHEMAS_DIR"/*.json; do
        collection_name=$(jq -r '.name' "$schema")
        delete_collection_if_exists "$collection_name"
    done

    # Delete existing stopwords
    curl -sf -X DELETE "${SOURCE_URL}/stopwords/common-words" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" > /dev/null 2>&1 || true

    # Create collections
    log "Creating collections..."
    for schema in "$SCHEMAS_DIR"/*.json; do
        create_collection "$schema"
    done

    # Import documents
    import_documents "products" "$PRODUCTS_COUNT"
    import_documents "users" "$USERS_COUNT"
    import_documents "articles" "$ARTICLES_COUNT"
    import_documents "events" "$EVENTS_COUNT"
    import_documents "edge_cases" "$EDGE_CASES_COUNT"

    # Create additional resources
    create_synonyms "products"
    create_overrides "products"
    create_stopwords

    # Summary
    log ""
    log "=== Seeding Complete ==="
    log "Collections created:"
    curl -sf "${SOURCE_URL}/collections" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" | \
        jq -r '.[] | "  - \(.name): \(.num_documents) documents"'

    log ""
    log "Total documents: $((PRODUCTS_COUNT + USERS_COUNT + ARTICLES_COUNT + EVENTS_COUNT + EDGE_CASES_COUNT))"
    log "Source cluster is ready for testing!"
}

main "$@"
