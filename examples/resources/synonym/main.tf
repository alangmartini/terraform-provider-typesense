# Example: Typesense Synonyms

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
    name  = "category"
    type  = "string"
    facet = true
  }
}

# Multi-way synonyms - all terms are equivalent
resource "typesense_synonym" "apparel" {
  collection = typesense_collection.products.name
  name       = "apparel-synonyms"
  synonyms   = ["shirt", "tee", "t-shirt", "top", "blouse"]
}

# Multi-way synonyms for electronics
resource "typesense_synonym" "phones" {
  collection = typesense_collection.products.name
  name       = "phone-synonyms"
  synonyms   = ["phone", "mobile", "cellphone", "smartphone", "cell"]
}

# One-way synonym - map alternatives to a root term
resource "typesense_synonym" "laptop" {
  collection = typesense_collection.products.name
  name       = "laptop-synonyms"
  root       = "laptop"
  synonyms   = ["notebook", "portable computer", "macbook", "chromebook"]
}

# Size-related synonyms
resource "typesense_synonym" "sizes" {
  collection = typesense_collection.products.name
  name       = "size-synonyms"
  synonyms   = ["small", "sm", "s"]
}

resource "typesense_synonym" "sizes_medium" {
  collection = typesense_collection.products.name
  name       = "size-medium-synonyms"
  synonyms   = ["medium", "med", "m"]
}

resource "typesense_synonym" "sizes_large" {
  collection = typesense_collection.products.name
  name       = "size-large-synonyms"
  synonyms   = ["large", "lg", "l"]
}

# Color synonyms
resource "typesense_synonym" "colors_red" {
  collection = typesense_collection.products.name
  name       = "red-color-synonyms"
  synonyms   = ["red", "scarlet", "crimson", "ruby", "maroon"]
}

resource "typesense_synonym" "colors_blue" {
  collection = typesense_collection.products.name
  name       = "blue-color-synonyms"
  synonyms   = ["blue", "navy", "azure", "cobalt", "indigo"]
}
