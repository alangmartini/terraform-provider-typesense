# Typesense Terraform Provider - Implementation Tasks

This document tracks the gaps between the current Terraform provider and the Typesense API spec (v30.0).

**Source of truth:** https://github.com/typesense/typesense-api-spec/blob/master/openapi.yml

---

## High Priority - Vector Search & JOINs

### Task 1: Add vector search field attributes
**Status:** [ ] Not Started

Add missing field attributes required for vector/semantic search:

- [ ] `num_dim` (integer) - Number of dimensions for float[] vector fields
- [ ] `vec_dist` (string) - Distance metric: `cosine` or `ip` (inner product)
- [ ] `embed` (object) - Auto-embedding configuration with:
  - `from` (array of field names to embed)
  - `model_config` (object with model_name, api_key, url, etc.)

**Files to modify:**
- `internal/resources/collection.go` - Add to `CollectionFieldModel` struct and schema
- `internal/client/server_client.go` - Update collection field mapping

**Example usage after implementation:**
```hcl
resource "typesense_collection" "products" {
  name = "products"

  field {
    name    = "embedding"
    type    = "float[]"
    num_dim = 384
    vec_dist = "cosine"
  }

  field {
    name = "description_embedding"
    type = "float[]"
    embed {
      from = ["title", "description"]
      model_config {
        model_name = "ts/all-MiniLM-L12-v2"
      }
    }
  }
}
```

---

### Task 2: Add JOIN field attributes
**Status:** [ ] Not Started

Add missing field attributes required for cross-collection JOINs:

- [ ] `reference` (string) - Reference to field in another collection (format: `collection.field`)
- [ ] `async_reference` (boolean) - Allow indexing even when referenced document doesn't exist

**Files to modify:**
- `internal/resources/collection.go` - Add to `CollectionFieldModel` struct and schema
- `internal/client/server_client.go` - Update collection field mapping

**Example usage after implementation:**
```hcl
resource "typesense_collection" "orders" {
  name = "orders"

  field {
    name      = "product_id"
    type      = "string"
    reference = "products.id"
  }

  field {
    name            = "user_id"
    type            = "string"
    reference       = "users.id"
    async_reference = true  # Allow order creation before user exists
  }
}
```

---

## Medium Priority - Search Quality & Performance

### Task 3: Add stemming field attributes
**Status:** [ ] Not Started

Add missing field attributes for stemming:

- [ ] `stem` (boolean) - Enable algorithmic stemming for the field
- [ ] `stem_dictionary` (string) - Name of custom stemming dictionary to use

**Files to modify:**
- `internal/resources/collection.go` - Add to `CollectionFieldModel` struct and schema
- `internal/client/server_client.go` - Update collection field mapping

**Example usage after implementation:**
```hcl
resource "typesense_collection" "articles" {
  name = "articles"

  field {
    name   = "content"
    type   = "string"
    stem   = true
    locale = "en"
  }

  field {
    name            = "title"
    type            = "string"
    stem_dictionary = "english-plurals"
  }
}
```

---

### Task 4: Add range_index field attribute
**Status:** [ ] Not Started

Add optimization attribute for range queries:

- [ ] `range_index` (boolean) - Enables index optimized for range filtering on numerical fields

**Files to modify:**
- `internal/resources/collection.go` - Add to `CollectionFieldModel` struct and schema
- `internal/client/server_client.go` - Update collection field mapping

**Example usage after implementation:**
```hcl
resource "typesense_collection" "products" {
  name = "products"

  field {
    name        = "price"
    type        = "float"
    range_index = true  # Optimizes queries like price:>=100
  }

  field {
    name        = "rating"
    type        = "float"
    range_index = true  # Optimizes queries like rating:>4.5
  }
}
```

---

### Task 5: Create stemming dictionary resource
**Status:** [ ] Not Started

Create new resource `typesense_stemming_dictionary` for custom word-to-root mappings.

**API endpoints:**
- `GET /stemming/dictionaries` - List all dictionaries
- `GET /stemming/dictionaries/{dictionaryId}` - Retrieve dictionary
- `POST /stemming/dictionaries/import` - Import dictionary (JSONL format)

**Files to create:**
- `internal/resources/stemming_dictionary.go` - New resource implementation

**Files to modify:**
- `internal/client/server_client.go` - Add stemming dictionary client methods
- `internal/provider/provider.go` - Register new resource

**Example usage after implementation:**
```hcl
resource "typesense_stemming_dictionary" "english_plurals" {
  name = "english-plurals"

  # JSONL format: each line is {"word": "x", "root": "y"}
  words = {
    "people"   = "person"
    "children" = "child"
    "mice"     = "mouse"
    "geese"    = "goose"
  }
}

# Reference in collection field
resource "typesense_collection" "articles" {
  name = "articles"

  field {
    name            = "content"
    type            = "string"
    stem_dictionary = typesense_stemming_dictionary.english_plurals.name
  }
}
```

---

## Low Priority - Optimizations & v30 Features

### Task 6: Add store field attribute
**Status:** [ ] Not Started

Add disk storage optimization attribute:

- [ ] `store` (boolean, default: true) - When false, field value not stored on disk

**Files to modify:**
- `internal/resources/collection.go` - Add to `CollectionFieldModel` struct and schema
- `internal/client/server_client.go` - Update collection field mapping

**Example usage after implementation:**
```hcl
resource "typesense_collection" "logs" {
  name = "logs"

  field {
    name  = "search_vector"
    type  = "float[]"
    store = false  # Don't persist to disk, only keep in index
  }
}
```

---

### Task 7: Add field-level token_separators and symbols_to_index
**Status:** [ ] Not Started

Currently these are collection-level only. The API spec shows they can be field-level too.

- [ ] `token_separators` (array of strings) - Per-field token separators
- [ ] `symbols_to_index` (array of strings) - Per-field symbols to index

**Files to modify:**
- `internal/resources/collection.go` - Add to `CollectionFieldModel` struct and schema
- `internal/client/server_client.go` - Update collection field mapping

---

### Task 8: Add drop field attribute for schema updates
**Status:** [ ] Not Started

Add attribute to mark fields for removal during schema updates:

- [ ] `drop` (boolean) - When true, removes the field during collection update

**Files to modify:**
- `internal/resources/collection.go` - Add to `CollectionFieldModel` struct and schema
- `internal/client/server_client.go` - Update collection field mapping

---

### Task 9: Create v30 synonym_set resource
**Status:** [ ] Not Started

Create new top-level resource for v30 synonym sets (shareable across collections).

**API endpoints:**
- `GET /synonym_sets` - List all synonym sets
- `GET/PUT/DELETE /synonym_sets/{synonymSetName}` - Manage synonym set
- `GET /synonym_sets/{synonymSetName}/items` - List items in set
- `GET/PUT/DELETE /synonym_sets/{synonymSetName}/items/{itemId}` - Manage individual synonyms

**Files to create:**
- `internal/resources/synonym_set.go` - New resource for the set
- `internal/resources/synonym_set_item.go` - New resource for items (or nested block)

**Files to modify:**
- `internal/client/server_client.go` - Add synonym set client methods
- `internal/provider/provider.go` - Register new resources

**Note:** Keep existing `typesense_synonym` resource for backward compatibility with v29.

---

### Task 10: Create v30 curation_set resource
**Status:** [ ] Not Started

Create new top-level resource for v30 curation sets (shareable across collections).

**API endpoints:**
- `GET /curation_sets` - List all curation sets
- `GET/PUT/DELETE /curation_sets/{curationSetName}` - Manage curation set
- `GET /curation_sets/{curationSetName}/items` - List items in set
- `GET/PUT/DELETE /curation_sets/{curationSetName}/items/{itemId}` - Manage individual overrides

**Files to create:**
- `internal/resources/curation_set.go` - New resource for the set
- `internal/resources/curation_set_item.go` - New resource for items (or nested block)

**Files to modify:**
- `internal/client/server_client.go` - Add curation set client methods
- `internal/provider/provider.go` - Register new resources

**Note:** Keep existing `typesense_override` resource for backward compatibility with v29.

---

## Not Planned (Operational Endpoints)

These endpoints are operational/runtime and not suitable for Terraform configuration management:

| Endpoint | Reason |
|----------|--------|
| `/operations/snapshot` | One-time operation, not declarative state |
| `/operations/cache/clear` | One-time operation |
| `/operations/db/compact` | One-time operation |
| `/config` | Runtime configuration |
| `/documents/*` | Data management, not infrastructure |
| `/multi_search` | Query operation |
| `/debug`, `/health`, `/metrics.json`, `/stats.json` | Read-only monitoring |

---

## Implementation Notes

### Testing Requirements

For each task:
1. Add the resource/attribute to `examples/chinook/` with realistic examples
2. Run `make chinook-test` to verify integration
3. Add consistency tests if the attribute has server-side defaults
4. Update documentation

### Version Compatibility

- Tasks 1-8: Should work with Typesense v28+
- Tasks 9-10: Require Typesense v30+ (keep legacy resources for v29 compatibility)

### Order of Implementation

Recommended order based on dependencies:

1. **Task 1** (vector fields) - Unlocks semantic search use cases
2. **Task 2** (JOINs) - Unlocks relational patterns
3. **Task 3** (stemming attrs) - Quick win, simple addition
4. **Task 4** (range_index) - Quick win, simple addition
5. **Task 5** (stemming dictionary) - Depends on Task 3
6. **Task 6** (store) - Quick win, simple addition
7. **Task 7** (field-level separators) - Quick win
8. **Task 8** (drop) - Useful for schema migrations
9. **Task 9** (synonym sets) - v30 feature
10. **Task 10** (curation sets) - v30 feature
