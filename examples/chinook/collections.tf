# Chinook Database Collections
# Denormalized schema optimized for search performance

# =============================================================================
# TRACKS COLLECTION (Primary Search)
# Denormalized from: Track + Album + Artist + Genre + MediaType
# =============================================================================
resource "typesense_collection" "tracks" {
  name                 = "tracks"
  enable_nested_fields = true

  # Primary key
  field {
    name = "id"
    type = "string"
  }

  # Track info
  field {
    name  = "name"
    type  = "string"
    infix = true
  }

  field {
    name     = "composer"
    type     = "string"
    infix    = true
    optional = true
  }

  field {
    name = "milliseconds"
    type = "int64"
    sort = true
  }

  field {
    name     = "bytes"
    type     = "int64"
    optional = true
  }

  field {
    name  = "unit_price"
    type  = "float"
    facet = true
    sort  = true
  }

  # Embedded album info (denormalized)
  field {
    name = "album_id"
    type = "string"
  }

  field {
    name  = "album_title"
    type  = "string"
    infix = true
  }

  # Embedded artist info (denormalized)
  field {
    name = "artist_id"
    type = "string"
  }

  field {
    name  = "artist_name"
    type  = "string"
    facet = true
    infix = true
  }

  # Embedded genre info (denormalized)
  field {
    name = "genre_id"
    type = "string"
  }

  field {
    name  = "genre_name"
    type  = "string"
    facet = true
  }

  # Embedded media type info (denormalized)
  field {
    name = "media_type_id"
    type = "string"
  }

  field {
    name  = "media_type_name"
    type  = "string"
    facet = true
  }

  # Computed/analytics field
  field {
    name     = "popularity_score"
    type     = "int32"
    sort     = true
    optional = true
  }
}

# =============================================================================
# ALBUMS COLLECTION
# Denormalized from: Album + Artist (with aggregated track stats)
# =============================================================================
resource "typesense_collection" "albums" {
  name                 = "albums"
  enable_nested_fields = true

  # Primary key
  field {
    name = "id"
    type = "string"
  }

  # Album info
  field {
    name  = "title"
    type  = "string"
    infix = true
  }

  # Embedded artist info (denormalized)
  field {
    name = "artist_id"
    type = "string"
  }

  field {
    name  = "artist_name"
    type  = "string"
    facet = true
    infix = true
  }

  # Aggregated from tracks (denormalized)
  field {
    name  = "genres"
    type  = "string[]"
    facet = true
  }

  field {
    name = "track_count"
    type = "int32"
    sort = true
  }

  field {
    name = "total_duration_seconds"
    type = "int64"
    sort = true
  }

  field {
    name     = "release_year"
    type     = "int32"
    facet    = true
    sort     = true
    optional = true
  }
}

# =============================================================================
# ARTISTS COLLECTION
# Denormalized from: Artist (with aggregated album/track stats)
# =============================================================================
resource "typesense_collection" "artists" {
  name                 = "artists"
  enable_nested_fields = true

  # Primary key
  field {
    name = "id"
    type = "string"
  }

  # Artist info
  field {
    name  = "name"
    type  = "string"
    infix = true
  }

  # Aggregated stats (denormalized)
  field {
    name  = "genres"
    type  = "string[]"
    facet = true
  }

  field {
    name = "album_count"
    type = "int32"
    sort = true
  }

  field {
    name = "track_count"
    type = "int32"
    sort = true
  }
}

# =============================================================================
# CUSTOMERS COLLECTION
# Denormalized from: Customer + Employee (support rep embedded as object)
# =============================================================================
resource "typesense_collection" "customers" {
  name                 = "customers"
  enable_nested_fields = true

  # Primary key
  field {
    name = "id"
    type = "string"
  }

  # Customer info
  field {
    name  = "full_name"
    type  = "string"
    infix = true
  }

  field {
    name = "first_name"
    type = "string"
  }

  field {
    name = "last_name"
    type = "string"
  }

  field {
    name     = "company"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name     = "address"
    type     = "string"
    optional = true
  }

  field {
    name     = "city"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name     = "state"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name     = "country"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name     = "postal_code"
    type     = "string"
    optional = true
  }

  field {
    name     = "phone"
    type     = "string"
    optional = true
  }

  field {
    name     = "fax"
    type     = "string"
    optional = true
  }

  field {
    name = "email"
    type = "string"
  }

  # Embedded support rep info (denormalized from Employee)
  field {
    name     = "support_rep"
    type     = "object"
    optional = true
  }

  # Aggregated purchase info (denormalized)
  field {
    name     = "total_purchases"
    type     = "float"
    sort     = true
    optional = true
  }

  field {
    name     = "invoice_count"
    type     = "int32"
    sort     = true
    optional = true
  }
}

# =============================================================================
# INVOICES COLLECTION
# Denormalized from: Invoice + InvoiceLine + Track + Customer
# =============================================================================
resource "typesense_collection" "invoices" {
  name                 = "invoices"
  enable_nested_fields = true

  # Primary key
  field {
    name = "id"
    type = "string"
  }

  # Invoice info
  field {
    name = "invoice_date"
    type = "int64"
    sort = true
  }

  field {
    name     = "billing_address"
    type     = "string"
    optional = true
  }

  field {
    name     = "billing_city"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name     = "billing_state"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name     = "billing_country"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name     = "billing_postal_code"
    type     = "string"
    optional = true
  }

  field {
    name  = "total"
    type  = "float"
    facet = true
    sort  = true
  }

  # Embedded customer info (denormalized)
  field {
    name = "customer_id"
    type = "string"
  }

  field {
    name  = "customer_name"
    type  = "string"
    infix = true
  }

  field {
    name     = "customer_email"
    type     = "string"
    optional = true
  }

  # Embedded line items with track info (denormalized)
  field {
    name = "line_items"
    type = "object[]"
  }

  # Searchable track names from line items (denormalized)
  field {
    name  = "track_names"
    type  = "string[]"
    infix = true
  }

  # Line item count
  field {
    name = "line_item_count"
    type = "int32"
  }
}

# =============================================================================
# EMPLOYEES COLLECTION
# Denormalized from: Employee (with manager embedded as object)
# =============================================================================
resource "typesense_collection" "employees" {
  name                 = "employees"
  enable_nested_fields = true

  # Primary key
  field {
    name = "id"
    type = "string"
  }

  # Employee info
  field {
    name  = "full_name"
    type  = "string"
    infix = true
  }

  field {
    name = "first_name"
    type = "string"
  }

  field {
    name = "last_name"
    type = "string"
  }

  field {
    name  = "title"
    type  = "string"
    facet = true
  }

  field {
    name     = "birth_date"
    type     = "int64"
    optional = true
  }

  field {
    name = "hire_date"
    type = "int64"
    sort = true
  }

  field {
    name     = "address"
    type     = "string"
    optional = true
  }

  field {
    name  = "city"
    type  = "string"
    facet = true
  }

  field {
    name     = "state"
    type     = "string"
    facet    = true
    optional = true
  }

  field {
    name  = "country"
    type  = "string"
    facet = true
  }

  field {
    name     = "postal_code"
    type     = "string"
    optional = true
  }

  field {
    name     = "phone"
    type     = "string"
    optional = true
  }

  field {
    name     = "fax"
    type     = "string"
    optional = true
  }

  field {
    name = "email"
    type = "string"
  }

  # Embedded manager info (denormalized from self-referencing Employee)
  field {
    name     = "manager"
    type     = "object"
    optional = true
  }

  # Aggregated stats (denormalized)
  field {
    name     = "customers_supported"
    type     = "int32"
    sort     = true
    optional = true
  }
}

# =============================================================================
# PLAYLISTS COLLECTION
# Denormalized from: Playlist + PlaylistTrack + Track
# =============================================================================
resource "typesense_collection" "playlists" {
  name                 = "playlists"
  enable_nested_fields = true

  # Primary key
  field {
    name = "id"
    type = "string"
  }

  # Playlist info
  field {
    name  = "name"
    type  = "string"
    infix = true
  }

  # Embedded tracks with basic info (denormalized)
  field {
    name = "tracks"
    type = "object[]"
  }

  # Searchable track names (denormalized)
  field {
    name  = "track_names"
    type  = "string[]"
    infix = true
  }

  # Aggregated from tracks (denormalized)
  field {
    name  = "artists"
    type  = "string[]"
    facet = true
  }

  field {
    name  = "genres"
    type  = "string[]"
    facet = true
  }

  field {
    name = "track_count"
    type = "int32"
    sort = true
  }

  field {
    name = "total_duration_seconds"
    type = "int64"
    sort = true
  }
}
