# Typesense Terraform Provider Examples

This directory contains a complete example demonstrating how to use the Typesense Terraform provider.

## Chinook Music Database Example

The [`chinook/`](./chinook/) example shows how to build search infrastructure for the [Chinook Database](https://github.com/lerocha/chinook-database) - a sample music store database.

### What It Demonstrates

- **Collections:** 7 denormalized collections (tracks, albums, artists, customers, invoices, employees, playlists)
- **Denormalization patterns:** Embedding related data for optimal search performance
- **Field configuration:** Facets, sorting, infix search, nested objects
- **Synonyms:** Music-specific synonyms for genres, media types, and artist names

### Quick Start

```bash
cd examples/chinook

# Configure your Typesense credentials
export TF_VAR_typesense_host="xxx.a1.typesense.net"
export TF_VAR_typesense_api_key="your-admin-api-key"

# Deploy
terraform init
terraform plan
terraform apply
```

See [`chinook/README.md`](./chinook/README.md) for full documentation including:
- Collection schemas and example documents
- Sample search queries
- Denormalization strategy explanation

## Environment Variables

Instead of using `terraform.tfvars`, you can set credentials via environment variables:

```bash
# Terraform variable format
export TF_VAR_typesense_host="your-cluster.a1.typesense.net"
export TF_VAR_typesense_api_key="your-admin-api-key"

# Or native provider environment variables
export TYPESENSE_HOST="your-cluster.a1.typesense.net"
export TYPESENSE_API_KEY="your-admin-api-key"
```

## Security Notes

- Never commit API keys or `terraform.tfvars` files to version control
- Use environment variables or a secrets manager in production
- Mark sensitive outputs with `sensitive = true`
