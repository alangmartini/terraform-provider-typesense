# Output values for Chinook database collections

# =============================================================================
# COLLECTION NAMES
# =============================================================================

output "tracks_collection_name" {
  description = "Tracks collection name"
  value       = typesense_collection.tracks.name
}

output "albums_collection_name" {
  description = "Albums collection name"
  value       = typesense_collection.albums.name
}

output "artists_collection_name" {
  description = "Artists collection name"
  value       = typesense_collection.artists.name
}

output "customers_collection_name" {
  description = "Customers collection name"
  value       = typesense_collection.customers.name
}

output "invoices_collection_name" {
  description = "Invoices collection name"
  value       = typesense_collection.invoices.name
}

output "employees_collection_name" {
  description = "Employees collection name"
  value       = typesense_collection.employees.name
}

output "playlists_collection_name" {
  description = "Playlists collection name"
  value       = typesense_collection.playlists.name
}

# =============================================================================
# DOCUMENT COUNTS
# =============================================================================

output "tracks_num_documents" {
  description = "Number of documents in tracks collection"
  value       = typesense_collection.tracks.num_documents
}

output "albums_num_documents" {
  description = "Number of documents in albums collection"
  value       = typesense_collection.albums.num_documents
}

output "artists_num_documents" {
  description = "Number of documents in artists collection"
  value       = typesense_collection.artists.num_documents
}

output "customers_num_documents" {
  description = "Number of documents in customers collection"
  value       = typesense_collection.customers.num_documents
}

output "invoices_num_documents" {
  description = "Number of documents in invoices collection"
  value       = typesense_collection.invoices.num_documents
}

output "employees_num_documents" {
  description = "Number of documents in employees collection"
  value       = typesense_collection.employees.num_documents
}

output "playlists_num_documents" {
  description = "Number of documents in playlists collection"
  value       = typesense_collection.playlists.num_documents
}

# =============================================================================
# COLLECTION SUMMARY
# =============================================================================

output "all_collections" {
  description = "Summary of all Chinook collections"
  value = {
    tracks = {
      name          = typesense_collection.tracks.name
      num_documents = typesense_collection.tracks.num_documents
      created_at    = typesense_collection.tracks.created_at
    }
    albums = {
      name          = typesense_collection.albums.name
      num_documents = typesense_collection.albums.num_documents
      created_at    = typesense_collection.albums.created_at
    }
    artists = {
      name          = typesense_collection.artists.name
      num_documents = typesense_collection.artists.num_documents
      created_at    = typesense_collection.artists.created_at
    }
    customers = {
      name          = typesense_collection.customers.name
      num_documents = typesense_collection.customers.num_documents
      created_at    = typesense_collection.customers.created_at
    }
    invoices = {
      name          = typesense_collection.invoices.name
      num_documents = typesense_collection.invoices.num_documents
      created_at    = typesense_collection.invoices.created_at
    }
    employees = {
      name          = typesense_collection.employees.name
      num_documents = typesense_collection.employees.num_documents
      created_at    = typesense_collection.employees.created_at
    }
    playlists = {
      name          = typesense_collection.playlists.name
      num_documents = typesense_collection.playlists.num_documents
      created_at    = typesense_collection.playlists.created_at
    }
  }
}

# =============================================================================
# NATURAL LANGUAGE SEARCH
# =============================================================================

output "nl_search_model_id" {
  description = "ID of the Natural Language Search model (use this as nl_model_id in search queries)"
  value       = length(typesense_nl_search_model.music_search) > 0 ? typesense_nl_search_model.music_search[0].id : null
}

output "nl_search_enabled" {
  description = "Whether Natural Language Search is enabled"
  value       = length(typesense_nl_search_model.music_search) > 0
}

# =============================================================================
# STOPWORDS SETS
# =============================================================================

output "stopwords_sets" {
  description = "Configured stopwords sets for the Chinook database"
  value = {
    english_common = {
      name   = typesense_stopwords_set.english_common.name
      locale = typesense_stopwords_set.english_common.locale
      count  = length(typesense_stopwords_set.english_common.stopwords)
    }
    music_terms = {
      name   = typesense_stopwords_set.music_terms.name
      locale = typesense_stopwords_set.music_terms.locale
      count  = length(typesense_stopwords_set.music_terms.stopwords)
    }
    billing_terms = {
      name   = typesense_stopwords_set.billing_terms.name
      locale = typesense_stopwords_set.billing_terms.locale
      count  = length(typesense_stopwords_set.billing_terms.stopwords)
    }
  }
}

# =============================================================================
# COLLECTION ALIASES
# =============================================================================

output "collection_aliases" {
  description = "Collection aliases for stable application endpoints"
  value = {
    music = {
      alias      = typesense_collection_alias.music.name
      collection = typesense_collection_alias.music.collection_name
    }
    catalog = {
      alias      = typesense_collection_alias.catalog.name
      collection = typesense_collection_alias.catalog.collection_name
    }
    artists = {
      alias      = typesense_collection_alias.artists.name
      collection = typesense_collection_alias.artists.collection_name
    }
    customers = {
      alias      = typesense_collection_alias.customers.name
      collection = typesense_collection_alias.customers.collection_name
    }
    orders = {
      alias      = typesense_collection_alias.orders.name
      collection = typesense_collection_alias.orders.collection_name
    }
    playlists = {
      alias      = typesense_collection_alias.playlists.name
      collection = typesense_collection_alias.playlists.collection_name
    }
  }
}

# =============================================================================
# SEARCH PRESETS
# =============================================================================

output "search_presets" {
  description = "Configured search presets for different use cases"
  value = {
    tracks = {
      listing      = typesense_preset.track_listing.name
      faceted      = typesense_preset.track_search_faceted.name
      autocomplete = typesense_preset.track_autocomplete.name
    }
    albums = {
      browse    = typesense_preset.album_browse.name
      discovery = typesense_preset.album_discovery.name
    }
    artists = {
      directory    = typesense_preset.artist_directory.name
      autocomplete = typesense_preset.artist_autocomplete.name
    }
    customers = {
      lookup    = typesense_preset.customer_lookup.name
      analytics = typesense_preset.customer_analytics.name
    }
    orders = {
      recent = typesense_preset.recent_orders.name
      search = typesense_preset.order_search.name
    }
    playlists = {
      browse = typesense_preset.playlist_browse.name
    }
  }
}

# =============================================================================
# ANALYTICS RULES
# =============================================================================

output "analytics_rules" {
  description = "Configured analytics rules for tracking search behavior"
  value = {
    popular_queries = {
      tracks = typesense_analytics_rule.track_popular_queries.name
      albums = typesense_analytics_rule.album_popular_queries.name
    }
    nohits_queries = {
      tracks = typesense_analytics_rule.track_nohits.name
    }
    counters = {
      track_popularity = typesense_analytics_rule.track_popularity.name
    }
  }
}

output "analytics_collections" {
  description = "Collections storing analytics data"
  value = {
    track_queries  = typesense_collection.track_queries.name
    album_queries  = typesense_collection.album_queries.name
    nohits_queries = typesense_collection.nohits_queries.name
  }
}

# =============================================================================
# CURATIONS (OVERRIDES)
# =============================================================================

output "curations" {
  description = "Configured curations for search result customization"
  value = {
    featured = {
      best_of_tracks = typesense_override.best_of_tracks.name
      new_releases   = typesense_override.new_releases.name
    }
    query_corrections = {
      acdc_redirect  = typesense_override.acdc_redirect.name
      beatles_boost  = typesense_override.beatles_search.name
    }
    genre_curations = {
      rock = typesense_override.rock_enhanced.name
      jazz = typesense_override.jazz_discovery.name
    }
    tag_based = {
      mobile  = typesense_override.mobile_search.name
      premium = typesense_override.premium_search.name
    }
  }
}
