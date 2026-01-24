# Example: Typesense Collections

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

# Basic collection for products
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
    name  = "price"
    type  = "float"
    facet = true
  }

  field {
    name  = "categories"
    type  = "string[]"
    facet = true
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
    name = "location"
    type = "geopoint"
  }
}

# Collection with nested fields enabled
resource "typesense_collection" "orders" {
  name                 = "orders"
  enable_nested_fields = true

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "customer"
    type = "object"
  }

  field {
    name = "items"
    type = "object[]"
  }

  field {
    name = "order_date"
    type = "int64"
    sort = true
  }

  field {
    name  = "status"
    type  = "string"
    facet = true
  }
}

# Collection with custom token separators
resource "typesense_collection" "documents" {
  name             = "documents"
  token_separators = ["-", "_", "."]
  symbols_to_index = ["#", "@"]

  field {
    name  = "title"
    type  = "string"
    infix = true
  }

  field {
    name = "content"
    type = "string"
  }

  field {
    name     = "tags"
    type     = "string[]"
    facet    = true
    optional = true
  }

  field {
    name   = "language"
    type   = "string"
    locale = "en"
  }
}

output "products_collection" {
  value = typesense_collection.products.name
}

output "products_num_documents" {
  value = typesense_collection.products.num_documents
}
