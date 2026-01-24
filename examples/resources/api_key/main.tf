# Example: Typesense API Keys

terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

provider "typesense" {
  server_host    = var.typesense_host
  server_api_key = var.typesense_api_key
}

variable "typesense_host" {
  type        = string
  description = "Typesense server hostname"
}

variable "typesense_api_key" {
  type        = string
  description = "Typesense server API key (admin)"
  sensitive   = true
}

# Create collections first
resource "typesense_collection" "products" {
  name = "products"

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "name"
    type = "string"
  }
}

resource "typesense_collection" "orders" {
  name = "orders"

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "customer_id"
    type = "string"
  }
}

# Search-only key for frontend (public)
resource "typesense_api_key" "search_only" {
  description = "Search-only key for frontend application"
  actions     = ["documents:search"]
  collections = [typesense_collection.products.name]
}

# Search key for all collections
resource "typesense_api_key" "search_all" {
  description = "Search key for all collections"
  actions     = ["documents:search"]
  collections = ["*"]
}

# Read-only key for analytics/reporting
resource "typesense_api_key" "read_only" {
  description = "Read-only key for analytics"
  actions     = ["documents:search", "documents:get", "collections:get"]
  collections = ["*"]
}

# Write key for backend indexing service
resource "typesense_api_key" "indexer" {
  description = "Indexer key for backend service"
  actions = [
    "documents:create",
    "documents:upsert",
    "documents:update",
    "documents:delete",
    "documents:import"
  ]
  collections = [
    typesense_collection.products.name,
    typesense_collection.orders.name
  ]
}

# Full access key for a specific collection
resource "typesense_api_key" "products_admin" {
  description = "Admin key for products collection"
  actions     = ["*"]
  collections = [typesense_collection.products.name]
}

# Expiring key (useful for temporary access)
resource "typesense_api_key" "temporary" {
  description = "Temporary access key (expires in 24 hours)"
  actions     = ["documents:search"]
  collections = ["*"]
  expires_at  = 1735776000  # Set to a future Unix timestamp
}

# Key with collection management permissions
resource "typesense_api_key" "collection_manager" {
  description = "Key for managing collections"
  actions = [
    "collections:create",
    "collections:delete",
    "collections:get",
    "collections:list"
  ]
  collections = ["*"]
}

# Key for managing synonyms and overrides
resource "typesense_api_key" "search_config" {
  description = "Key for managing search configuration"
  actions = [
    "synonyms:*",
    "overrides:*",
    "stopwords:*"
  ]
  collections = ["*"]
}

# Outputs - mark API key values as sensitive
output "search_only_key" {
  description = "Search-only API key for frontend"
  value       = typesense_api_key.search_only.value
  sensitive   = true
}

output "search_only_key_id" {
  description = "Search-only API key ID"
  value       = typesense_api_key.search_only.id
}

output "indexer_key" {
  description = "Indexer API key for backend"
  value       = typesense_api_key.indexer.value
  sensitive   = true
}

output "indexer_key_id" {
  description = "Indexer API key ID"
  value       = typesense_api_key.indexer.id
}
