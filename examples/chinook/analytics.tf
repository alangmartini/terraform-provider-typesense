# Analytics Rules for Chinook Music Database
# Track search behavior and user interactions

# =============================================================================
# ANALYTICS DESTINATION COLLECTIONS
# Collections to store aggregated analytics data
# =============================================================================

# Store popular search queries for tracks
resource "typesense_collection" "track_queries" {
  name = "track_queries"

  field {
    name = "q"
    type = "string"
  }

  field {
    name = "count"
    type = "int32"
    sort = true
  }
}

# Store queries that returned no results
resource "typesense_collection" "nohits_queries" {
  name = "nohits_queries"

  field {
    name = "q"
    type = "string"
  }

  field {
    name = "count"
    type = "int32"
    sort = true
  }
}

# Store popular album searches
resource "typesense_collection" "album_queries" {
  name = "album_queries"

  field {
    name = "q"
    type = "string"
  }

  field {
    name = "count"
    type = "int32"
    sort = true
  }
}

# =============================================================================
# POPULAR QUERIES RULES
# Track frequently searched terms for query suggestions
# =============================================================================

# Track popular track searches
resource "typesense_analytics_rule" "track_popular_queries" {
  name = "track-popular-queries"
  type = "popular_queries"
  params = jsonencode({
    source = {
      collections = [typesense_collection.tracks.name]
    }
    destination = {
      collection = typesense_collection.track_queries.name
    }
    limit = 1000
  })
}

# Track popular album searches
resource "typesense_analytics_rule" "album_popular_queries" {
  name = "album-popular-queries"
  type = "popular_queries"
  params = jsonencode({
    source = {
      collections = [typesense_collection.albums.name]
    }
    destination = {
      collection = typesense_collection.album_queries.name
    }
    limit = 500
  })
}

# =============================================================================
# NO HITS QUERIES RULES
# Identify content gaps by tracking zero-result searches
# =============================================================================

# Track track searches with no results
resource "typesense_analytics_rule" "track_nohits" {
  name = "track-nohits-queries"
  type = "nohits_queries"
  params = jsonencode({
    source = {
      collections = [typesense_collection.tracks.name]
    }
    destination = {
      collection = typesense_collection.nohits_queries.name
    }
    limit = 500
  })
}

# =============================================================================
# COUNTER RULES
# Track user interactions to build popularity scores
# =============================================================================

# Track track popularity based on clicks and plays
resource "typesense_analytics_rule" "track_popularity" {
  name = "track-popularity-counter"
  type = "counter"
  params = jsonencode({
    source = {
      collections = [typesense_collection.tracks.name]
      events = [
        {
          type   = "click"
          weight = 1
          name   = "track_click"
        },
        {
          type   = "conversion"
          weight = 3
          name   = "track_play"
        }
      ]
    }
    destination = {
      collection    = typesense_collection.tracks.name
      counter_field = "popularity_score"
    }
  })
}
