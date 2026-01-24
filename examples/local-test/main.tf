terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

# Configure provider for local Typesense server
provider "typesense" {
  server_host     = "localhost"
  server_api_key  = "xyz"           # Match the API key from docker run
  server_port     = 8108            # Default Typesense port
  server_protocol = "http"          # Local uses HTTP
}

# Test collection
resource "typesense_collection" "test_products" {
  name                  = "test_products"
  default_sorting_field = "popularity"

  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "name"
    type  = "string"
    index = true
  }

  field {
    name  = "price"
    type  = "float"
    facet = true
  }

  field {
    name = "popularity"
    type = "int32"
    sort = true
  }
}

# Test synonym
resource "typesense_synonym" "test_synonym" {
  collection = typesense_collection.test_products.name
  name       = "shoe-synonyms"
  synonyms   = ["shoe", "sneaker", "trainer"]
}

output "collection_name" {
  value = typesense_collection.test_products.name
}
