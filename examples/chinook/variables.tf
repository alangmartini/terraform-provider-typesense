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

# Natural Language Search configuration
variable "openai_api_key" {
  type        = string
  description = "OpenAI API key for natural language search (optional - set to enable NL queries)"
  sensitive   = true
  default     = ""
}

variable "nl_model_name" {
  type        = string
  description = "LLM model to use for natural language queries"
  default     = "openai/gpt-4o-mini"
}

# Optional override for the LLM endpoint, used by the chinook E2E suite to
# redirect Typesense's nl_search_model validation call at an in-process mock
# server instead of api.openai.com. Empty string keeps the default behavior.
variable "mock_openai_url" {
  type        = string
  description = "Override URL for the LLM endpoint (test hook; leave empty in production)"
  default     = ""
}

# Conversation Model (RAG) configuration
variable "conversation_model_name" {
  type        = string
  description = "LLM model to use for conversational search (RAG)"
  default     = "openai/gpt-4o-mini"
}

# Multi-environment API key
variable "shared_search_key" {
  type        = string
  description = "User-provided search key value for consistent keys across environments (optional)"
  sensitive   = true
  default     = ""
}
