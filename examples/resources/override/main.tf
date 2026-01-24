# Example: Typesense Overrides (Curation Rules)

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
  description = "Typesense server API key"
  sensitive   = true
}

# Assume we have a products collection
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

  field {
    name  = "brand"
    type  = "string"
    facet = true
  }

  field {
    name  = "category"
    type  = "string"
    facet = true
  }
}

# Pin featured products for "best sellers" query
resource "typesense_override" "best_sellers" {
  collection = typesense_collection.products.name
  name       = "best-sellers-override"

  rule {
    query = "best sellers"
    match = "exact"
  }

  includes {
    id       = "product-001"
    position = 1
  }

  includes {
    id       = "product-002"
    position = 2
  }

  includes {
    id       = "product-003"
    position = 3
  }
}

# Exclude discontinued products
resource "typesense_override" "exclude_discontinued" {
  collection = typesense_collection.products.name
  name       = "exclude-discontinued"

  rule {
    query = "*"
    match = "contains"
  }

  excludes {
    id = "discontinued-001"
  }

  excludes {
    id = "discontinued-002"
  }
}

# Apply filter for brand-specific searches
resource "typesense_override" "apple_brand" {
  collection = typesense_collection.products.name
  name       = "apple-brand-filter"

  rule {
    query = "apple"
    match = "exact"
  }

  filter_by = "brand:=Apple"
}

# Replace query for common misspellings
resource "typesense_override" "iphone_spelling" {
  collection = typesense_collection.products.name
  name       = "iphone-spelling-fix"

  rule {
    query = "ifone"
    match = "exact"
  }

  replace_query        = "iphone"
  remove_matched_tokens = true
}

# Time-limited promotion override
resource "typesense_override" "holiday_sale" {
  collection = typesense_collection.products.name
  name       = "holiday-sale-promotion"

  rule {
    query = "sale"
    match = "contains"
  }

  includes {
    id       = "holiday-deal-001"
    position = 1
  }

  includes {
    id       = "holiday-deal-002"
    position = 2
  }

  # Active during the holiday period
  effective_from_ts = 1703030400  # Dec 20, 2024
  effective_to_ts   = 1704240000  # Jan 3, 2025
}

# Override with custom sorting
resource "typesense_override" "newest_first" {
  collection = typesense_collection.products.name
  name       = "newest-products-sort"

  rule {
    query = "new arrivals"
    match = "exact"
  }

  sort_by         = "created_at:desc"
  stop_processing = true
}

# Tag-based override (useful for A/B testing or user segments)
resource "typesense_override" "vip_products" {
  collection = typesense_collection.products.name
  name       = "vip-customer-products"

  rule {
    tags = ["vip", "premium"]
  }

  filter_by           = "category:=Premium"
  filter_curated_hits = true
}
