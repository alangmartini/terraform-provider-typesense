# Search Presets for Chinook Music Database
# Store common search configurations server-side

# =============================================================================
# TRACK SEARCH PRESETS
# =============================================================================

# Default track listing - sorted by popularity
resource "typesense_preset" "track_listing" {
  name = "track-listing"
  value = jsonencode({
    q        = "*"
    query_by = "name,artist_name,album_title"
    sort_by  = "popularity_score:desc"
    per_page = 25
  })
}

# Track search with facets for filtering UI
resource "typesense_preset" "track_search_faceted" {
  name = "track-search-faceted"
  value = jsonencode({
    query_by       = "name,artist_name,album_title,composer"
    facet_by       = "genre_name,artist_name,media_type_name"
    max_facet_values = 20
    per_page       = 20
  })
}

# Quick track search - minimal response for autocomplete
resource "typesense_preset" "track_autocomplete" {
  name = "track-autocomplete"
  value = jsonencode({
    query_by         = "name,artist_name"
    include_fields   = "id,name,artist_name,album_title"
    per_page         = 10
    prefix           = true
    num_typos        = 1
  })
}

# =============================================================================
# ALBUM SEARCH PRESETS
# =============================================================================

# Album browse - sorted by track count
resource "typesense_preset" "album_browse" {
  name = "album-browse"
  value = jsonencode({
    q        = "*"
    query_by = "title,artist_name"
    sort_by  = "track_count:desc"
    facet_by = "genres,artist_name"
    per_page = 20
  })
}

# Album search for discovery
resource "typesense_preset" "album_discovery" {
  name = "album-discovery"
  value = jsonencode({
    query_by = "title,artist_name,genres"
    facet_by = "genres,release_year"
    sort_by  = "total_duration_seconds:desc"
    per_page = 12
  })
}

# =============================================================================
# ARTIST SEARCH PRESETS
# =============================================================================

# Artist directory listing
resource "typesense_preset" "artist_directory" {
  name = "artist-directory"
  value = jsonencode({
    q        = "*"
    query_by = "name"
    sort_by  = "album_count:desc,track_count:desc"
    facet_by = "genres"
    per_page = 50
  })
}

# Artist autocomplete for search box
resource "typesense_preset" "artist_autocomplete" {
  name = "artist-autocomplete"
  value = jsonencode({
    query_by       = "name"
    include_fields = "id,name,album_count"
    per_page       = 8
    prefix         = true
  })
}

# =============================================================================
# CUSTOMER SEARCH PRESETS
# =============================================================================

# Customer lookup by name or email
resource "typesense_preset" "customer_lookup" {
  name = "customer-lookup"
  value = jsonencode({
    query_by = "full_name,email,company"
    sort_by  = "total_purchases:desc"
    per_page = 20
  })
}

# Customer analytics view
resource "typesense_preset" "customer_analytics" {
  name = "customer-analytics"
  value = jsonencode({
    q        = "*"
    query_by = "full_name"
    facet_by = "country,city,company"
    sort_by  = "total_purchases:desc"
    per_page = 100
  })
}

# =============================================================================
# INVOICE/ORDER SEARCH PRESETS
# =============================================================================

# Recent orders listing
resource "typesense_preset" "recent_orders" {
  name = "recent-orders"
  value = jsonencode({
    q        = "*"
    query_by = "customer_name,track_names"
    sort_by  = "invoice_date:desc"
    per_page = 50
  })
}

# Order search with billing filters
resource "typesense_preset" "order_search" {
  name = "order-search"
  value = jsonencode({
    query_by = "customer_name,customer_email,track_names"
    facet_by = "billing_country,billing_city"
    sort_by  = "total:desc"
    per_page = 25
  })
}

# =============================================================================
# PLAYLIST SEARCH PRESETS
# =============================================================================

# Playlist browse
resource "typesense_preset" "playlist_browse" {
  name = "playlist-browse"
  value = jsonencode({
    q        = "*"
    query_by = "name,track_names"
    sort_by  = "track_count:desc"
    facet_by = "genres,artists"
    per_page = 20
  })
}
