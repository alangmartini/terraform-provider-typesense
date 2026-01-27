# E2E Testbed for Typesense Terraform Provider

This testbed provides a complete environment for running end-to-end tests of the Typesense Terraform provider, including migration between clusters.

## Architecture

```
testbed/
├── docker-compose.yml          # Source (8108) + Target (8109) clusters
├── fixtures/
│   ├── generate-fixtures.go    # Go program to generate test documents
│   └── schemas/                # Collection schema definitions
│       ├── products.json
│       ├── users.json
│       ├── articles.json
│       ├── events.json
│       └── edge_cases.json
├── scripts/
│   ├── seed-source.sh          # Populate source cluster
│   ├── verify-migration.sh     # Verify target matches source
│   └── run-e2e-test.sh         # Full orchestration
└── README.md
```

## Quick Start

From the project root:

```bash
# Start both Typesense clusters
make testbed-up

# Seed the source cluster with test data (~50k documents)
make testbed-seed

# Run the complete E2E test workflow
make testbed-e2e

# Verify current state (after manual operations)
make testbed-verify

# Stop and clean up
make testbed-down
```

## Clusters

| Cluster | Port | API Key | Purpose |
|---------|------|---------|---------|
| Source | 8108 | `source-test-api-key` | Original data |
| Target | 8109 | `target-test-api-key` | Migration destination |

## Collections

### products
Tests: facets, geopoints, optional fields, arrays, numeric types
- ~10,000 documents
- Fields: name, description, brand, categories[], price, location (geo), stock info

### users
Tests: unicode diversity, locales, roles/permissions arrays
- ~10,000 documents
- Fields: display_name (unicode), bio, locale, roles[], permissions[]

### articles
Tests: large text fields (up to 100KB), markdown content
- ~10,000 documents
- Fields: content (large), content_markdown, word_count, reading_time

### events
Tests: geopoints, timestamps, date ranges
- ~10,000 documents
- Fields: location (geo), start/end timestamps, timezone

### edge_cases
Tests: boundary conditions, special characters, extreme values
- ~1,000 documents
- Fields: unicode, special chars, large arrays (1000 elements), float precision

## Edge Cases Covered

| Category | Examples |
|----------|----------|
| Unicode | CJK (中文, 日本語, 한국어), Cyrillic (Русский), Arabic (العربية), Hindi (हिंदी) |
| Special chars | Newlines, tabs, backslashes, JSON escapes, SQL injection attempts |
| Size | 100KB text fields, 1000-element arrays |
| Numbers | Float precision (0.1+0.2), max int32, large int64 |
| Geo | Near poles (lat ±89), near dateline (lon ±179) |
| Empty | Empty strings, all optional fields null |

## E2E Test Workflow

The `run-e2e-test.sh` script orchestrates:

1. **Start clusters** - Docker Compose up with health checks
2. **Seed source** - Generate and import ~50k documents
3. **Generate export** - `./terraform-provider-typesense generate --include-data`
4. **Migrate** - `./terraform-provider-typesense migrate` to target
5. **Verify** - Compare document counts, schemas, sample data
6. **Terraform plan** - Should show no changes

## Environment Variables

```bash
# Source cluster
SOURCE_HOST=localhost
SOURCE_PORT=8108
SOURCE_API_KEY=source-test-api-key

# Target cluster
TARGET_HOST=localhost
TARGET_PORT=8109
TARGET_API_KEY=target-test-api-key

# Document counts (for faster testing)
PRODUCTS_COUNT=10000
USERS_COUNT=10000
ARTICLES_COUNT=10000
EVENTS_COUNT=10000
EDGE_CASES_COUNT=1000

# Verification
SAMPLE_SIZE=10  # Documents to sample per collection
```

## Manual Testing

### Generate fixtures only

```bash
cd testbed/fixtures
go run generate-fixtures.go products 100 > products.jsonl
go run generate-fixtures.go edge_cases 50 > edge_cases.jsonl
```

### Quick verification

```bash
# Check source cluster
curl http://localhost:8108/collections \
  -H "X-TYPESENSE-API-KEY: source-test-api-key" | jq

# Check target cluster
curl http://localhost:8109/collections \
  -H "X-TYPESENSE-API-KEY: target-test-api-key" | jq
```

### Reduced dataset for development

```bash
# Faster seeding with fewer documents
PRODUCTS_COUNT=100 \
USERS_COUNT=100 \
ARTICLES_COUNT=100 \
EVENTS_COUNT=100 \
EDGE_CASES_COUNT=50 \
  make testbed-seed
```

## Troubleshooting

### Clusters won't start
```bash
# Check container logs
docker compose -f testbed/docker-compose.yml logs

# Check ports
lsof -i :8108
lsof -i :8109
```

### Seeding fails
```bash
# Check cluster health
curl http://localhost:8108/health

# Check Go version (requires 1.18+)
go version
```

### Migration fails
```bash
# Ensure export directory exists
ls -la testbed/export/

# Check binary is built
ls -la terraform-provider-typesense
```
