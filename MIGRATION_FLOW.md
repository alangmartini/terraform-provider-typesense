# Typesense Cluster Migration Flow

This document describes the complete workflow for exporting configuration from a running Typesense cluster and importing it into a new one.

## Overview

```
┌─────────────────────┐         ┌─────────────────────┐         ┌─────────────────────┐
│   Source Cluster    │         │    Generate Tool    │         │   Target Cluster    │
│   (Running)         │ ──────> │   (Export Config)   │ ──────> │   (New)             │
└─────────────────────┘         └─────────────────────┘         └─────────────────────┘
         │                               │                               │
         │ API Calls                     │ Generates                     │ Terraform
         │ - List Collections            │ - main.tf                     │ - init
         │ - List Synonyms               │ - imports.sh                  │ - import (if existing)
         │ - List Overrides              │                               │ - apply
         │ - List Stopwords              │                               │
         └───────────────────────────────┘                               │
                                                                         │
```

---

## Phase 1: Export Configuration from Source Cluster

### Command

```bash
terraform-provider-typesense generate \
  --host=source-cluster.example.com \
  --port=8108 \
  --protocol=https \
  --api-key=<SOURCE_API_KEY> \
  --output=./generated
```

### Implementation

| Component | File | Lines |
|-----------|------|-------|
| CLI Entry Point | `cmd/generate/generate.go` | 1-100 |
| Generator Orchestration | `internal/generator/generator.go` | 54-127 |

### What Gets Exported

1. **Collections** (schema only, not documents)
2. **Synonyms** (v29: per-collection, v30+: system-level synonym sets)
3. **Overrides/Curations** (v29: per-collection, v30+: system-level curation sets)
4. **Stopwords Sets**
5. **Clusters** (if using Typesense Cloud)

### Data Fetching Implementation

| Data | Method | File | Lines |
|------|--------|------|-------|
| Collections | `ListCollections()` | `internal/client/server_client.go` | 745-770 |
| Synonyms (v29) | `ListSynonyms()` | `internal/client/server_client.go` | 775-810 |
| Synonym Sets (v30+) | `ListSynonymSets()` | `internal/client/server_client.go` | 679-709 |
| Overrides (v29) | `ListOverrides()` | `internal/client/server_client.go` | 815-850 |
| Curation Sets (v30+) | `ListCurationSets()` | `internal/client/server_client.go` | 712-742 |
| Stopwords Sets | `ListStopwordsSets()` | `internal/client/server_client.go` | 853-878 |
| Clusters (Cloud) | `ListClusters()` | `internal/client/cloud_client.go` | 324-349 |

---

## Phase 2: HCL Generation (Terraform Files)

### Output Files

1. **`main.tf`** - Complete Terraform configuration
2. **`imports.sh`** - Shell script with terraform import commands

### HCL Block Generation

| Resource Type | Generator Function | File | Lines |
|---------------|-------------------|------|-------|
| Terraform block | `generateTerraformBlock()` | `internal/generator/hcl.go` | 13-20 |
| Provider block | `generateProviderBlock()` | `internal/generator/hcl.go` | 23-33 |
| Collection | `generateCollectionBlock()` | `internal/generator/hcl.go` | 36-95 |
| Synonym | `generateSynonymBlock()` | `internal/generator/hcl.go` | 98-125 |
| Override | `generateOverrideBlock()` | `internal/generator/hcl.go` | 128-201 |
| Stopwords Set | `generateStopwordsBlock()` | `internal/generator/hcl.go` | 204-223 |
| Cluster (Cloud) | `generateClusterBlock()` | `internal/generator/hcl.go` | 226-254 |

### Import Script Generation

| Component | Function | File | Lines |
|-----------|----------|------|-------|
| Script Generator | `GenerateImportScript()` | `internal/generator/imports.go` | 16-35 |
| Collection Import ID | `CollectionImportID()` | `internal/generator/imports.go` | 38-40 |
| Synonym Import ID | `SynonymImportID()` | `internal/generator/imports.go` | 43-45 |
| Override Import ID | `OverrideImportID()` | `internal/generator/imports.go` | 48-50 |
| Stopwords Import ID | `StopwordsImportID()` | `internal/generator/imports.go` | 53-55 |
| Cluster Import ID | `ClusterImportID()` | `internal/generator/imports.go` | 58-60 |

### Resource Name Sanitization

| Function | Purpose | File | Lines |
|----------|---------|------|-------|
| `SanitizeResourceName()` | Convert Typesense names to valid Terraform names | `internal/generator/names.go` | 21-51 |
| `MakeUniqueResourceName()` | Ensure no duplicate resource names | `internal/generator/names.go` | 54-72 |

---

## Phase 3: Configure for Target Cluster

### Edit `main.tf`

Update the provider block to point to your target cluster:

```hcl
provider "typesense" {
  host     = "target-cluster.example.com"  # Change this
  port     = 8108
  protocol = "https"
  api_key  = var.typesense_api_key         # Use the TARGET cluster's API key
}
```

### Provider Configuration Implementation

| Component | File | Lines |
|-----------|------|-------|
| Provider Schema | `internal/provider/provider.go` | 50-78 |
| Client Initialization | `internal/provider/provider.go` | 80-110 |
| Resource Registration | `internal/provider/provider.go` | 113-121 |

### Environment Variables

The provider supports these environment variables:
- `TYPESENSE_HOST`
- `TYPESENSE_PORT`
- `TYPESENSE_PROTOCOL`
- `TYPESENSE_API_KEY`
- `TYPESENSE_CLOUD_MANAGEMENT_API_KEY`

---

## Phase 4: Initialize Terraform

```bash
cd generated
terraform init
```

This downloads the Typesense provider and prepares the working directory.

---

## Phase 5: Import or Apply

### Option A: Fresh Cluster (No Existing Resources)

```bash
terraform apply
```

Terraform will create all resources from scratch.

### Option B: Existing Cluster (Import State First)

```bash
# Run the generated import script
chmod +x imports.sh
./imports.sh

# Verify the plan
terraform plan

# Apply any changes
terraform apply
```

### Import State Implementation (Terraform Resources)

| Resource | Import Method | File | Lines |
|----------|---------------|------|-------|
| Collection | `ImportState()` | `internal/resources/collection.go` | 326-327 |
| Synonym | `ImportState()` | `internal/resources/synonym.go` | 228-242 |
| Override | `ImportState()` | `internal/resources/override.go` | 306-320 |
| Stopwords Set | `ImportState()` | `internal/resources/stopwords.go` | 217-220 |
| Cluster | `ImportState()` | `internal/resources/cluster.go` | 278-279 |
| API Key | `ImportState()` | `internal/resources/api_key.go` | 250-251 |

### Import ID Formats

| Resource | Format | Example |
|----------|--------|---------|
| Collection | `{collection_name}` | `products` |
| Synonym | `{collection}/{synonym_id}` | `products/shoe-synonyms` |
| Override | `{collection}/{override_id}` | `products/featured-items` |
| Stopwords Set | `{stopwords_id}` | `english-stopwords` |
| Cluster | `{cluster_id}` | `abc123` |

---

## Complete Example Workflow

```bash
# Step 1: Export from source cluster
terraform-provider-typesense generate \
  --host=source.typesense.example.com \
  --api-key=s5rbKtkWDCu4S3zmMGBzeJzqobiYQvOM \
  --output=./migration

# Step 2: Review generated files
cd migration
cat main.tf
cat imports.sh

# Step 3: Update main.tf with target cluster credentials
# Edit the provider block to use target cluster host/api-key

# Step 4: Initialize Terraform
terraform init

# Step 5a: For fresh target cluster
terraform apply

# Step 5b: For target with existing resources
./imports.sh
terraform plan
terraform apply
```

---

## What is NOT Exported

| Item | Reason |
|------|--------|
| **API Key Values** | Security - only placeholder comments in generated config |
| **Document Data** | Only schema is exported, not the actual documents |
| **Cluster API Keys** | Stored separately, not included in export |

---

## Version Compatibility

| Typesense Version | Synonyms | Overrides |
|-------------------|----------|-----------|
| v29 | Per-collection (`/collections/{name}/synonyms`) | Per-collection (`/collections/{name}/overrides`) |
| v30+ | System-level (`/synonym_sets`) | System-level (`/curation_sets`) |

The generator automatically detects the version and uses the appropriate API endpoints.

---

## File Structure Summary

```
terraform-provider-typesense/
├── cmd/generate/
│   └── generate.go              # CLI entry point
├── internal/
│   ├── generator/
│   │   ├── generator.go         # Export orchestration
│   │   ├── hcl.go               # Terraform HCL generation
│   │   ├── imports.go           # Import script generation
│   │   └── names.go             # Resource name sanitization
│   ├── client/
│   │   ├── server_client.go     # Typesense Server API client
│   │   └── cloud_client.go      # Typesense Cloud API client
│   ├── provider/
│   │   └── provider.go          # Terraform provider config
│   └── resources/
│       ├── collection.go        # Collection resource
│       ├── synonym.go           # Synonym resource
│       ├── override.go          # Override resource
│       ├── stopwords.go         # Stopwords resource
│       ├── cluster.go           # Cluster resource (Cloud)
│       ├── cluster_config.go    # Cluster config resource
│       └── api_key.go           # API key resource
```
