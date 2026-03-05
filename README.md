# Typesense Terraform Provider

>[!WARNING]
> **This provider is currently in BETA.** APIs, resource schemas, and behaviors may change between releases.

A Terraform provider for managing [Typesense](https://typesense.org/) search infrastructure as code. Works with both **Typesense Cloud** and **self-hosted** instances.

## Prerequisites

- **Terraform** >= 1.0
- A **Typesense Cloud** cluster or a **self-hosted Typesense** server
- An **admin API key** with full permissions

## Installation

```hcl
terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}
```

```bash
terraform init
```

### Using a Local Build (dev_overrides)

To use a locally built version of the provider instead of downloading from the registry:

```bash
# Build the provider
cd /path/to/terraform-provider-typesense
go build -o terraform-provider-typesense .
```

Create or edit your Terraform CLI config (`~/.terraformrc` on Linux/Mac, `%APPDATA%/terraform.rc` on Windows):

```hcl
provider_installation {
  dev_overrides {
    "alanm/typesense" = "/path/to/terraform-provider-typesense"
  }
  direct {}
}
```

With `dev_overrides`, **skip `terraform init`** — Terraform uses the local binary directly. You'll see a warning about development overrides which is expected.

## Provider Configuration

### Cloud Cluster

```hcl
provider "typesense" {
  server_host     = "xyz.a1.typesense.net"
  server_api_key  = var.typesense_api_key
  server_port     = 443
  server_protocol = "https"
}
```

### Local / Self-Hosted

```hcl
provider "typesense" {
  server_host     = "localhost"
  server_api_key  = "your-api-key"
  server_port     = 8108
  server_protocol = "http"
}
```

### Cloud Management API (for managing clusters themselves)

```hcl
provider "typesense" {
  cloud_management_api_key = "your-cloud-management-key"
}
```

### Environment Variables

All provider settings can be set via environment variables:

```bash
export TYPESENSE_HOST="xyz.a1.typesense.net"
export TYPESENSE_API_KEY="your-admin-api-key"
export TYPESENSE_PORT="443"
export TYPESENSE_PROTOCOL="https"
export TYPESENSE_CLOUD_MANAGEMENT_API_KEY="your-cloud-key"
```

**Precedence:** Terraform config > Environment variables > Default values

## Importing Existing Resources

If you have an existing Typesense cluster and want to manage it with Terraform, you need to import its resources into Terraform state.

### Bulk Import with `generate` (Recommended)

The provider binary includes a `generate` command that reads all resources from your cluster and creates `.tf` files and import scripts automatically.

```bash
# Build the provider binary (if using from source)
go build -o terraform-provider-typesense .

# Export from a cloud cluster
./terraform-provider-typesense generate \
  --host=xyz.a1.typesense.net --port=443 --protocol=https \
  --api-key=YOUR_ADMIN_API_KEY \
  --output=./my-typesense-config

# Or from a local instance
./terraform-provider-typesense generate \
  --host=localhost --port=8108 --protocol=http \
  --api-key=YOUR_API_KEY \
  --output=./my-typesense-config
```

This creates:

| File | Contents |
|------|----------|
| `main.tf` | All resources as Terraform configuration |
| `imports.sh` | `terraform import` commands for every resource |

Then import into Terraform state:

```bash
cd my-typesense-config
terraform init
chmod +x imports.sh
./imports.sh
terraform plan   # Should show "No changes"
```

### Importing Individual Resources

Write the `.tf` definition first, then import:

```bash
terraform import typesense_collection.products products
terraform import typesense_synonym.shoe_synonyms products/shoe-synonyms
terraform plan   # Adjust .tf until this shows "No changes"
```

See [Import ID Reference](#import-id-reference) for the ID format of each resource type.

## Cluster-to-Cluster Migration

The `generate` and `migrate` commands work together for cluster-to-cluster migration.

```bash
# Step 1: Export from source (schema only by default, add --include-data for documents)
./terraform-provider-typesense generate \
  --host=source.typesense.net --port=443 --protocol=https \
  --api-key=SOURCE_API_KEY \
  --include-data \
  --output=./migration

# Step 2: Import to target (schema only by default, add --include-documents for documents)
./terraform-provider-typesense migrate \
  --source-dir=./migration \
  --target-host=target.typesense.net --target-port=443 --target-protocol=https \
  --target-api-key=TARGET_API_KEY \
  --include-documents
```

| Data | `generate` flag | `migrate` flag | Default |
|------|----------------|----------------|---------|
| Collection schemas, synonyms, overrides, stopwords | Always | Always | Yes |
| **Documents** | `--include-data` | `--include-documents` | **No** |

> **Warning:** `--include-data` / `--include-documents` exports/imports ALL documents. For large clusters this can take a long time and use significant disk/bandwidth.

## Keeping Terraform in Sync

```bash
terraform refresh   # Update state from the real cluster
terraform plan      # See what differs between .tf and cluster
terraform apply     # Push .tf config to the cluster
```

If resources were created outside Terraform, add a resource block to your `.tf` and import them:

```bash
terraform import <resource_type>.<name> <import_id>
terraform plan   # Verify no drift
```

**Best practice:** Never make manual API changes in production — always go through Terraform. Run `terraform plan` in CI to catch drift early.

## Available Resources

### Cloud Management

| Resource | Purpose |
|----------|---------|
| `typesense_cluster` | Create and manage Typesense Cloud clusters |
| `typesense_cluster_config_change` | Schedule cluster configuration changes |

### Server Resources

| Resource | Purpose |
|----------|---------|
| `typesense_collection` | Search collections with typed schemas |
| `typesense_collection_alias` | Stable aliases pointing to collections |
| `typesense_synonym` | Search term synonyms (multi-way or one-way) |
| `typesense_override` | Search result curations (pin/hide documents) |
| `typesense_stopwords_set` | Custom stopword lists |
| `typesense_preset` | Saved search parameter presets |
| `typesense_analytics_rule` | Analytics event collection rules |
| `typesense_api_key` | API keys with granular permissions |
| `typesense_stemming_dictionary` | Language-specific stemming rules |
| `typesense_nl_search_model` | Natural language search models |
| `typesense_conversation_model` | Conversational search / RAG models |

## Import ID Reference

| Resource | Import ID Format | Example |
|----------|------------------|---------|
| `typesense_collection` | `{name}` | `terraform import typesense_collection.x products` |
| `typesense_collection_alias` | `{alias_name}` | `terraform import typesense_collection_alias.x music` |
| `typesense_synonym` | `{collection}/{synonym_name}` | `terraform import typesense_synonym.x products/shoe-synonyms` |
| `typesense_override` | `{collection}/{override_name}` | `terraform import typesense_override.x products/featured` |
| `typesense_stopwords_set` | `{set_name}` | `terraform import typesense_stopwords_set.x english` |
| `typesense_preset` | `{preset_name}` | `terraform import typesense_preset.x track-listing` |
| `typesense_analytics_rule` | `{rule_name}` | `terraform import typesense_analytics_rule.x popular-queries` |
| `typesense_api_key` | `{key_id}` | `terraform import typesense_api_key.x 123` |
| `typesense_stemming_dictionary` | `{dictionary_id}` | `terraform import typesense_stemming_dictionary.x english` |
| `typesense_cluster` | `{cluster_id}` | `terraform import typesense_cluster.x abc123` |
| `typesense_nl_search_model` | `{model_id}` | `terraform import typesense_nl_search_model.x music-nl` |
| `typesense_conversation_model` | `{model_id}` | `terraform import typesense_conversation_model.x rag-model` |

## Development

### Building from Source

```bash
git clone https://github.com/alanm/terraform-provider-typesense.git
cd terraform-provider-typesense
go build -o terraform-provider-typesense .
```

### Acceptance Tests

```bash
make chinook-test     # Full cycle: start Typesense, apply, verify, cleanup
make chinook-apply    # Apply only (assumes Typesense is running)
make chinook-destroy  # Tear down chinook resources
```

### CLI Commands

```bash
./terraform-provider-typesense generate --help    # Export cluster config to .tf files
./terraform-provider-typesense migrate --help     # Migrate data between clusters
./terraform-provider-typesense version            # Print version
```

## License

MPL-2.0

## Links

- [Typesense Documentation](https://typesense.org/docs/)
- [Typesense Cloud](https://cloud.typesense.org/)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
