#!/usr/bin/env bash
# verify-migration.sh - Verify target cluster matches source cluster
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
SOURCE_HOST="${SOURCE_HOST:-localhost}"
SOURCE_PORT="${SOURCE_PORT:-8108}"
SOURCE_API_KEY="${SOURCE_API_KEY:-source-test-api-key}"
SOURCE_URL="http://${SOURCE_HOST}:${SOURCE_PORT}"

TARGET_HOST="${TARGET_HOST:-localhost}"
TARGET_PORT="${TARGET_PORT:-8109}"
TARGET_API_KEY="${TARGET_API_KEY:-target-test-api-key}"
TARGET_URL="http://${TARGET_HOST}:${TARGET_PORT}"

# Verification settings
SAMPLE_SIZE="${SAMPLE_SIZE:-10}"  # Number of random documents to sample per collection
STRICT_MODE="${STRICT_MODE:-true}"  # Fail on any mismatch

ERRORS=0
WARNINGS=0

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    log "ERROR: $*" >&2
    ((++ERRORS))
}

warn() {
    log "WARNING: $*" >&2
    ((++WARNINGS))
}

success() {
    log "OK: $*"
}

wait_for_health() {
    local url=$1
    local name=$2
    local max_attempts=30
    local attempt=1

    log "Waiting for $name to be healthy..."
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "${url}/health" > /dev/null 2>&1; then
            return 0
        fi
        sleep 2
        ((attempt++))
    done

    error "$name did not become healthy"
    return 1
}

get_collections() {
    local url=$1
    local api_key=$2

    curl -sf "${url}/collections" \
        -H "X-TYPESENSE-API-KEY: ${api_key}" | jq -r '.[].name' | sort
}

get_collection_info() {
    local url=$1
    local api_key=$2
    local collection=$3

    curl -sf "${url}/collections/${collection}" \
        -H "X-TYPESENSE-API-KEY: ${api_key}"
}

get_document() {
    local url=$1
    local api_key=$2
    local collection=$3
    local doc_id=$4

    curl -sf "${url}/collections/${collection}/documents/${doc_id}" \
        -H "X-TYPESENSE-API-KEY: ${api_key}" 2>/dev/null || echo ""
}

search_random_docs() {
    local url=$1
    local api_key=$2
    local collection=$3
    local count=$4

    # Get random documents using a wildcard search with random page
    local total
    total=$(curl -sf "${url}/collections/${collection}" \
        -H "X-TYPESENSE-API-KEY: ${api_key}" | jq -r '.num_documents')

    if [ "$total" -eq 0 ]; then
        echo "[]"
        return
    fi

    local max_page=$((total / count))
    if [ "$max_page" -lt 1 ]; then
        max_page=1
    fi
    local random_page=$((RANDOM % max_page + 1))

    curl -sf "${url}/collections/${collection}/documents/search?q=*&per_page=${count}&page=${random_page}" \
        -H "X-TYPESENSE-API-KEY: ${api_key}" | jq -r '.hits[].document.id'
}

verify_collection_exists() {
    local collection=$1

    log "Checking collection: $collection"

    local source_info target_info
    source_info=$(get_collection_info "$SOURCE_URL" "$SOURCE_API_KEY" "$collection")
    target_info=$(get_collection_info "$TARGET_URL" "$TARGET_API_KEY" "$collection")

    if [ -z "$target_info" ] || [ "$target_info" = "null" ]; then
        error "Collection '$collection' missing from target"
        return 1
    fi

    return 0
}

verify_document_counts() {
    local collection=$1

    local source_count target_count
    source_count=$(get_collection_info "$SOURCE_URL" "$SOURCE_API_KEY" "$collection" | jq -r '.num_documents')
    target_count=$(get_collection_info "$TARGET_URL" "$TARGET_API_KEY" "$collection" | jq -r '.num_documents')

    if [ "$source_count" -ne "$target_count" ]; then
        error "Document count mismatch for '$collection': source=$source_count, target=$target_count"
        return 1
    fi

    success "Document count matches for '$collection': $source_count"
    return 0
}

verify_schema_fields() {
    local collection=$1

    local source_fields target_fields
    source_fields=$(get_collection_info "$SOURCE_URL" "$SOURCE_API_KEY" "$collection" | \
        jq -r '[.fields[] | {name, type}] | sort_by(.name)')
    target_fields=$(get_collection_info "$TARGET_URL" "$TARGET_API_KEY" "$collection" | \
        jq -r '[.fields[] | {name, type}] | sort_by(.name)')

    if [ "$source_fields" != "$target_fields" ]; then
        error "Schema mismatch for '$collection'"
        log "Source fields: $source_fields"
        log "Target fields: $target_fields"
        return 1
    fi

    local field_count
    field_count=$(echo "$source_fields" | jq length)
    success "Schema matches for '$collection': $field_count fields"
    return 0
}

verify_sample_documents() {
    local collection=$1

    log "Sampling $SAMPLE_SIZE random documents from '$collection'..."

    local doc_ids
    doc_ids=$(search_random_docs "$SOURCE_URL" "$SOURCE_API_KEY" "$collection" "$SAMPLE_SIZE")

    if [ -z "$doc_ids" ]; then
        warn "No documents to sample in '$collection'"
        return 0
    fi

    local mismatches=0
    while read -r doc_id; do
        [ -z "$doc_id" ] && continue

        local source_doc target_doc
        source_doc=$(get_document "$SOURCE_URL" "$SOURCE_API_KEY" "$collection" "$doc_id")
        target_doc=$(get_document "$TARGET_URL" "$TARGET_API_KEY" "$collection" "$doc_id")

        if [ -z "$target_doc" ]; then
            error "Document '$doc_id' missing from target '$collection'"
            ((mismatches++))
            continue
        fi

        # Compare JSON (sorted keys for consistency)
        local source_sorted target_sorted
        source_sorted=$(echo "$source_doc" | jq -S '.')
        target_sorted=$(echo "$target_doc" | jq -S '.')

        if [ "$source_sorted" != "$target_sorted" ]; then
            error "Document '$doc_id' differs between source and target in '$collection'"
            log "Source: $source_sorted"
            log "Target: $target_sorted"
            ((mismatches++))
        fi
    done <<< "$doc_ids"

    if [ "$mismatches" -eq 0 ]; then
        success "Sample documents match for '$collection'"
    fi

    return "$mismatches"
}

verify_synonyms() {
    local collection=$1

    local source_synonyms target_synonyms
    source_synonyms=$(curl -sf "${SOURCE_URL}/collections/${collection}/synonyms" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" 2>/dev/null | jq -r '.synonyms | length' || echo "0")
    target_synonyms=$(curl -sf "${TARGET_URL}/collections/${collection}/synonyms" \
        -H "X-TYPESENSE-API-KEY: ${TARGET_API_KEY}" 2>/dev/null | jq -r '.synonyms | length' || echo "0")

    if [ "$source_synonyms" != "$target_synonyms" ]; then
        warn "Synonym count mismatch for '$collection': source=$source_synonyms, target=$target_synonyms"
        return 0  # Non-fatal
    fi

    if [ "$source_synonyms" -gt 0 ]; then
        success "Synonyms match for '$collection': $source_synonyms"
    fi
    return 0
}

verify_overrides() {
    local collection=$1

    local source_overrides target_overrides
    source_overrides=$(curl -sf "${SOURCE_URL}/collections/${collection}/overrides" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" 2>/dev/null | jq -r '.overrides | length' || echo "0")
    target_overrides=$(curl -sf "${TARGET_URL}/collections/${collection}/overrides" \
        -H "X-TYPESENSE-API-KEY: ${TARGET_API_KEY}" 2>/dev/null | jq -r '.overrides | length' || echo "0")

    if [ "$source_overrides" != "$target_overrides" ]; then
        warn "Override count mismatch for '$collection': source=$source_overrides, target=$target_overrides"
        return 0  # Non-fatal
    fi

    if [ "$source_overrides" -gt 0 ]; then
        success "Overrides match for '$collection': $source_overrides"
    fi
    return 0
}

verify_stopwords() {
    log "Checking stopwords..."

    local source_stopwords target_stopwords
    source_stopwords=$(curl -sf "${SOURCE_URL}/stopwords" \
        -H "X-TYPESENSE-API-KEY: ${SOURCE_API_KEY}" 2>/dev/null | jq -r '.stopwords | length' || echo "0")
    target_stopwords=$(curl -sf "${TARGET_URL}/stopwords" \
        -H "X-TYPESENSE-API-KEY: ${TARGET_API_KEY}" 2>/dev/null | jq -r '.stopwords | length' || echo "0")

    if [ "$source_stopwords" != "$target_stopwords" ]; then
        warn "Stopwords count mismatch: source=$source_stopwords, target=$target_stopwords"
        return 0
    fi

    if [ "$source_stopwords" -gt 0 ]; then
        success "Stopwords match: $source_stopwords sets"
    fi
    return 0
}

main() {
    log "=== Migration Verification ==="
    log "Source: $SOURCE_URL"
    log "Target: $TARGET_URL"
    log "Sample size: $SAMPLE_SIZE documents per collection"
    log ""

    # Wait for both clusters
    wait_for_health "$SOURCE_URL" "source" || exit 1
    wait_for_health "$TARGET_URL" "target" || exit 1

    # Get source collections
    local source_collections
    source_collections=$(get_collections "$SOURCE_URL" "$SOURCE_API_KEY")

    if [ -z "$source_collections" ]; then
        error "No collections found in source cluster"
        exit 1
    fi

    log "Found collections: $(echo "$source_collections" | tr '\n' ' ')"
    log ""

    # Verify each collection
    while read -r collection; do
        [ -z "$collection" ] && continue

        log "--- Verifying: $collection ---"

        verify_collection_exists "$collection" || continue
        verify_document_counts "$collection"
        verify_schema_fields "$collection"
        verify_sample_documents "$collection"
        verify_synonyms "$collection"
        verify_overrides "$collection"

        log ""
    done <<< "$source_collections"

    # Verify stopwords
    verify_stopwords

    # Summary
    log ""
    log "=== Verification Summary ==="
    log "Errors: $ERRORS"
    log "Warnings: $WARNINGS"

    if [ "$ERRORS" -gt 0 ]; then
        log "FAILED: Migration verification found $ERRORS errors"
        exit 1
    elif [ "$WARNINGS" -gt 0 ]; then
        log "PASSED with warnings: $WARNINGS issues found (non-critical)"
        exit 0
    else
        log "PASSED: All verifications successful"
        exit 0
    fi
}

main "$@"
