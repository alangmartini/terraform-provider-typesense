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
