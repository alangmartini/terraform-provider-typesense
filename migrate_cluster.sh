#!/bin/bash

set -e

# Typesense Cluster Migration Script
# Exports all collections from source cluster and imports them to target cluster

# Source Cluster Configuration
SOURCE_HOST="8jc1etbnx2firg7mp-1.a1.typesense.net"
SOURCE_API_KEY="s5rbKtkWDCu4S3zmMGBzeJzqobiYQvOM"
SOURCE_PROTOCOL="https"
SOURCE_PORT="443"

# Target Cluster Configuration
TARGET_HOST="rd1ywsp7geluc9khp-1.a1.typesense.net"
TARGET_API_KEY="9ycySOrlvHAK2yl7JSrXrlhjn1PrJU66"
TARGET_PROTOCOL="https"
TARGET_PORT="443"

# Export directory
EXPORT_DIR="./typesense_export_$(date +%Y%m%d_%H%M%S)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create export directory
mkdir -p "$EXPORT_DIR"

log_info "Starting Typesense cluster migration"
log_info "Source: $SOURCE_HOST"
log_info "Target: $TARGET_HOST"
log_info "Export directory: $EXPORT_DIR"

# Step 1: Get list of collections from source
log_info "Fetching collections from source cluster..."
COLLECTIONS=$(curl -s -H "X-TYPESENSE-API-KEY: $SOURCE_API_KEY" \
    "$SOURCE_PROTOCOL://$SOURCE_HOST:$SOURCE_PORT/collections")

if [ -z "$COLLECTIONS" ] || [ "$COLLECTIONS" = "[]" ]; then
    log_warn "No collections found in source cluster"
    exit 0
fi

# Save collections list
echo "$COLLECTIONS" > "$EXPORT_DIR/collections_list.json"

# Extract collection names
COLLECTION_NAMES=$(echo "$COLLECTIONS" | jq -r '.[].name')

if [ -z "$COLLECTION_NAMES" ]; then
    log_warn "No collection names found"
    exit 0
fi

log_info "Found collections: $(echo $COLLECTION_NAMES | tr '\n' ' ')"

# Step 2: Export each collection's schema and documents
for COLLECTION in $COLLECTION_NAMES; do
    log_info "Processing collection: $COLLECTION"

    # Create collection directory
    mkdir -p "$EXPORT_DIR/$COLLECTION"

    # Export collection schema
    log_info "  Exporting schema..."
    curl -s -H "X-TYPESENSE-API-KEY: $SOURCE_API_KEY" \
        "$SOURCE_PROTOCOL://$SOURCE_HOST:$SOURCE_PORT/collections/$COLLECTION" \
        > "$EXPORT_DIR/$COLLECTION/schema.json"

    # Export documents using export endpoint (JSONL format)
    log_info "  Exporting documents..."
    curl -s -H "X-TYPESENSE-API-KEY: $SOURCE_API_KEY" \
        "$SOURCE_PROTOCOL://$SOURCE_HOST:$SOURCE_PORT/collections/$COLLECTION/documents/export" \
        > "$EXPORT_DIR/$COLLECTION/documents.jsonl"

    DOC_COUNT=$(wc -l < "$EXPORT_DIR/$COLLECTION/documents.jsonl" | tr -d ' ')
    log_info "  Exported $DOC_COUNT documents"
done

log_info "Export complete!"

# Step 3: Import to target cluster
log_info "Starting import to target cluster..."

for COLLECTION in $COLLECTION_NAMES; do
    log_info "Importing collection: $COLLECTION"

    # Check if collection already exists in target
    EXISTING=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "X-TYPESENSE-API-KEY: $TARGET_API_KEY" \
        "$TARGET_PROTOCOL://$TARGET_HOST:$TARGET_PORT/collections/$COLLECTION")

    if [ "$EXISTING" = "200" ]; then
        log_warn "  Collection '$COLLECTION' already exists in target. Skipping schema creation."
    else
        # Create collection in target (need to remove certain fields from schema)
        log_info "  Creating collection schema..."

        # Remove fields that shouldn't be in create request (num_documents, created_at, etc.)
        SCHEMA=$(cat "$EXPORT_DIR/$COLLECTION/schema.json" | jq 'del(.num_documents, .created_at, .num_memory_shards)')

        CREATE_RESULT=$(echo "$SCHEMA" | curl -s -X POST \
            -H "X-TYPESENSE-API-KEY: $TARGET_API_KEY" \
            -H "Content-Type: application/json" \
            -d @- \
            "$TARGET_PROTOCOL://$TARGET_HOST:$TARGET_PORT/collections")

        # Check if creation was successful
        if echo "$CREATE_RESULT" | jq -e '.name' > /dev/null 2>&1; then
            log_info "  Collection created successfully"
        else
            log_error "  Failed to create collection: $CREATE_RESULT"
            continue
        fi
    fi

    # Import documents
    DOC_FILE="$EXPORT_DIR/$COLLECTION/documents.jsonl"
    if [ -s "$DOC_FILE" ]; then
        log_info "  Importing documents..."

        IMPORT_RESULT=$(curl -s -X POST \
            -H "X-TYPESENSE-API-KEY: $TARGET_API_KEY" \
            -H "Content-Type: text/plain" \
            --data-binary @"$DOC_FILE" \
            "$TARGET_PROTOCOL://$TARGET_HOST:$TARGET_PORT/collections/$COLLECTION/documents/import?action=upsert")

        # Count successes and failures
        SUCCESS_COUNT=$(echo "$IMPORT_RESULT" | grep -c '"success":true' || true)
        FAIL_COUNT=$(echo "$IMPORT_RESULT" | grep -c '"success":false' || true)

        log_info "  Import complete: $SUCCESS_COUNT successful, $FAIL_COUNT failed"

        if [ "$FAIL_COUNT" -gt 0 ]; then
            log_warn "  Some documents failed to import. Check $EXPORT_DIR/$COLLECTION/import_errors.log"
            # Format errors for readability: code, error message, then document
            echo "$IMPORT_RESULT" | grep '"success":false' | jq -r '
                "---",
                "Code: \(.code)",
                "Error: \(.error)",
                "Document ID: \(.document | fromjson | .id // .stockcode // "unknown")",
                "Document: \(.document)"
            ' > "$EXPORT_DIR/$COLLECTION/import_errors.log"
        fi
    else
        log_info "  No documents to import"
    fi
done

log_info "Migration complete!"
log_info "Export data saved in: $EXPORT_DIR"
