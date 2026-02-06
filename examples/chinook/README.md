# Chinook Database on Typesense Cloud

This example demonstrates how to use the Typesense Terraform provider to create a search infrastructure for the [Chinook Database](https://github.com/lerocha/chinook-database) - a sample music store database commonly used for learning SQL.

## Overview

The Chinook database models a digital media store with:
- **Artists, Albums, and Tracks** - Music catalog
- **Customers and Employees** - Business entities
- **Invoices** - Purchase history
- **Playlists** - User-created collections

## Denormalization Strategy

Since Typesense is a search-optimized document store (not a relational database), this example uses **denormalization** - embedding related data directly into documents for optimal search performance.

### Collection Mapping

| Collection | Source Tables | Purpose |
|------------|---------------|---------|
| `tracks` | Track + Album + Artist + Genre + MediaType | Primary music search |
| `albums` | Album + Artist + aggregated track stats | Album browsing |
| `artists` | Artist + aggregated stats | Artist search/autocomplete |
| `customers` | Customer + Employee (support rep) | Customer lookup |
| `invoices` | Invoice + InvoiceLine + Track + Customer | Order history |
| `employees` | Employee (with manager embedded) | Staff directory |
| `playlists` | Playlist + PlaylistTrack + Track | Playlist search |

### Tables Embedded (Not Separate Collections)
- **Genre** - Embedded in tracks as `genre_name`
- **MediaType** - Embedded in tracks as `media_type_name`
- **PlaylistTrack** - Embedded in playlists as `tracks[]`
- **InvoiceLine** - Embedded in invoices as `line_items[]`

## Prerequisites

1. A Typesense Cloud account with an active cluster
2. Admin API key for your Typesense Cloud cluster
3. Terraform >= 1.0

## Usage

### 1. Configure Variables

Create a `terraform.tfvars` file (do not commit to version control):

```hcl
typesense_host    = "xxx.a1.typesense.net"
typesense_api_key = "your-admin-api-key"
```

Or use environment variables:

```bash
export TF_VAR_typesense_host="xxx.a1.typesense.net"
export TF_VAR_typesense_api_key="your-admin-api-key"
```

### 2. Initialize and Apply

```bash
terraform init
terraform plan
terraform apply
```

### 3. Verify Collections

After applying, verify collections via the Typesense API:

```bash
curl "https://${TF_VAR_typesense_host}/collections" \
  -H "X-TYPESENSE-API-KEY: ${TF_VAR_typesense_api_key}"
```

## Collection Schemas

### tracks (Primary Search)

**Search fields**: `name`, `composer`, `album_title`, `artist_name` (all with infix search)
**Facets**: `artist_name`, `genre_name`, `media_type_name`, `unit_price`
**Sortable**: `milliseconds`, `unit_price`, `popularity_score`

Example document:
```json
{
  "id": "1",
  "name": "For Those About To Rock",
  "composer": "Angus Young",
  "milliseconds": 343719,
  "unit_price": 0.99,
  "album_id": "1",
  "album_title": "For Those About To Rock",
  "artist_id": "1",
  "artist_name": "AC/DC",
  "genre_id": "1",
  "genre_name": "Rock",
  "media_type_id": "1",
  "media_type_name": "MPEG audio file",
  "popularity_score": 85
}
```

### albums

**Search fields**: `title`, `artist_name` (infix)
**Facets**: `artist_name`, `genres[]`, `release_year`
**Sortable**: `track_count`, `total_duration_seconds`, `release_year`

Example document:
```json
{
  "id": "1",
  "title": "For Those About To Rock We Salute You",
  "artist_id": "1",
  "artist_name": "AC/DC",
  "genres": ["Rock"],
  "track_count": 10,
  "total_duration_seconds": 2400,
  "release_year": 1981
}
```

### artists

**Search fields**: `name` (infix for autocomplete)
**Facets**: `genres[]`
**Sortable**: `album_count`, `track_count`

### customers

**Search fields**: `full_name`, `email`, `company`
**Facets**: `company`, `city`, `state`, `country`
**Sortable**: `total_purchases`, `invoice_count`

The `support_rep` field is an embedded object:
```json
{
  "support_rep": {
    "id": "3",
    "full_name": "Jane Peacock",
    "email": "jane@chinookcorp.com"
  }
}
```

### invoices

**Search fields**: `customer_name`, `track_names[]`
**Facets**: `billing_city`, `billing_state`, `billing_country`, `total`
**Sortable**: `invoice_date`, `total`

The `line_items` field contains embedded track purchase details:
```json
{
  "line_items": [
    {
      "track_id": "1",
      "track_name": "For Those About To Rock",
      "unit_price": 0.99,
      "quantity": 1
    }
  ]
}
```

### employees

**Search fields**: `full_name`, `email`
**Facets**: `title`, `city`, `country`
**Sortable**: `hire_date`, `customers_supported`

The `manager` field is an embedded object for self-referencing hierarchy.

### playlists

**Search fields**: `name`, `track_names[]`
**Facets**: `artists[]`, `genres[]`
**Sortable**: `track_count`, `total_duration_seconds`

## Synonyms

The configuration includes music-related synonyms to improve search:

- **Genre synonyms**: "rock" = "rock and roll", "hip-hop" = "rap", etc.
- **Media type synonyms**: "mp3" -> "MPEG audio file"
- **Artist synonyms**: "ac/dc" = "acdc"

## Sample Queries

### Search tracks by title with artist faceting
```bash
curl "https://${TF_VAR_typesense_host}/collections/tracks/documents/search" \
  -H "X-TYPESENSE-API-KEY: ${TF_VAR_typesense_api_key}" \
  -d '{
    "q": "rock",
    "query_by": "name,album_title,artist_name",
    "facet_by": "artist_name,genre_name",
    "sort_by": "popularity_score:desc"
  }'
```

### Autocomplete artists
```bash
curl "https://${TF_VAR_typesense_host}/collections/artists/documents/search" \
  -H "X-TYPESENSE-API-KEY: ${TF_VAR_typesense_api_key}" \
  -d '{
    "q": "ac",
    "query_by": "name",
    "prefix": "true"
  }'
```

### Find customer invoices
```bash
curl "https://${TF_VAR_typesense_host}/collections/invoices/documents/search" \
  -H "X-TYPESENSE-API-KEY: ${TF_VAR_typesense_api_key}" \
  -d '{
    "q": "John",
    "query_by": "customer_name",
    "sort_by": "invoice_date:desc"
  }'
```

## Cleanup

To destroy all collections:

```bash
terraform destroy
```

## Notes

- All ID fields use `string` type for flexibility
- Dates are stored as `int64` Unix timestamps
- Nested objects require `enable_nested_fields = true` on the collection
- `optional = true` is set for nullable fields from the source database
