# Complete Example: Typesense Terraform Provider
# This example shows how to use all resources together

terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

# Configure provider for both Cloud and Server APIs
provider "typesense" {
  # For Typesense Cloud cluster management
  cloud_management_api_key = var.cloud_api_key

  # For server API operations (will use the created cluster)
  server_host    = var.server_host
  server_api_key = var.server_api_key
}

variable "cloud_api_key" {
  type        = string
  description = "Typesense Cloud Management API key"
  sensitive   = true
  default     = ""
}

variable "server_host" {
  type        = string
  description = "Typesense server hostname (use cluster output or provide directly)"
}

variable "server_api_key" {
  type        = string
  description = "Typesense server admin API key"
  sensitive   = true
}

# ============================================
# COLLECTIONS
# ============================================

# Products collection for e-commerce
resource "typesense_collection" "products" {
  name                  = "products"
  default_sorting_field = "popularity_score"

  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "name"
    type  = "string"
    infix = true
  }

  field {
    name     = "description"
    type     = "string"
    optional = true
  }

  field {
    name  = "brand"
    type  = "string"
    facet = true
  }

  field {
    name  = "category"
    type  = "string[]"
    facet = true
  }

  field {
    name  = "price"
    type  = "float"
    facet = true
    sort  = true
  }

  field {
    name = "popularity_score"
    type = "int32"
    sort = true
  }

  field {
    name  = "in_stock"
    type  = "bool"
    facet = true
  }

  field {
    name     = "tags"
    type     = "string[]"
    facet    = true
    optional = true
  }
}

# ============================================
# SYNONYMS
# ============================================

# Apparel synonyms
resource "typesense_synonym" "apparel" {
  collection = typesense_collection.products.name
  name       = "apparel-synonyms"
  synonyms   = ["shirt", "tee", "t-shirt", "top"]
}

# Electronics synonyms
resource "typesense_synonym" "phones" {
  collection = typesense_collection.products.name
  name       = "phone-synonyms"
  synonyms   = ["phone", "mobile", "cellphone", "smartphone"]
}

# Laptop one-way synonym
resource "typesense_synonym" "laptop" {
  collection = typesense_collection.products.name
  name       = "laptop-synonyms"
  root       = "laptop"
  synonyms   = ["notebook", "macbook", "chromebook"]
}

# ============================================
# OVERRIDES (CURATION RULES)
# ============================================

# Pin featured products
resource "typesense_override" "featured" {
  collection = typesense_collection.products.name
  name       = "featured-products"

  rule {
    query = "featured"
    match = "exact"
  }

  includes {
    id       = "featured-001"
    position = 1
  }

  includes {
    id       = "featured-002"
    position = 2
  }
}

# Brand filter override
resource "typesense_override" "apple_search" {
  collection = typesense_collection.products.name
  name       = "apple-brand-filter"

  rule {
    query = "apple products"
    match = "contains"
  }

  filter_by = "brand:=Apple"
}

# Exclude out of stock from certain queries
resource "typesense_override" "in_stock_only" {
  collection = typesense_collection.products.name
  name       = "in-stock-filter"

  rule {
    query = "available"
    match = "contains"
  }

  filter_by       = "in_stock:=true"
  stop_processing = false
}

# ============================================
# STOPWORDS
# ============================================

# English stopwords
resource "typesense_stopwords_set" "english" {
  name   = "english-common"
  locale = "en"
  stopwords = [
    "a", "an", "and", "are", "as", "at", "be", "by", "for",
    "from", "has", "he", "in", "is", "it", "its", "of", "on",
    "that", "the", "to", "was", "were", "will", "with"
  ]
}

# E-commerce specific stopwords
resource "typesense_stopwords_set" "ecommerce" {
  name = "ecommerce-noise"
  stopwords = [
    "buy", "shop", "purchase", "order", "price", "cost",
    "cheap", "deal", "sale", "discount", "free", "best"
  ]
}

# ============================================
# API KEYS
# ============================================

# Search-only key for frontend
resource "typesense_api_key" "frontend_search" {
  description = "Search-only key for frontend application"
  actions     = ["documents:search"]
  collections = [typesense_collection.products.name]
}

# Backend indexing key
resource "typesense_api_key" "backend_indexer" {
  description = "Indexer key for backend service"
  actions = [
    "documents:create",
    "documents:upsert",
    "documents:update",
    "documents:delete",
    "documents:import"
  ]
  collections = [typesense_collection.products.name]
}

# Admin key for search configuration
resource "typesense_api_key" "search_admin" {
  description = "Admin key for managing search configuration"
  actions = [
    "documents:search",
    "synonyms:*",
    "overrides:*"
  ]
  collections = [typesense_collection.products.name]
}

# ============================================
# OUTPUTS
# ============================================

output "collection_name" {
  description = "Products collection name"
  value       = typesense_collection.products.name
}

output "frontend_search_key" {
  description = "API key for frontend search (use this in your client app)"
  value       = typesense_api_key.frontend_search.value
  sensitive   = true
}

output "frontend_search_key_id" {
  description = "Frontend search key ID"
  value       = typesense_api_key.frontend_search.id
}

output "backend_indexer_key" {
  description = "API key for backend indexing"
  value       = typesense_api_key.backend_indexer.value
  sensitive   = true
}

output "backend_indexer_key_id" {
  description = "Backend indexer key ID"
  value       = typesense_api_key.backend_indexer.id
}
