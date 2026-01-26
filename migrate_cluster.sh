#!/bin/bash
set -e

# Typesense Cluster Migration Script
# Exports all collections with documents from source cluster and imports to target cluster

# Configuration via environment variables
SOURCE_HOST="${SOURCE_HOST:?Set SOURCE_HOST}"
SOURCE_API_KEY="${SOURCE_API_KEY:?Set SOURCE_API_KEY}"
SOURCE_PROTOCOL="${SOURCE_PROTOCOL:-https}"
SOURCE_PORT="${SOURCE_PORT:-443}"

TARGET_HOST="${TARGET_HOST:?Set TARGET_HOST}"
TARGET_API_KEY="${TARGET_API_KEY:?Set TARGET_API_KEY}"
TARGET_PROTOCOL="${TARGET_PROTOCOL:-https}"
TARGET_PORT="${TARGET_PORT:-443}"

EXPORT_DIR="./typesense_export_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$EXPORT_DIR"

echo "Migration: $SOURCE_HOST -> $TARGET_HOST"
echo "Export dir: $EXPORT_DIR"

# Get collections
COLLECTIONS=$(curl -s -H "X-TYPESENSE-API-KEY: $SOURCE_API_KEY" \
    "$SOURCE_PROTOCOL://$SOURCE_HOST:$SOURCE_PORT/collections")

COLLECTION_NAMES=$(echo "$COLLECTIONS" | jq -r '.[].name')
[ -z "$COLLECTION_NAMES" ] && echo "No collections found" && exit 0

echo "Collections: $(echo $COLLECTION_NAMES | tr '\n' ' ')"

# Export each collection
for COLLECTION in $COLLECTION_NAMES; do
    echo "Exporting: $COLLECTION"
    mkdir -p "$EXPORT_DIR/$COLLECTION"

    curl -s -H "X-TYPESENSE-API-KEY: $SOURCE_API_KEY" \
        "$SOURCE_PROTOCOL://$SOURCE_HOST:$SOURCE_PORT/collections/$COLLECTION" \
        > "$EXPORT_DIR/$COLLECTION/schema.json"

    curl -s -H "X-TYPESENSE-API-KEY: $SOURCE_API_KEY" \
        "$SOURCE_PROTOCOL://$SOURCE_HOST:$SOURCE_PORT/collections/$COLLECTION/documents/export" \
        > "$EXPORT_DIR/$COLLECTION/documents.jsonl"

    echo "  $(wc -l < "$EXPORT_DIR/$COLLECTION/documents.jsonl" | tr -d ' ') documents"
done

# Import to target
for COLLECTION in $COLLECTION_NAMES; do
    echo "Importing: $COLLECTION"

    # Check if collection exists
    EXISTING=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "X-TYPESENSE-API-KEY: $TARGET_API_KEY" \
        "$TARGET_PROTOCOL://$TARGET_HOST:$TARGET_PORT/collections/$COLLECTION")

    if [ "$EXISTING" != "200" ]; then
        # Create collection (remove computed fields)
        SCHEMA=$(cat "$EXPORT_DIR/$COLLECTION/schema.json" | jq 'del(.num_documents, .created_at, .num_memory_shards)')
        echo "$SCHEMA" | curl -s -X POST \
            -H "X-TYPESENSE-API-KEY: $TARGET_API_KEY" \
            -H "Content-Type: application/json" \
            -d @- \
            "$TARGET_PROTOCOL://$TARGET_HOST:$TARGET_PORT/collections" > /dev/null
    fi

    # Import documents
    DOC_FILE="$EXPORT_DIR/$COLLECTION/documents.jsonl"
    if [ -s "$DOC_FILE" ]; then
        RESULT=$(curl -s -X POST \
            -H "X-TYPESENSE-API-KEY: $TARGET_API_KEY" \
            -H "Content-Type: text/plain" \
            --data-binary @"$DOC_FILE" \
            "$TARGET_PROTOCOL://$TARGET_HOST:$TARGET_PORT/collections/$COLLECTION/documents/import?action=upsert")

        SUCCESS=$(echo "$RESULT" | grep -c '"success":true' || true)
        FAIL=$(echo "$RESULT" | grep -c '"success":false' || true)
        echo "  $SUCCESS ok, $FAIL failed"

        [ "$FAIL" -gt 0 ] && echo "$RESULT" | grep '"success":false' > "$EXPORT_DIR/$COLLECTION/errors.log"
    fi
done

echo "Done! Export saved in: $EXPORT_DIR"
