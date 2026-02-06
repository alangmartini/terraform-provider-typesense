# Curations (Overrides) for Chinook Music Database
# Pin, hide, or modify search results for specific queries

# =============================================================================
# FEATURED CONTENT CURATIONS
# Pin specific tracks/albums for promotional queries
# =============================================================================

# Promote classic rock when users search for "best of"
resource "typesense_override" "best_of_tracks" {
  collection = typesense_collection.tracks.name
  name       = "best-of-classics"

  rule = {
    query = "best of"
    match = "contains"
  }

  # Pin legendary tracks at top positions
  includes {
    id       = "1"  # Example: "For Those About to Rock"
    position = 1
  }

  includes {
    id       = "6"  # Example: "Put The Finger On You"
    position = 2
  }

  # Apply filter to show only high-rated tracks
  filter_by = "popularity_score:>50"
  sort_by   = "popularity_score:desc"
}

# Featured new releases promotion
resource "typesense_override" "new_releases" {
  collection = typesense_collection.albums.name
  name       = "new-releases-promo"

  rule = {
    query = "new"
    match = "contains"
  }

  sort_by = "release_year:desc"
}

# =============================================================================
# QUERY CORRECTIONS
# Fix common misspellings or redirect searches
# =============================================================================

# Redirect "acdc" variations to canonical search
resource "typesense_override" "acdc_redirect" {
  collection = typesense_collection.tracks.name
  name       = "acdc-redirect"

  rule = {
    query = "acdc"
    match = "exact"
  }

  replace_query         = "AC/DC"
  remove_matched_tokens = true
}

# Handle "beatles" search (common misspelling scenarios)
resource "typesense_override" "beatles_search" {
  collection = typesense_collection.tracks.name
  name       = "beatles-boost"

  rule = {
    query = "beatles"
    match = "contains"
  }

  # Boost Beatles tracks to top
  sort_by = "artist_name:asc,popularity_score:desc"
}

# =============================================================================
# CONTENT HIDING
# Exclude specific content from search results
# =============================================================================

# Hide placeholder/test tracks from production searches
resource "typesense_override" "hide_test_content" {
  collection = typesense_collection.tracks.name
  name       = "hide-test-tracks"

  rule = {
    query = "*"
    match = "exact"
  }

  # Example: Hide test track IDs
  excludes {
    id = "9999"
  }

  excludes {
    id = "9998"
  }

  # Don't block other overrides from running
  stop_processing = false
}

# =============================================================================
# GENRE-SPECIFIC CURATIONS
# Customize results for genre searches
# =============================================================================

# Enhance rock music searches
resource "typesense_override" "rock_enhanced" {
  collection = typesense_collection.tracks.name
  name       = "rock-genre-boost"

  rule = {
    query = "rock"
    match = "contains"
  }

  filter_by           = "genre_name:Rock"
  filter_curated_hits = true
  sort_by             = "popularity_score:desc"
}

# Jazz discovery curation
resource "typesense_override" "jazz_discovery" {
  collection = typesense_collection.tracks.name
  name       = "jazz-discovery"

  rule = {
    query = "jazz"
    match = "contains"
  }

  filter_by = "genre_name:Jazz"
  sort_by   = "artist_name:asc"
}

# =============================================================================
# TAG-BASED CURATIONS
# Apply curations based on search context tags
# =============================================================================

# Mobile app search optimization
resource "typesense_override" "mobile_search" {
  collection = typesense_collection.tracks.name
  name       = "mobile-optimized"

  rule = {
    tags = ["mobile", "app"]
  }

  # Limit results for mobile bandwidth
  filter_by = "popularity_score:>25"
}

# Premium user experience
resource "typesense_override" "premium_search" {
  collection = typesense_collection.tracks.name
  name       = "premium-experience"

  rule = {
    tags = ["premium"]
  }

  # Premium users get access to all content, sorted by quality
  sort_by = "popularity_score:desc"
}
