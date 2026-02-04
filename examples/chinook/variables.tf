# Input variables for Chinook database on Typesense Cloud

variable "typesense_api_key" {
  type        = string
  description = "Typesense server API key (admin key with full permissions)"
  sensitive   = true
}

variable "typesense_host" {
  type        = string
  description = "Typesense Cloud hostname (e.g., xxx.a1.typesense.net)"
}

variable "typesense_port" {
  type        = number
  description = "Typesense server port"
  default     = 443
}

variable "typesense_protocol" {
  type        = string
  description = "Typesense server protocol"
  default     = "https"
}
