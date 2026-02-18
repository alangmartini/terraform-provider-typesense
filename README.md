# Typesense Terraform Provider

A Terraform provider for managing [Typesense](https://typesense.org/) search infrastructure as code. Works with both **Typesense Cloud** and **self-hosted** instances.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Provider Configuration](#provider-configuration)
- [1. Import from Typesense Cloud into Terraform](#1-import-from-typesense-cloud-into-terraform)
- [2. Apply Terraform to a Cloud Cluster](#2-apply-terraform-to-a-cloud-cluster)
- [3. Import from a Local Typesense into Terraform](#3-import-from-a-local-typesense-into-terraform)
- [4. Apply Terraform to a Local Cluster](#4-apply-terraform-to-a-local-cluster)
- [5. Keeping Terraform and Your Cluster in Sync](#5-keeping-terraform-and-your-cluster-in-sync)
- [6. Cluster-to-Cluster Migration](#6-cluster-to-cluster-migration)
- [Available Resources](#available-resources)
- [Import ID Reference](#import-id-reference)
- [Development](#development)

---

## Prerequisites

- **Terraform** >= 1.0
- A **Typesense Cloud** cluster or a **self-hosted Typesense** server (Docker recommended for local)
- An **admin API key** with full permissions

---

## Installation

Add the provider to your Terraform configuration:

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

Then run:

```bash
terraform init
```

---

## Provider Configuration

The provider needs to know where your Typesense instance lives. There are two main setups:

### Cloud Cluster

```hcl
provider "typesense" {
  server_host     = "xyz.a1.typesense.net"   # Your cluster hostname
  server_api_key  = var.typesense_api_key     # Admin API key
  server_port     = 443                       # Default for cloud
  server_protocol = "https"                   # Default for cloud
}
```

### Local / Self-Hosted

```hcl
provider "typesense" {
  server_host     = "localhost"
  server_api_key  = "your-api-key"
  server_port     = 8108                      # Default Docker port
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

All provider settings can be set via environment variables instead:

```bash
export TYPESENSE_HOST="xyz.a1.typesense.net"
export TYPESENSE_API_KEY="your-admin-api-key"
export TYPESENSE_PORT="443"
export TYPESENSE_PROTOCOL="https"
export TYPESENSE_CLOUD_MANAGEMENT_API_KEY="your-cloud-key"
```

**Precedence:** Terraform config > Environment variables > Default values

---

## 1. Import from Typesense Cloud into Terraform

> **Goal:** You have an existing Typesense Cloud cluster with collections, synonyms, overrides, etc., and you want to manage it with Terraform going forward.

### Option A: Bulk Export with `generate` (Recommended)

The provider binary includes a `generate` command that reads all resources from your cluster and creates `.tf` files and import scripts automatically.

```bash
# Build the provider binary (if using from source)
go build -o terraform-provider-typesense .

# Export everything from your cloud cluster
./terraform-provider-typesense generate \
  --host=xyz.a1.typesense.net \
  --port=443 \
  --protocol=https \
  --api-key=YOUR_ADMIN_API_KEY \
  --output=./my-typesense-config
```

This creates two files:

| File | Contents |
|------|----------|
| `my-typesense-config/main.tf` | All resources as Terraform configuration |
| `my-typesense-config/imports.sh` | `terraform import` commands for every resource |

Now bring everything into Terraform state:

```bash
cd my-typesense-config

# Review the generated config
cat main.tf

# Initialize Terraform
terraform init

# Import all existing resources into state
chmod +x imports.sh
./imports.sh

# Verify: should show "No changes"
terraform plan
```

### Option B: Import Individual Resources

If you only want to manage specific resources, write the `.tf` definition first, then import:

```hcl
# main.tf
provider "typesense" {
  server_host     = "xyz.a1.typesense.net"
  server_api_key  = var.typesense_api_key
  server_port     = 443
  server_protocol = "https"
}

resource "typesense_collection" "products" {
  name = "products"

  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "title"
    type  = "string"
    facet = true
    index = true
  }

  field {
    name = "price"
    type = "float"
    sort = true
  }

  default_sorting_field = "price"
}

resource "typesense_synonym" "shoe_synonyms" {
  collection = typesense_collection.products.name
  name       = "shoe-synonyms"
  synonyms   = ["shoe", "sneaker", "trainer", "footwear"]
}
```

Then import each resource:

```bash
terraform init

# Import the collection
terraform import typesense_collection.products products

# Import the synonym (format: collection/synonym_name)
terraform import typesense_synonym.shoe_synonyms products/shoe-synonyms

# Verify everything matches
terraform plan
```

If `terraform plan` shows differences, update your `.tf` file to match the actual cloud state, then run `terraform plan` again until it shows "No changes."

---

## 2. Apply Terraform to a Cloud Cluster

> **Goal:** You have Terraform files defining your search configuration and you want to create or update resources on a Typesense Cloud cluster.

### Step 1: Write Your Configuration

```hcl
# main.tf
terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

variable "typesense_api_key" {
  type      = string
  sensitive = true
}

provider "typesense" {
  server_host     = "xyz.a1.typesense.net"
  server_api_key  = var.typesense_api_key
  server_port     = 443
  server_protocol = "https"
}

# Define a collection
resource "typesense_collection" "articles" {
  name                 = "articles"
  enable_nested_fields = true

  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "title"
    type  = "string"
    index = true
  }

  field {
    name  = "category"
    type  = "string"
    facet = true
  }

  field {
    name = "published_at"
    type = "int64"
    sort = true
  }

  default_sorting_field = "published_at"
}

# Add synonyms
resource "typesense_synonym" "tech_terms" {
  collection = typesense_collection.articles.name
  name       = "tech-synonyms"
  synonyms   = ["javascript", "js", "ecmascript"]
}

# Add stopwords
resource "typesense_stopwords_set" "english" {
  name      = "english-common"
  locale    = "en"
  stopwords = ["the", "a", "an", "and", "or", "but", "is", "are"]
}

# Create a search-only API key (auto-generated value)
resource "typesense_api_key" "frontend_search" {
  description = "Frontend search-only key"
  actions     = ["documents:search"]
  collections = [typesense_collection.articles.name]
}

# Create a key with a specific value (same across prod/staging)
resource "typesense_api_key" "shared_search" {
  description = "Shared search key"
  value       = var.shared_search_key
  actions     = ["documents:search"]
  collections = ["*"]
}

# Create a temporary key that auto-deletes after expiration
resource "typesense_api_key" "temp_key" {
  description = "Temporary ingest key"
  actions     = ["documents:create"]
  collections = ["*"]
  expires_at  = 1735689600
  autodelete  = true
}

output "search_api_key" {
  value     = typesense_api_key.frontend_search.value
  sensitive = true
}

output "search_key_prefix" {
  value = typesense_api_key.frontend_search.value_prefix
}
```

---

## 3. Import from a Local Typesense into Terraform

> **Goal:** You have a local Typesense instance (e.g., Docker) with existing resources, and you want to capture them as Terraform code.

### Start Your Local Typesense (if not running)

```bash
docker run -d \
  --name typesense \
  -p 8108:8108 \
  -v typesense-data:/data \
  typesense/typesense:27.1 \
  --data-dir=/data \
  --api-key=test-api-key
```

### Option A: Bulk Export with `generate`

```bash
./terraform-provider-typesense generate \
  --host=localhost \
  --port=8108 \
  --protocol=http \
  --api-key=test-api-key \
  --output=./local-config
```

Then import into Terraform state:

```bash
cd local-config
terraform init
chmod +x imports.sh
./imports.sh
terraform plan   # Should show "No changes"
```

### Option B: Import Individual Resources

Write your `.tf` file pointing at localhost:

```hcl
# main.tf
provider "typesense" {
  server_host     = "localhost"
  server_api_key  = "test-api-key"
  server_port     = 8108
  server_protocol = "http"
}

resource "typesense_collection" "products" {
  name = "products"

  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "name"
    type  = "string"
    index = true
  }
}
```

Import the existing resource:

```bash
terraform init
terraform import typesense_collection.products products
terraform plan
```

Adjust your `.tf` until `terraform plan` shows no differences.

---

## 4. Apply Terraform to a Local Cluster

> **Goal:** You have Terraform files and want to create resources on a local Typesense instance.

### Step 1: Start a Local Typesense

```bash
docker run -d \
  --name typesense \
  -p 8108:8108 \
  -v typesense-data:/data \
  typesense/typesense:27.1 \
  --data-dir=/data \
  --api-key=test-api-key
```

### Step 2: Write Your Configuration

```hcl
# main.tf
terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

provider "typesense" {
  server_host     = "localhost"
  server_api_key  = "test-api-key"
  server_port     = 8108
  server_protocol = "http"
}

resource "typesense_collection" "products" {
  name = "products"

  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "name"
    type  = "string"
    index = true
    facet = true
  }

  field {
    name = "price"
    type = "float"
    sort = true
  }

  default_sorting_field = "price"
}

resource "typesense_synonym" "shoe_synonyms" {
  collection = typesense_collection.products.name
  name       = "shoe-synonyms"
  synonyms   = ["shoe", "sneaker", "trainer"]
}

resource "typesense_override" "featured" {
  collection = typesense_collection.products.name
  name       = "featured-product"

  rule {
    query = "best"
    match = "exact"
  }

  includes {
    id       = "product-1"
    position = 1
  }
}
```

### Step 3: Apply

```bash
terraform init
terraform plan    # Preview
terraform apply   # Create resources locally
```

### Step 4: Verify

```bash
# List collections
curl "http://localhost:8108/collections" \
  -H "X-TYPESENSE-API-KEY: test-api-key"

# Check synonyms
curl "http://localhost:8108/collections/products/synonyms" \
  -H "X-TYPESENSE-API-KEY: test-api-key"
```

---

## 5. Keeping Terraform and Your Cluster in Sync

Over time, your Terraform state can drift from the actual cluster state (e.g., someone makes a change directly via the API). Here's how to detect and fix drift.

### Key Commands

| Command | What It Does |
|---------|--------------|
| `terraform plan` | Compares your `.tf` files against current state, shows what would change |
| `terraform refresh` | Updates Terraform state to match the real cluster (without changing anything) |
| `terraform apply` | Pushes your `.tf` definitions to the cluster, making it match |
| `terraform import` | Brings an existing resource into state so Terraform can manage it |

### Example: Keeping a Cloud Cluster in Sync

Suppose someone added a new synonym directly via the Typesense API and modified a stopwords set.

**Detect drift:**

```bash
# Refresh state from the actual cloud cluster
terraform refresh -var="typesense_api_key=YOUR_ADMIN_KEY"

# See what's different between your .tf and the cluster
terraform plan -var="typesense_api_key=YOUR_ADMIN_KEY"
```

The plan output tells you exactly what differs:

```
~ typesense_stopwords_set.english will be updated in-place
  ~ stopwords = ["the", "a", "an"] -> ["the", "a", "an", "and", "or"]
```

**Choose how to resolve:**

- **Keep Terraform as source of truth** (revert the manual change):
  ```bash
  terraform apply -var="typesense_api_key=YOUR_ADMIN_KEY"
  ```
  This pushes your `.tf` config back to the cluster, overwriting the manual change.

- **Accept the manual change** (update your `.tf` to match):
  Edit your `.tf` file to include the new stopwords, then verify:
  ```bash
  terraform plan   # Should now show "No changes"
  ```

- **Import the new synonym** that was created outside Terraform:
  Add a resource block for it in your `.tf`:
  ```hcl
  resource "typesense_synonym" "new_manual_synonym" {
    collection = typesense_collection.products.name
    name       = "manually-added-synonym"
    synonyms   = ["laptop", "notebook", "computer"]
  }
  ```
  Then import it:
  ```bash
  terraform import typesense_synonym.new_manual_synonym products/manually-added-synonym
  terraform plan   # Should show "No changes"
  ```

### Example: Keeping a Local Cluster in Sync

Same workflow, just with local connection settings.

**Detect drift:**

```bash
terraform refresh
terraform plan
```

**Resolve by applying Terraform as the source of truth:**

```bash
terraform apply
```

**Or update Terraform to match what's on the local instance:**

```bash
# See what the cluster actually has
curl "http://localhost:8108/collections" \
  -H "X-TYPESENSE-API-KEY: test-api-key" | jq .

# Update your .tf files to match, then verify
terraform plan   # Should show "No changes"
```

### Recommended Sync Workflow

1. **Never make manual API changes in production.** Always go through Terraform.
2. Run `terraform plan` in CI on every PR to catch config drift early.
3. If manual changes are unavoidable, immediately import them into Terraform:
   ```bash
   # Add the resource block to .tf
   # Then import
   terraform import <resource_type>.<name> <import_id>
   terraform plan  # Verify no drift
   ```
4. Use `terraform refresh` before `plan` if you suspect out-of-band changes.

---

## 6. Cluster-to-Cluster Migration

> **Goal:** You want to migrate collections (and optionally documents) from one Typesense cluster to another — for example, from a production cloud cluster to a staging environment, or between two self-hosted instances.

The provider binary includes `generate` and `migrate` commands that work together for cluster-to-cluster migration.

### Step 1: Export from Source Cluster

```bash
# Schema only (collections, synonyms, overrides, stopwords)
./terraform-provider-typesense generate \
  --host=source.typesense.net --port=443 --protocol=https \
  --api-key=SOURCE_API_KEY \
  --output=./migration

# Schema + documents (use --include-data to export document JSONL files)
./terraform-provider-typesense generate \
  --host=source.typesense.net --port=443 --protocol=https \
  --api-key=SOURCE_API_KEY \
  --include-data \
  --output=./migration
```

### Step 2: Import to Target Cluster

```bash
# Schema only (default — safe for any cluster size)
./terraform-provider-typesense migrate \
  --source-dir=./migration \
  --target-host=target.typesense.net --target-port=443 --target-protocol=https \
  --target-api-key=TARGET_API_KEY

# Schema + documents (opt-in via --include-documents)
./terraform-provider-typesense migrate \
  --source-dir=./migration \
  --target-host=target.typesense.net --target-port=443 --target-protocol=https \
  --target-api-key=TARGET_API_KEY \
  --include-documents
```

> **WARNING: `--include-documents` imports ALL document data from the exported JSONL files.**
> If your source cluster has millions of documents, this can take a very long time, consume significant disk space on the target, and use substantial network bandwidth. Only use this flag when you explicitly need to copy document data. Omit it to migrate schema only.

### What Gets Migrated

| Data | `generate` flag | `migrate` flag | Default |
|------|----------------|----------------|---------|
| Collection schemas | Always | Always | Yes |
| Synonyms | Always | Always | Yes |
| Overrides (curations) | Always | Always | Yes |
| Stopwords sets | Always | Always | Yes |
| **Documents** | `--include-data` | `--include-documents` | **No** |

### Migration Between Local Clusters

```bash
# Export from local source (port 8108)
./terraform-provider-typesense generate \
  --host=localhost --port=8108 --protocol=http \
  --api-key=source-api-key \
  --include-data \
  --output=./migration

# Import to local target (port 8109), including documents
./terraform-provider-typesense migrate \
  --source-dir=./migration \
  --target-host=localhost --target-port=8109 --target-protocol=http \
  --target-api-key=target-api-key \
  --include-documents
```
---

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
| `typesense_nl_search_model` | Natural language search models (requires OpenAI key) |
| `typesense_conversation_model` | Conversational search / RAG models (requires OpenAI key) |

---

## Import ID Reference

Each resource type uses a specific ID format for `terraform import`:

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

---

## Development

### Building from Source

```bash
git clone https://github.com/alanm/terraform-provider-typesense.git
cd terraform-provider-typesense
go build -o terraform-provider-typesense .
```

### Running the Chinook Acceptance Tests

The `examples/chinook/` directory is the primary acceptance test suite:

```bash
make chinook-test     # Full cycle: start Typesense, apply, verify, cleanup
make chinook-apply    # Apply only (assumes Typesense is running)
make chinook-destroy  # Tear down chinook resources
```

### CLI Commands

The provider binary also works as a standalone CLI:

```bash
./terraform-provider-typesense generate --help    # Export cluster config to .tf files
./terraform-provider-typesense migrate --help     # Migrate data between clusters
./terraform-provider-typesense version            # Print version
```

| Command | Key Flags | Description |
|---------|-----------|-------------|
| `generate` | `--host`, `--api-key`, `--output` | Export cluster schema to `.tf` files and import scripts |
| `generate` | `--include-data` | Also export documents to JSONL files (off by default) |
| `migrate` | `--source-dir`, `--target-host`, `--target-api-key` | Import schema (synonyms, overrides, stopwords) to target cluster |
| `migrate` | `--include-documents` | Also import documents from JSONL files (off by default) |
| `version` | | Print binary version |

---

## License

MPL-2.0

---

## Links

- [Typesense Documentation](https://typesense.org/docs/)
- [Typesense Cloud](https://cloud.typesense.org/)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
