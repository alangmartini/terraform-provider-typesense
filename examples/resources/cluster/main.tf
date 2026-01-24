# Example: Typesense Cloud Cluster

terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

provider "typesense" {
  cloud_management_api_key = var.cloud_api_key
}

variable "cloud_api_key" {
  type        = string
  description = "Typesense Cloud Management API key"
  sensitive   = true
}

# Create a basic cluster
resource "typesense_cluster" "basic" {
  name                     = "my-basic-cluster"
  memory                   = "0.5_gb"
  vcpu                     = "2_vcpus_4_hr_burst_per_day"
  regions                  = ["us-east-1"]
  typesense_server_version = "27.1"
}

# Create a production-ready HA cluster
resource "typesense_cluster" "production" {
  name                     = "production-search"
  memory                   = "4_gb"
  vcpu                     = "2_vcpus"
  regions                  = ["us-east-1"]
  high_availability        = "yes"
  search_delivery_network  = "off"
  typesense_server_version = "27.1"
  auto_upgrade_capacity    = true
}

# Schedule a configuration change
resource "typesense_cluster_config_change" "upgrade" {
  cluster_id                   = typesense_cluster.production.id
  new_memory                   = "8_gb"
  new_vcpu                     = "4_vcpus"
  new_typesense_server_version = "28.0"
  # perform_change_at = 1735689600  # Uncomment to schedule for a specific time
}

output "basic_cluster_hostname" {
  value = typesense_cluster.basic.load_balanced_hostname
}

output "production_cluster_hostname" {
  value = typesense_cluster.production.load_balanced_hostname
}

output "production_admin_key" {
  value     = typesense_cluster.production.admin_api_key
  sensitive = true
}
