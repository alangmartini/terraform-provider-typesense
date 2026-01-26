#!/bin/bash
set -e

# Typesense Cluster Migration Script
# Exports all collections with documents from source cluster and imports to target cluster

# Configuration via environment variables
SOURCE_HOST="sl1n8ub6gvziopq4p-1.a1.typesense.net"
SOURCE_API_KEY="IBZAimI1JYjk5ToXbFvpwJLpLmdGFJH9"
SOURCE_PROTOCOL="${SOURCE_PROTOCOL:-https}"
SOURCE_PORT="${SOURCE_PORT:-443}"

TARGET_HOST="rd1ywsp7geluc9khp-1.a1.typesense.net"
TARGET_API_KEY="9ycySOrlvHAK2yl7JSrXrlhjn1PrJU66"
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
        RESULT_FILE="$EXPORT_DIR/$COLLECTION/import_result.jsonl"

        # Stream import - uses -T to stream file without loading into memory
        curl -s -X POST \
            -H "X-TYPESENSE-API-KEY: $TARGET_API_KEY" \
            -H "Content-Type: text/plain" \
            -T "$DOC_FILE" \
            "$TARGET_PROTOCOL://$TARGET_HOST:$TARGET_PORT/collections/$COLLECTION/documents/import?action=upsert" \
            > "$RESULT_FILE"

        # Count results from file (memory efficient)
        SUCCESS=$(grep -c '"success":true' "$RESULT_FILE" || true)
        FAIL=$(grep -c '"success":false' "$RESULT_FILE" || true)
        echo "  $SUCCESS ok, $FAIL failed"

        if [ "$FAIL" -gt 0 ]; then
            # Error summary
            echo "  Error summary:"
            grep '"success":false' "$RESULT_FILE" | jq -sr '
                group_by(.error) |
                map({error: .[0].error, count: length}) |
                sort_by(-.count) |
                .[] | "    \(.error) -> \(.count) documents"
            '

            # Save detailed errors
            grep '"success":false' "$RESULT_FILE" | jq -r '
                "---",
                "Code: \(.code)",
                "Error: \(.error)",
                "Document ID: \(.document | fromjson | .id // "unknown")",
                "Document: \(.document)"
            ' > "$EXPORT_DIR/$COLLECTION/errors.log"
        fi

        # Clean up result file (optional - keep for debugging)
        rm -f "$RESULT_FILE"
    fi
done

echo "Done! Export saved in: $EXPORT_DIR"
