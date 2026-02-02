#!/bin/bash
#
# Populate a Typesense cluster with test data for export/import testing
#
# Usage:
#   ./scripts/populate-test-cluster.sh --host=localhost --port=8108 --protocol=http --api-key=xyz
#
# This script creates:
#   - Multiple collections with various field types
#   - Documents with diverse data patterns
#   - Synonyms (multi-way and one-way)
#   - Overrides/Curations (includes, excludes, filters, time-based)
#   - Stopwords sets with different locales
#   - API keys with various permissions

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
HOST="localhost"
PORT="8108"
PROTOCOL="http"
API_KEY=""

# Parse arguments
for arg in "$@"; do
    case $arg in
        --host=*)
            HOST="${arg#*=}"
            ;;
        --port=*)
            PORT="${arg#*=}"
            ;;
        --protocol=*)
            PROTOCOL="${arg#*=}"
            ;;
        --api-key=*)
            API_KEY="${arg#*=}"
            ;;
        --help)
            echo "Usage: $0 --host=HOST --port=PORT --protocol=PROTOCOL --api-key=KEY"
            echo ""
            echo "Options:"
            echo "  --host        Typesense host (default: localhost)"
            echo "  --port        Typesense port (default: 8108)"
            echo "  --protocol    Protocol http or https (default: http)"
            echo "  --api-key     Admin API key (required)"
            exit 0
            ;;
        *)
            echo "Unknown argument: $arg"
            exit 1
            ;;
    esac
done

if [ -z "$API_KEY" ]; then
    echo -e "${RED}Error: --api-key is required${NC}"
    exit 1
fi

BASE_URL="${PROTOCOL}://${HOST}:${PORT}"
CURL_OPTS="-s -H 'X-TYPESENSE-API-KEY: ${API_KEY}' -H 'Content-Type: application/json'"

# Helper function to make API calls
api() {
    local method=$1
    local endpoint=$2
    local data=$3

    if [ -n "$data" ]; then
        curl -s -X "$method" \
            -H "X-TYPESENSE-API-KEY: ${API_KEY}" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "${BASE_URL}${endpoint}"
    else
        curl -s -X "$method" \
            -H "X-TYPESENSE-API-KEY: ${API_KEY}" \
            "${BASE_URL}${endpoint}"
    fi
}

# Helper to import JSONL documents
import_jsonl() {
    local collection=$1
    local data=$2

    echo "$data" | curl -s -X POST \
        -H "X-TYPESENSE-API-KEY: ${API_KEY}" \
        -H "Content-Type: text/plain" \
        --data-binary @- \
        "${BASE_URL}/collections/${collection}/documents/import?action=upsert"
}

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Populating Typesense Test Cluster${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "Host: ${BASE_URL}"
echo ""

# Check connection
echo -e "${YELLOW}Checking connection...${NC}"
HEALTH=$(api GET "/health")
if echo "$HEALTH" | grep -q "ok"; then
    echo -e "${GREEN}✓ Connected to Typesense${NC}"
else
    echo -e "${RED}✗ Failed to connect to Typesense${NC}"
    echo "$HEALTH"
    exit 1
fi

# =============================================================================
# COLLECTION 1: Products (comprehensive field types)
# =============================================================================
echo ""
echo -e "${YELLOW}Creating collection: products${NC}"

api DELETE "/collections/products" > /dev/null 2>&1 || true

api POST "/collections" '{
    "name": "products",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "title", "type": "string", "facet": true, "infix": true},
        {"name": "description", "type": "string", "optional": true},
        {"name": "brand", "type": "string", "facet": true},
        {"name": "category", "type": "string", "facet": true},
        {"name": "tags", "type": "string[]", "facet": true, "optional": true},
        {"name": "price", "type": "float", "facet": true},
        {"name": "rating", "type": "float", "facet": true, "optional": true},
        {"name": "stock_count", "type": "int32", "facet": true},
        {"name": "views", "type": "int64", "optional": true},
        {"name": "is_active", "type": "bool", "facet": true},
        {"name": "created_at", "type": "int64"},
        {"name": "location", "type": "geopoint", "optional": true}
    ],
    "default_sorting_field": "created_at",
    "token_separators": ["-", "_"],
    "symbols_to_index": ["#", "@"]
}'
echo ""

echo -e "${YELLOW}Importing products documents...${NC}"
PRODUCTS_DATA='{"id":"prod-001","title":"Wireless Bluetooth Headphones","description":"Premium noise-cancelling headphones with 30-hour battery life","brand":"AudioTech","category":"Electronics","tags":["audio","wireless","bluetooth"],"price":149.99,"rating":4.5,"stock_count":250,"views":15420,"is_active":true,"created_at":1704067200,"location":[37.7749,-122.4194]}
{"id":"prod-002","title":"Ergonomic Office Chair","description":"Adjustable lumbar support with breathable mesh back","brand":"ComfortPro","category":"Furniture","tags":["office","ergonomic","seating"],"price":349.99,"rating":4.8,"stock_count":75,"views":8932,"is_active":true,"created_at":1704153600,"location":[40.7128,-74.0060]}
{"id":"prod-003","title":"Smart Watch Series X","description":"Fitness tracking, heart rate monitor, GPS","brand":"TechGear","category":"Electronics","tags":["wearable","fitness","smart"],"price":299.99,"rating":4.2,"stock_count":180,"views":22150,"is_active":true,"created_at":1704240000}
{"id":"prod-004","title":"Organic Green Tea","description":"Premium loose leaf tea from Japanese highlands","brand":"TeaLeaf","category":"Food & Beverage","tags":["organic","tea","healthy"],"price":24.99,"rating":4.9,"stock_count":500,"views":3421,"is_active":true,"created_at":1704326400}
{"id":"prod-005","title":"Vintage Leather Jacket","description":"Classic biker style genuine leather","brand":"UrbanStyle","category":"Clothing","tags":["leather","vintage","jacket"],"price":199.99,"rating":4.6,"stock_count":45,"views":12890,"is_active":true,"created_at":1704412800,"location":[51.5074,-0.1278]}
{"id":"prod-006","title":"Professional DSLR Camera","description":"24MP sensor with 4K video recording","brand":"PhotoPro","category":"Electronics","tags":["camera","photography","professional"],"price":1299.99,"rating":4.7,"stock_count":30,"views":45200,"is_active":true,"created_at":1704499200}
{"id":"prod-007","title":"Bamboo Cutting Board Set","description":"Eco-friendly kitchen cutting boards, set of 3","brand":"EcoKitchen","category":"Home & Kitchen","tags":["bamboo","eco-friendly","kitchen"],"price":34.99,"rating":4.4,"stock_count":200,"views":5670,"is_active":true,"created_at":1704585600}
{"id":"prod-008","title":"Running Shoes Pro","description":"Lightweight with advanced cushioning technology","brand":"SportMax","category":"Sports","tags":["running","athletic","shoes"],"price":129.99,"rating":4.3,"stock_count":150,"views":18900,"is_active":true,"created_at":1704672000,"location":[34.0522,-118.2437]}
{"id":"prod-009","title":"Stainless Steel Water Bottle","description":"Double-wall insulated, keeps drinks cold 24hrs","brand":"HydroLife","category":"Sports","tags":["hydration","eco-friendly","insulated"],"price":29.99,"rating":4.8,"stock_count":800,"views":9540,"is_active":true,"created_at":1704758400}
{"id":"prod-010","title":"Mechanical Gaming Keyboard","description":"RGB backlit with Cherry MX switches","brand":"GameTech","category":"Electronics","tags":["gaming","keyboard","mechanical"],"price":149.99,"rating":4.6,"stock_count":120,"views":31200,"is_active":true,"created_at":1704844800}
{"id":"prod-011","title":"Discontinued Widget","description":"This product is no longer available","brand":"OldBrand","category":"Misc","price":9.99,"stock_count":0,"is_active":false,"created_at":1609459200}
{"id":"prod-012","title":"Yoga Mat Premium","description":"Extra thick non-slip surface for comfort","brand":"ZenFit","category":"Sports","tags":["yoga","fitness","mat"],"price":49.99,"rating":4.7,"stock_count":300,"views":7800,"is_active":true,"created_at":1704931200}'

import_jsonl "products" "$PRODUCTS_DATA"
echo -e "${GREEN}✓ Imported 12 products${NC}"

# =============================================================================
# COLLECTION 2: Users (nested objects, auto fields)
# =============================================================================
echo ""
echo -e "${YELLOW}Creating collection: users${NC}"

api DELETE "/collections/users" > /dev/null 2>&1 || true

api POST "/collections" '{
    "name": "users",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "username", "type": "string", "facet": true},
        {"name": "email", "type": "string"},
        {"name": "full_name", "type": "string", "infix": true},
        {"name": "age", "type": "int32", "facet": true, "optional": true},
        {"name": "country", "type": "string", "facet": true},
        {"name": "interests", "type": "string[]", "facet": true, "optional": true},
        {"name": "signup_date", "type": "int64"},
        {"name": "is_verified", "type": "bool", "facet": true},
        {"name": "profile_score", "type": "float", "optional": true}
    ],
    "default_sorting_field": "signup_date"
}'
echo ""

echo -e "${YELLOW}Importing users documents...${NC}"
USERS_DATA='{"id":"user-001","username":"techguru42","email":"tech@example.com","full_name":"Alex Thompson","age":32,"country":"USA","interests":["technology","gaming","photography"],"signup_date":1704067200,"is_verified":true,"profile_score":92.5}
{"id":"user-002","username":"naturelover","email":"nature@example.com","full_name":"Emma Wilson","age":28,"country":"Canada","interests":["hiking","photography","travel"],"signup_date":1704153600,"is_verified":true,"profile_score":88.0}
{"id":"user-003","username":"bookworm99","email":"books@example.com","full_name":"James Chen","age":45,"country":"UK","interests":["reading","writing","history"],"signup_date":1704240000,"is_verified":false,"profile_score":75.5}
{"id":"user-004","username":"fitnessfan","email":"fit@example.com","full_name":"Sarah Martinez","age":26,"country":"USA","interests":["fitness","nutrition","yoga"],"signup_date":1704326400,"is_verified":true,"profile_score":95.0}
{"id":"user-005","username":"musicmaster","email":"music@example.com","full_name":"David Kim","age":35,"country":"South Korea","interests":["music","piano","concerts"],"signup_date":1704412800,"is_verified":true,"profile_score":89.5}
{"id":"user-006","username":"chef_marie","email":"chef@example.com","full_name":"Marie Dubois","age":41,"country":"France","interests":["cooking","wine","travel"],"signup_date":1704499200,"is_verified":true,"profile_score":91.0}
{"id":"user-007","username":"codingwiz","email":"code@example.com","full_name":"Michael Brown","country":"Germany","interests":["programming","AI","startups"],"signup_date":1704585600,"is_verified":false,"profile_score":82.0}
{"id":"user-008","username":"artsy_anna","email":"art@example.com","full_name":"Anna Petrov","age":29,"country":"Russia","interests":["art","design","museums"],"signup_date":1704672000,"is_verified":true}'

import_jsonl "users" "$USERS_DATA"
echo -e "${GREEN}✓ Imported 8 users${NC}"

# =============================================================================
# COLLECTION 3: Articles (text-heavy, locale support)
# =============================================================================
echo ""
echo -e "${YELLOW}Creating collection: articles${NC}"

api DELETE "/collections/articles" > /dev/null 2>&1 || true

api POST "/collections" '{
    "name": "articles",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "title", "type": "string", "locale": "en"},
        {"name": "content", "type": "string", "locale": "en"},
        {"name": "author", "type": "string", "facet": true},
        {"name": "category", "type": "string", "facet": true},
        {"name": "tags", "type": "string[]", "facet": true},
        {"name": "published_at", "type": "int64"},
        {"name": "word_count", "type": "int32"},
        {"name": "is_featured", "type": "bool", "facet": true}
    ],
    "default_sorting_field": "published_at"
}'
echo ""

echo -e "${YELLOW}Importing articles documents...${NC}"
ARTICLES_DATA='{"id":"art-001","title":"Getting Started with Machine Learning","content":"Machine learning is transforming how we interact with technology. In this comprehensive guide, we explore the fundamentals of ML algorithms and their practical applications in everyday software development.","author":"Dr. Smith","category":"Technology","tags":["machine-learning","AI","tutorial"],"published_at":1704067200,"word_count":2500,"is_featured":true}
{"id":"art-002","title":"The Future of Remote Work","content":"As companies adapt to new working models, remote work has become a permanent fixture. This article examines the challenges and opportunities presented by distributed teams.","author":"Jane Doe","category":"Business","tags":["remote-work","productivity","future"],"published_at":1704153600,"word_count":1800,"is_featured":false}
{"id":"art-003","title":"Sustainable Living Tips","content":"Small changes in daily habits can make a big environmental impact. Learn practical tips for reducing your carbon footprint and living more sustainably.","author":"Green Guide","category":"Lifestyle","tags":["sustainability","environment","tips"],"published_at":1704240000,"word_count":1200,"is_featured":true}
{"id":"art-004","title":"Understanding Blockchain Technology","content":"Blockchain is more than just cryptocurrency. Discover how this distributed ledger technology is revolutionizing industries from finance to healthcare.","author":"Tech Weekly","category":"Technology","tags":["blockchain","cryptocurrency","innovation"],"published_at":1704326400,"word_count":3200,"is_featured":false}
{"id":"art-005","title":"Healthy Meal Prep for Busy Professionals","content":"Save time and eat better with these meal prep strategies designed for people with demanding schedules. Includes recipes and shopping lists.","author":"Nutrition Now","category":"Health","tags":["nutrition","meal-prep","health"],"published_at":1704412800,"word_count":2100,"is_featured":true}'

import_jsonl "articles" "$ARTICLES_DATA"
echo -e "${GREEN}✓ Imported 5 articles${NC}"

# =============================================================================
# COLLECTION 4: Events (with geopoint arrays, dates)
# =============================================================================
echo ""
echo -e "${YELLOW}Creating collection: events${NC}"

api DELETE "/collections/events" > /dev/null 2>&1 || true

api POST "/collections" '{
    "name": "events",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "name", "type": "string", "infix": true},
        {"name": "description", "type": "string", "optional": true},
        {"name": "venue", "type": "string", "facet": true},
        {"name": "city", "type": "string", "facet": true},
        {"name": "event_type", "type": "string", "facet": true},
        {"name": "start_date", "type": "int64"},
        {"name": "end_date", "type": "int64", "optional": true},
        {"name": "ticket_price", "type": "float", "facet": true},
        {"name": "capacity", "type": "int32"},
        {"name": "is_virtual", "type": "bool", "facet": true},
        {"name": "location", "type": "geopoint"}
    ],
    "default_sorting_field": "start_date"
}'
echo ""

echo -e "${YELLOW}Importing events documents...${NC}"
EVENTS_DATA='{"id":"evt-001","name":"Tech Conference 2024","description":"Annual technology conference featuring industry leaders","venue":"Convention Center","city":"San Francisco","event_type":"Conference","start_date":1709251200,"end_date":1709424000,"ticket_price":499.99,"capacity":5000,"is_virtual":false,"location":[37.7749,-122.4194]}
{"id":"evt-002","name":"Virtual AI Workshop","description":"Hands-on workshop on building AI applications","venue":"Online","city":"Virtual","event_type":"Workshop","start_date":1709510400,"ticket_price":49.99,"capacity":500,"is_virtual":true,"location":[0,0]}
{"id":"evt-003","name":"Summer Music Festival","description":"Three days of live music performances","venue":"Central Park","city":"New York","event_type":"Festival","start_date":1717200000,"end_date":1717459200,"ticket_price":199.99,"capacity":20000,"is_virtual":false,"location":[40.7829,-73.9654]}
{"id":"evt-004","name":"Startup Pitch Night","description":"Watch startups pitch to investors","venue":"Innovation Hub","city":"Austin","event_type":"Networking","start_date":1710288000,"ticket_price":25.00,"capacity":200,"is_virtual":false,"location":[30.2672,-97.7431]}
{"id":"evt-005","name":"Photography Masterclass","venue":"Art Gallery","city":"London","event_type":"Workshop","start_date":1711152000,"ticket_price":150.00,"capacity":50,"is_virtual":false,"location":[51.5074,-0.1278]}'

import_jsonl "events" "$EVENTS_DATA"
echo -e "${GREEN}✓ Imported 5 events${NC}"

# =============================================================================
# COLLECTION 5: Books (with nested fields enabled)
# =============================================================================
echo ""
echo -e "${YELLOW}Creating collection: books${NC}"

api DELETE "/collections/books" > /dev/null 2>&1 || true

api POST "/collections" '{
    "name": "books",
    "enable_nested_fields": true,
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "title", "type": "string", "infix": true},
        {"name": "author", "type": "object"},
        {"name": "author.name", "type": "string", "facet": true},
        {"name": "author.nationality", "type": "string", "facet": true, "optional": true},
        {"name": "genre", "type": "string[]", "facet": true},
        {"name": "publication_year", "type": "int32", "facet": true},
        {"name": "pages", "type": "int32"},
        {"name": "rating", "type": "float", "facet": true},
        {"name": "in_stock", "type": "bool", "facet": true},
        {"name": "price", "type": "float"}
    ],
    "default_sorting_field": "publication_year"
}'
echo ""

echo -e "${YELLOW}Importing books documents...${NC}"
BOOKS_DATA='{"id":"book-001","title":"The Art of Programming","author":{"name":"Robert Martin","nationality":"American"},"genre":["Technology","Education"],"publication_year":2020,"pages":450,"rating":4.8,"in_stock":true,"price":49.99}
{"id":"book-002","title":"Mystery at Midnight","author":{"name":"Agatha Stone","nationality":"British"},"genre":["Mystery","Fiction"],"publication_year":2019,"pages":320,"rating":4.5,"in_stock":true,"price":14.99}
{"id":"book-003","title":"Journey to the Stars","author":{"name":"Isaac Nova"},"genre":["Science Fiction","Adventure"],"publication_year":2021,"pages":580,"rating":4.7,"in_stock":false,"price":24.99}
{"id":"book-004","title":"Cooking with Love","author":{"name":"Julia Romano","nationality":"Italian"},"genre":["Cooking","Lifestyle"],"publication_year":2022,"pages":280,"rating":4.9,"in_stock":true,"price":34.99}
{"id":"book-005","title":"History of Ancient Civilizations","author":{"name":"Dr. Helen Troy","nationality":"Greek"},"genre":["History","Education"],"publication_year":2018,"pages":720,"rating":4.6,"in_stock":true,"price":39.99}
{"id":"book-006","title":"Modern Poetry Collection","author":{"name":"Various Authors"},"genre":["Poetry","Literature"],"publication_year":2023,"pages":200,"rating":4.3,"in_stock":true,"price":19.99}'

import_jsonl "books" "$BOOKS_DATA"
echo -e "${GREEN}✓ Imported 6 books${NC}"

# =============================================================================
# COLLECTION 6: Empty collection (schema only, no documents)
# =============================================================================
echo ""
echo -e "${YELLOW}Creating collection: empty_collection (schema only)${NC}"

api DELETE "/collections/empty_collection" > /dev/null 2>&1 || true

api POST "/collections" '{
    "name": "empty_collection",
    "fields": [
        {"name": "id", "type": "string"},
        {"name": "name", "type": "string"},
        {"name": "value", "type": "int32"}
    ]
}'
echo ""
echo -e "${GREEN}✓ Created empty collection (no documents)${NC}"

# =============================================================================
# SYNONYMS
# =============================================================================
echo ""
echo -e "${YELLOW}Creating synonyms...${NC}"

# Multi-way synonyms for products
api PUT "/collections/products/synonyms/clothing-synonyms" '{
    "synonyms": ["blazer", "coat", "jacket", "outerwear"]
}'
echo ""

api PUT "/collections/products/synonyms/audio-synonyms" '{
    "synonyms": ["headphones", "earphones", "earbuds", "headset"]
}'
echo ""

# One-way synonym (root)
api PUT "/collections/products/synonyms/pants-synonyms" '{
    "root": "pants",
    "synonyms": ["trousers", "jeans", "slacks", "chinos"]
}'
echo ""

# Synonyms for articles
api PUT "/collections/articles/synonyms/tech-synonyms" '{
    "synonyms": ["AI", "artificial intelligence", "machine learning", "ML"]
}'
echo ""

api PUT "/collections/articles/synonyms/work-synonyms" '{
    "root": "remote work",
    "synonyms": ["work from home", "WFH", "telecommuting", "distributed work"]
}'
echo ""

echo -e "${GREEN}✓ Created 5 synonym sets${NC}"

# =============================================================================
# OVERRIDES / CURATIONS
# =============================================================================
echo ""
echo -e "${YELLOW}Creating overrides/curations...${NC}"

# Override with includes (pin documents)
api PUT "/collections/products/overrides/featured-electronics" '{
    "rule": {
        "query": "electronics",
        "match": "contains"
    },
    "includes": [
        {"id": "prod-006", "position": 1},
        {"id": "prod-003", "position": 2}
    ]
}'
echo ""

# Override with excludes (hide documents)
api PUT "/collections/products/overrides/hide-discontinued" '{
    "rule": {
        "query": "*",
        "match": "exact"
    },
    "excludes": [
        {"id": "prod-011"}
    ]
}'
echo ""

# Override with filter_by
api PUT "/collections/products/overrides/sports-filter" '{
    "rule": {
        "query": "fitness",
        "match": "contains"
    },
    "filter_by": "category:Sports"
}'
echo ""

# Override with replace_query
api PUT "/collections/products/overrides/typo-correction" '{
    "rule": {
        "query": "headfones",
        "match": "exact"
    },
    "replace_query": "headphones"
}'
echo ""

# Override with sort_by
api PUT "/collections/products/overrides/price-sort" '{
    "rule": {
        "query": "cheap",
        "match": "contains"
    },
    "sort_by": "price:asc",
    "remove_matched_tokens": true
}'
echo ""

# Time-based override (effective dates)
FUTURE_TS=$(($(date +%s) + 86400 * 30))  # 30 days from now
api PUT "/collections/products/overrides/seasonal-promotion" '{
    "rule": {
        "query": "summer",
        "match": "contains"
    },
    "includes": [
        {"id": "prod-008", "position": 1}
    ],
    "effective_from_ts": '"$(date +%s)"',
    "effective_to_ts": '"$FUTURE_TS"'
}'
echo ""

# Override with tags
api PUT "/collections/articles/overrides/featured-articles" '{
    "rule": {
        "query": "*",
        "match": "exact",
        "tags": ["homepage", "featured"]
    },
    "includes": [
        {"id": "art-001", "position": 1},
        {"id": "art-003", "position": 2}
    ],
    "filter_curated_hits": true
}'
echo ""

# Override with stop_processing
api PUT "/collections/articles/overrides/priority-override" '{
    "rule": {
        "query": "technology",
        "match": "contains"
    },
    "includes": [
        {"id": "art-004", "position": 1}
    ],
    "stop_processing": true
}'
echo ""

echo -e "${GREEN}✓ Created 8 overrides${NC}"

# =============================================================================
# STOPWORDS
# =============================================================================
echo ""
echo -e "${YELLOW}Creating stopwords sets...${NC}"

api PUT "/stopwords/english-stopwords" '{
    "stopwords": ["the", "a", "an", "and", "or", "but", "is", "are", "was", "were", "be", "been", "being", "have", "has", "had", "do", "does", "did", "will", "would", "could", "should"],
    "locale": "en"
}'
echo ""

api PUT "/stopwords/german-stopwords" '{
    "stopwords": ["der", "die", "das", "ein", "eine", "und", "oder", "aber", "ist", "sind", "war", "waren", "sein", "haben", "hat", "hatte"],
    "locale": "de"
}'
echo ""

api PUT "/stopwords/french-stopwords" '{
    "stopwords": ["le", "la", "les", "un", "une", "et", "ou", "mais", "est", "sont", "etait", "etre", "avoir", "a", "avait"],
    "locale": "fr"
}'
echo ""

api PUT "/stopwords/common-words" '{
    "stopwords": ["very", "really", "just", "only", "also", "even", "still", "already"]
}'
echo ""

echo -e "${GREEN}✓ Created 4 stopwords sets${NC}"

# =============================================================================
# API KEYS
# =============================================================================
echo ""
echo -e "${YELLOW}Creating API keys...${NC}"

# Search-only key for all collections
SEARCH_KEY=$(api POST "/keys" '{
    "description": "Search-only key for all collections",
    "actions": ["documents:search"],
    "collections": ["*"]
}')
echo "Search key: $(echo "$SEARCH_KEY" | grep -o '"value":"[^"]*"' | head -1)"

# Read-only key for products
PRODUCTS_KEY=$(api POST "/keys" '{
    "description": "Read-only key for products collection",
    "actions": ["documents:search", "documents:get"],
    "collections": ["products"]
}')
echo "Products key: $(echo "$PRODUCTS_KEY" | grep -o '"value":"[^"]*"' | head -1)"

# Key for articles with expiration (30 days from now)
EXPIRES_AT=$(($(date +%s) + 86400 * 30))
ARTICLES_KEY=$(api POST "/keys" '{
    "description": "Temporary key for articles - expires in 30 days",
    "actions": ["documents:search", "documents:get"],
    "collections": ["articles"],
    "expires_at": '"$EXPIRES_AT"'
}')
echo "Articles key (expires): $(echo "$ARTICLES_KEY" | grep -o '"value":"[^"]*"' | head -1)"

# Admin key for specific collections
ADMIN_KEY=$(api POST "/keys" '{
    "description": "Admin key for users and events collections",
    "actions": ["*"],
    "collections": ["users", "events"]
}')
echo "Admin key: $(echo "$ADMIN_KEY" | grep -o '"value":"[^"]*"' | head -1)"

echo -e "${GREEN}✓ Created 4 API keys${NC}"

# =============================================================================
# SUMMARY
# =============================================================================
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Test Cluster Population Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Created resources:"
echo "  - 6 collections (products, users, articles, events, books, empty_collection)"
echo "  - 36 documents total"
echo "  - 5 synonym sets"
echo "  - 8 overrides/curations"
echo "  - 4 stopwords sets"
echo "  - 4 API keys"
echo ""
echo "Test features covered:"
echo "  ✓ Various field types (string, int32, int64, float, bool, geopoint, arrays)"
echo "  ✓ Field attributes (facet, optional, index, sort, infix, locale)"
echo "  ✓ Token separators and symbols_to_index"
echo "  ✓ Nested fields (books collection)"
echo "  ✓ Empty collection (schema only)"
echo "  ✓ Multi-way and one-way synonyms"
echo "  ✓ Overrides with includes, excludes, filter_by, sort_by, replace_query"
echo "  ✓ Time-based overrides (effective_from/to)"
echo "  ✓ Tag-based override rules"
echo "  ✓ Stopwords with different locales"
echo "  ✓ API keys with various permissions and expiration"
echo ""
echo "You can now test export/import with:"
echo "  ./terraform-provider-typesense generate --host=${HOST} --port=${PORT} --protocol=${PROTOCOL} --api-key=\$API_KEY --output=./test-export"
echo ""
