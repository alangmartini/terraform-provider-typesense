terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

# Configure the Typesense provider
# For Typesense Cloud clusters, set cloud_management_api_key
# For server API resources, set server_host and server_api_key

provider "typesense" {
  # Cloud Management API key (optional - only needed for cluster management)
  # Can also be set via TYPESENSE_CLOUD_MANAGEMENT_API_KEY environment variable
  cloud_management_api_key = var.typesense_cloud_api_key

  # Server API configuration (required for collections, synonyms, etc.)
  # Can also be set via environment variables:
  # - TYPESENSE_HOST
  # - TYPESENSE_API_KEY
  # - TYPESENSE_PORT
  # - TYPESENSE_PROTOCOL
  server_host     = var.typesense_host
  server_api_key  = var.typesense_api_key
  server_port     = 443      # optional, defaults to 443
  server_protocol = "https"  # optional, defaults to "https"
}

variable "typesense_cloud_api_key" {
  type        = string
  description = "Typesense Cloud Management API key"
  sensitive   = true
  default     = ""
}

variable "typesense_host" {
  type        = string
  description = "Typesense server hostname"
  default     = ""
}

variable "typesense_api_key" {
  type        = string
  description = "Typesense server API key"
  sensitive   = true
  default     = ""
}
