# API Keys for Chinook Music Database
# Demonstrates fine-grained access control for different use cases

# Search-only key for public-facing music search
# Can only search tracks, albums, artists, and playlists
resource "typesense_api_key" "search_only" {
  description = "Public search key for music catalog"
  actions     = ["documents:search"]
  collections = [
    typesense_collection.tracks.name,
    typesense_collection.albums.name,
    typesense_collection.artists.name,
    typesense_collection.playlists.name,
  ]
}

# Admin key for customer management
# Full access to customer and invoice data
resource "typesense_api_key" "customer_admin" {
  description = "Admin key for customer management"
  actions = [
    "documents:search",
    "documents:get",
    "documents:create",
    "documents:upsert",
    "documents:update",
    "documents:delete",
  ]
  collections = [
    typesense_collection.customers.name,
    typesense_collection.invoices.name,
  ]
}

# Analytics read key for reporting dashboards
# Read-only access to analytics destination collections
resource "typesense_api_key" "analytics_reader" {
  description = "Read-only key for analytics dashboards"
  actions = [
    "documents:search",
    "documents:get",
  ]
  collections = [
    typesense_collection.track_queries.name,
    typesense_collection.album_queries.name,
    typesense_collection.nohits_queries.name,
  ]
}

# Shared search key with user-provided value
# Demonstrates multi-environment pattern: use the same key value across
# prod/staging by passing it as a variable, so client applications don't
# need to update their key when switching environments.
resource "typesense_api_key" "shared_search" {
  count = var.shared_search_key != "" ? 1 : 0

  description = "Shared search key (same value across environments)"
  value       = var.shared_search_key
  actions     = ["documents:search"]
  collections = [
    typesense_collection.tracks.name,
    typesense_collection.albums.name,
    typesense_collection.artists.name,
  ]
}
