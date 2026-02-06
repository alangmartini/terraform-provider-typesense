# Typesense Terraform Provider Examples

This directory contains examples demonstrating how to use the Typesense Terraform provider for managing search infrastructure.

## Directory Structure

```
examples/
├── provider/           # Provider configuration
├── resources/          # Individual resource examples
│   ├── api_key/        # API key management
│   ├── cluster/        # Typesense Cloud clusters
│   ├── collection/     # Search collections
│   ├── override/       # Search curation rules
│   ├── stopwords/      # Stopwords sets
│   └── synonym/        # Synonym definitions
├── complete/           # All resources together
├── local-test/         # Quick local development setup
└── chinook/            # Real-world music database example
```

## Quick Start

### Prerequisites

- Terraform >= 1.0
- For server resources: A running Typesense server (local or cloud)
- For cloud resources: A Typesense Cloud account and Management API key

### Running an Example

```bash
cd examples/<example-name>
terraform init
terraform plan
terraform apply
```

## Example Descriptions

### [`provider/`](./provider/)

**Purpose:** Shows all provider configuration options.

Demonstrates:
- Cloud Management API key configuration for cluster management
- Server API configuration for collections, synonyms, and other resources
- Environment variable alternatives for sensitive values

Use this as a reference when setting up the provider in your own configurations.

---

### [`resources/collection/`](./resources/collection/)

**Purpose:** Create and configure Typesense collections (search indexes).

Demonstrates:
- Basic product collection with common field types
- Field options: `facet`, `sort`, `infix`, `optional`, `locale`
- Nested fields with `enable_nested_fields = true`
- Custom token separators and symbol indexing
- Geopoint fields for location-based search

---

### [`resources/api_key/`](./resources/api_key/)

**Purpose:** Manage scoped API keys for different access patterns.

Demonstrates:
- Search-only keys for frontend applications
- Write keys for backend indexing services
- Read-only keys for analytics/reporting
- Admin keys for specific collections
- Expiring temporary access keys
- Keys for managing synonyms and overrides

---

### [`resources/synonym/`](./resources/synonym/)

**Purpose:** Improve search recall with synonym definitions.

Demonstrates:
- **Multi-way synonyms:** All terms are equivalent (e.g., "shirt" = "tee" = "t-shirt")
- **One-way synonyms:** Map alternatives to a root term (e.g., "macbook" → "laptop")
- Practical examples for apparel, electronics, sizes, and colors

---

### [`resources/override/`](./resources/override/)

**Purpose:** Curate search results with override rules.

Demonstrates:
- **Pin results:** Force specific documents to appear at set positions
- **Exclude results:** Remove documents from search results
- **Filter injection:** Automatically apply filters for certain queries
- **Query replacement:** Fix misspellings or redirect queries
- **Time-limited overrides:** Promotional rules with start/end timestamps
- **Tag-based rules:** Segment users for A/B testing or VIP access

---

### [`resources/stopwords/`](./resources/stopwords/)

**Purpose:** Define stopwords to ignore during search.

Demonstrates:
- Language-specific stopwords (English, German, French)
- Custom e-commerce stopwords (buy, shop, price, etc.)
- Minimal stopwords sets for precise search requirements
- Locale configuration for language detection

---

### [`resources/cluster/`](./resources/cluster/)

**Purpose:** Manage Typesense Cloud clusters.

Demonstrates:
- Basic cluster provisioning
- Production-ready high-availability clusters
- Scheduled configuration changes (memory, vCPU, version upgrades)
- Cluster outputs (hostname, admin API key)

**Note:** Requires a Typesense Cloud Management API key.

---

### [`complete/`](./complete/)

**Purpose:** Production-ready example combining all resources.

Shows how to:
- Configure the provider for both Cloud and Server APIs
- Create a products collection with rich field configuration
- Define synonyms for apparel and electronics terms
- Set up curation overrides for featured products and filters
- Manage stopwords for English and e-commerce contexts
- Create scoped API keys (frontend search, backend indexing, admin)

This is a good starting point for real-world deployments.

---

### [`local-test/`](./local-test/)

**Purpose:** Quick testing with a local Typesense server.

Pre-configured for:
- `localhost:8108` with HTTP protocol
- API key `xyz` (default for local development)

Start a local Typesense server:

```bash
docker run -d \
  -p 8108:8108 \
  -v /tmp/typesense-data:/data \
  typesense/typesense:28.0 \
  --data-dir /data \
  --api-key=xyz \
  --enable-cors
```

Then apply the example:

```bash
cd examples/local-test
terraform init
terraform apply
```

---

### [`chinook/`](./chinook/)

**Purpose:** Real-world example based on the Chinook music database.

A comprehensive example showing:
- 7 denormalized collections (tracks, albums, artists, customers, invoices, employees, playlists)
- Proper handling of relational data in a document store
- Nested objects and embedded documents
- Music-specific synonyms (genres, media types, artists)
- Full documentation with sample queries

See [`chinook/README.md`](./chinook/README.md) for detailed documentation.

## Environment Variables

Instead of hardcoding values in `terraform.tfvars`, you can use environment variables:

```bash
# For Typesense Cloud cluster management
export TYPESENSE_CLOUD_MANAGEMENT_API_KEY="your-cloud-api-key"

# For server API operations
export TYPESENSE_HOST="your-cluster.a1.typesense.net"
export TYPESENSE_API_KEY="your-admin-api-key"
export TYPESENSE_PORT="443"
export TYPESENSE_PROTOCOL="https"
```

Or use `TF_VAR_` prefix for Terraform variables:

```bash
export TF_VAR_typesense_host="your-cluster.a1.typesense.net"
export TF_VAR_typesense_api_key="your-admin-api-key"
```

## Security Notes

- Never commit API keys or `terraform.tfvars` files to version control
- Use environment variables or a secrets manager in production
- Create scoped API keys with minimal required permissions
- Mark sensitive outputs with `sensitive = true`
