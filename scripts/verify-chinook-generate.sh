#!/usr/bin/env bash
# Verifies the Chinook example against a real local Typesense instance, then
# runs generate --include-data against that populated instance.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CHINOOK_DIR="$PROJECT_ROOT/examples/chinook"
DATA_DIR="$CHINOOK_DIR/data"
PROVIDER_BINARY="$PROJECT_ROOT/terraform-provider-typesense"
case "$(uname -s 2>/dev/null || echo unknown)" in
    MINGW*|MSYS*|CYGWIN*)
        PROVIDER_BINARY="$PROJECT_ROOT/terraform-provider-typesense.exe"
        ;;
esac

TYPESENSE_HOST="${TYPESENSE_HOST:-localhost}"
TYPESENSE_PORT="${TYPESENSE_PORT:-8108}"
TYPESENSE_PROTOCOL="${TYPESENSE_PROTOCOL:-http}"
TYPESENSE_API_KEY="${TYPESENSE_API_KEY:-test-api-key-for-acceptance-tests}"
BASE_URL="${TYPESENSE_PROTOCOL}://${TYPESENSE_HOST}:${TYPESENSE_PORT}"

EXPORT_ROOT="${CHINOOK_EXPORT_DIR:-$PROJECT_ROOT/tmp/chinook-generate}"
GENERATED_DIR="$EXPORT_ROOT/generated"

CORE_COLLECTIONS=(tracks albums artists customers invoices employees playlists)

log() {
    echo ""
    echo "==> $*"
}

error() {
    echo "ERROR: $*" >&2
    exit 1
}

require_command() {
    local command_name="$1"
    if ! command -v "$command_name" >/dev/null 2>&1; then
        error "$command_name is required for Chinook acceptance verification"
    fi
}

api_get() {
    local path="$1"
    curl -sfS "${BASE_URL}${path}" \
        -H "X-TYPESENSE-API-KEY: ${TYPESENSE_API_KEY}"
}

json_count() {
    local path="$1"
    local python_expression="$2"
    api_get "$path" | python -c 'import json, sys; data = json.load(sys.stdin); print(eval(sys.argv[1], {"__builtins__": {}, "data": data, "len": len, "sum": sum}, {}))' "$python_expression"
}

assert_equal() {
    local label="$1"
    local actual="$2"
    local expected="$3"
    if [ "$actual" != "$expected" ]; then
        error "$label = $actual, expected $expected"
    fi
    echo "OK: $label = $actual"
}

assert_at_least() {
    local label="$1"
    local actual="$2"
    local minimum="$3"
    if [ "$actual" -lt "$minimum" ]; then
        error "$label = $actual, expected at least $minimum"
    fi
    echo "OK: $label = $actual"
}

non_empty_line_count() {
    awk 'NF { count++ } END { print count + 0 }' "$1"
}

build_provider_binary() {
    log "Building terraform-provider-typesense"
    (cd "$PROJECT_ROOT" && go build -o "$(basename "$PROVIDER_BINARY")" .)
}

import_fixture_documents() {
    local collection="$1"
    local file="$DATA_DIR/${collection}.jsonl"
    local failures
    local expected
    local actual

    [ -f "$file" ] || error "Missing Chinook fixture: $file"

    failures="$(curl -sfS -X POST "${BASE_URL}/collections/${collection}/documents/import?action=upsert" \
        -H "X-TYPESENSE-API-KEY: ${TYPESENSE_API_KEY}" \
        -H "Content-Type: text/plain" \
        --data-binary @- < "$file" | python -c '
import json
import sys

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    item = json.loads(line)
    if item.get("success") is not True:
        print(item.get("error") or "unknown import error")
' || true
)"
    if [ -n "$failures" ]; then
        error "Document import failed for $collection: $failures"
    fi

    expected="$(non_empty_line_count "$file")"
    actual="$(json_count "/collections/${collection}" 'data["num_documents"]')"
    assert_equal "documents in $collection" "$actual" "$expected"
}

seed_documents() {
    log "Importing Chinook document fixtures"
    for collection in "${CORE_COLLECTIONS[@]}"; do
        import_fixture_documents "$collection"
    done
}

verify_live_cluster() {
    log "Verifying live Chinook resources"

    local expected_collections=10
    if [ -n "${TEST_OPENAI_API_KEY:-}" ]; then
        expected_collections=11
    fi

    assert_equal "collections" "$(json_count "/collections" 'len(data)')" "$expected_collections"
    assert_equal "aliases" "$(json_count "/aliases" 'len(data["aliases"])')" "6"
    assert_equal "stopwords sets" "$(json_count "/stopwords" 'len(data["stopwords"])')" "3"
    assert_equal "presets" "$(json_count "/presets" 'len(data["presets"])')" "12"
    assert_equal "analytics rules" "$(json_count "/analytics/rules" 'len(data)')" "3"
    assert_equal "stemming dictionaries" "$(json_count "/stemming/dictionaries" 'len(data)')" "1"
    assert_equal "synonym items" "$(json_count "/synonym_sets" 'sum(len(item.get("items") or []) for item in data)')" "20"
    assert_equal "curation items" "$(json_count "/curation_sets" 'sum(len(item.get("items") or []) for item in data)')" "9"
    assert_at_least "api keys" "$(json_count "/keys" 'len(data["keys"])')" "3"
}

run_generate() {
    log "Running generate --include-data against Chinook cluster"

    local cli_generated_dir="$GENERATED_DIR"
    if command -v cygpath >/dev/null 2>&1; then
        cli_generated_dir="$(cygpath -w "$GENERATED_DIR")"
    fi

    case "$EXPORT_ROOT" in
        "$PROJECT_ROOT"/tmp/*) ;;
        *) error "Refusing to remove CHINOOK_EXPORT_DIR outside project tmp/: $EXPORT_ROOT" ;;
    esac

    rm -rf "$EXPORT_ROOT"
    mkdir -p "$EXPORT_ROOT"

    "$PROVIDER_BINARY" generate \
        --host "$TYPESENSE_HOST" \
        --port "$TYPESENSE_PORT" \
        --protocol "$TYPESENSE_PROTOCOL" \
        --api-key "$TYPESENSE_API_KEY" \
        --output "$cli_generated_dir" \
        --include-data
}

generated_resource_count() {
    local resource_type="$1"
    (grep -Rho "resource \"${resource_type}\"" "$GENERATED_DIR"/*.tf 2>/dev/null || true) | wc -l | tr -d '[:space:]'
}

generated_import_count() {
    local resource_type="$1"
    (grep -ho "to = ${resource_type}\\." "$GENERATED_DIR/imports.tf" 2>/dev/null || true) | wc -l | tr -d '[:space:]'
}

assert_generated_resource_at_least() {
    local resource_type="$1"
    local minimum="$2"
    assert_at_least "generated ${resource_type} resources" "$(generated_resource_count "$resource_type")" "$minimum"
    assert_at_least "generated ${resource_type} imports" "$(generated_import_count "$resource_type")" "$minimum"
}

assert_generated_data_file() {
    local collection="$1"
    local minimum="$2"
    local file="$GENERATED_DIR/data/${collection}.jsonl"
    [ -f "$file" ] || error "Missing generated data file: $file"
    assert_at_least "generated documents for $collection" "$(non_empty_line_count "$file")" "$minimum"
}

verify_generated_output() {
    log "Verifying generated Terraform and data export"

    [ -f "$GENERATED_DIR/imports.tf" ] || error "Missing generated imports.tf"

    assert_generated_resource_at_least "typesense_collection" "10"
    assert_generated_resource_at_least "typesense_collection_alias" "6"
    assert_generated_resource_at_least "typesense_stopwords_set" "3"
    assert_generated_resource_at_least "typesense_preset" "12"
    assert_generated_resource_at_least "typesense_analytics_rule" "3"
    assert_generated_resource_at_least "typesense_stemming_dictionary" "1"
    assert_generated_resource_at_least "typesense_synonym" "20"
    assert_generated_resource_at_least "typesense_override" "9"
    assert_generated_resource_at_least "typesense_api_key" "3"

    if grep -R "Terraform resource not yet implemented" "$GENERATED_DIR" >/dev/null 2>&1; then
        error "Generated output still contains placeholder comments for unsupported resources"
    fi

    for collection in "${CORE_COLLECTIONS[@]}"; do
        assert_generated_data_file "$collection" "1"
    done
}

main() {
    require_command curl
    require_command python
    require_command go

    build_provider_binary
    seed_documents
    verify_live_cluster
    run_generate
    verify_generated_output

    log "Chinook generate acceptance verification complete"
    echo "Generated output: $GENERATED_DIR"
}

main "$@"
