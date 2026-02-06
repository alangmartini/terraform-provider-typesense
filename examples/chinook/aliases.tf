# Collection Aliases for Chinook Music Database
# Provides stable endpoints for applications during reindexing

# =============================================================================
# PRIMARY SEARCH ALIASES
# Applications should use these aliases instead of direct collection names
# =============================================================================

# Main music search alias - points to tracks collection
resource "typesense_collection_alias" "music" {
  name            = "music"
  collection_name = typesense_collection.tracks.name
}

# Catalog search alias - points to albums collection
resource "typesense_collection_alias" "catalog" {
  name            = "catalog"
  collection_name = typesense_collection.albums.name
}

# Artist directory alias
resource "typesense_collection_alias" "artists" {
  name            = "artist-directory"
  collection_name = typesense_collection.artists.name
}

# =============================================================================
# CUSTOMER/BUSINESS ALIASES
# Separate aliases for business operations vs public search
# =============================================================================

# Customer lookup alias
resource "typesense_collection_alias" "customers" {
  name            = "customer-search"
  collection_name = typesense_collection.customers.name
}

# Invoice/order search alias
resource "typesense_collection_alias" "orders" {
  name            = "order-search"
  collection_name = typesense_collection.invoices.name
}

# =============================================================================
# PLAYLIST ALIAS
# =============================================================================

# Playlist search alias
resource "typesense_collection_alias" "playlists" {
  name            = "playlist-search"
  collection_name = typesense_collection.playlists.name
}
