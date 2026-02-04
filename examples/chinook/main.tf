# Chinook Database on Typesense Cloud
# Music store database with artists, albums, tracks, customers, invoices, employees, and playlists

terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

# Configure the Typesense provider for Typesense Cloud
provider "typesense" {
  server_host     = var.typesense_host
  server_api_key  = var.typesense_api_key
  server_port     = var.typesense_port
  server_protocol = var.typesense_protocol
}
