# Typesense Terraform Provider

A Terraform provider for managing [Typesense](https://typesense.org/) Cloud clusters and server resources. This provider allows you to define and manage your Typesense infrastructure as code.

## Table of Contents

- [What is This Project?](#what-is-this-project)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Available Resources](#available-resources)
- [How It Works (For Non-Go Developers)](#how-it-works-for-non-go-developers)
- [Development](#development)
- [Building from Source](#building-from-source)

---

## What is This Project?

This is a **Terraform provider** that acts as a bridge between Terraform (infrastructure-as-code tool) and Typesense (a fast, typo-tolerant search engine). It allows you to:

- **Create and manage Typesense Cloud clusters** (infrastructure)
- **Define search collections** (like database tables/indexes)
- **Configure search features** (synonyms, search overrides, stopwords)
- **Manage API keys** with fine-grained permissions

Think of it as a translator: Terraform speaks one language, Typesense speaks another, and this provider translates between them.

---

## Project Structure

```
typesense-terraform/
│
├── main.go                          # Entry point - starts the provider plugin
├── go.mod & go.sum                  # Dependency management (like package.json or requirements.txt)
│
├── internal/                        # Core implementation (private to this project)
│   │
│   ├── provider/
│   │   └── provider.go              # Provider configuration & setup
│   │
│   ├── client/                      # HTTP clients for Typesense APIs
│   │   ├── cloud_client.go          # Talks to Cloud Management API (clusters)
│   │   └── server_client.go         # Talks to Typesense Server API (collections, etc.)
│   │
│   ├── types/
│   │   └── types.go                 # Shared data structures
│   │
│   └── resources/                   # Terraform resource implementations
│       ├── cluster.go               # Cloud cluster management
│       ├── cluster_config.go        # Cluster configuration changes
│       ├── collection.go            # Search collections/indexes
│       ├── synonym.go               # Search synonyms
│       ├── override.go              # Search result curation
│       ├── stopwords.go             # Custom stopwords
│       └── api_key.go               # API key management
│
├── examples/                        # Usage examples
│   ├── complete/                    # Full example with all resources
│   ├── provider/                    # Provider configuration examples
│   └── resources/                   # Individual resource examples
│
└── docs/                            # Generated documentation
```

### Key Directories Explained

- **`main.go`**: The program's entry point. When Terraform runs, it starts here.
- **`internal/`**: Contains all the implementation code. The name "internal" is a Go convention meaning "private to this module."
- **`internal/client/`**: HTTP clients that make API calls to Typesense services.
- **`internal/resources/`**: Each file implements one Terraform resource (like `typesense_collection`).
- **`examples/`**: Sample Terraform configurations showing how to use the provider.

---

## Prerequisites

### To Use This Provider

- **Terraform** >= 1.0
- A **Typesense Cloud account** (for cluster management) OR a **Typesense server** (for server resources)
- API keys for authentication

### To Build/Develop This Provider

- **Go** >= 1.21
- **Terraform** >= 1.0 (for testing)
- Basic understanding of REST APIs

---

## Installation

### Option 1: From Terraform Registry (When Published)

Add to your Terraform configuration:

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

### Option 2: Local Development Build

```bash
# Clone the repository
git clone <repository-url>
cd typesense-terraform

# Build the provider binary
go build -o terraform-provider-typesense

# Create Terraform plugins directory
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/alanm/typesense/0.1.0/linux_amd64

# Copy the binary
cp terraform-provider-typesense ~/.terraform.d/plugins/registry.terraform.io/alanm/typesense/0.1.0/linux_amd64/

# In your Terraform project, use it
terraform init
```

---

## Configuration

The provider supports **two operational modes**:

### 1. Cloud Management (For Managing Clusters)

```hcl
provider "typesense" {
  cloud_management_api_key = "your-cloud-api-key"
}
```

### 2. Server API (For Managing Collections, Synonyms, etc.)

```hcl
provider "typesense" {
  server_host     = "xxx.a1.typesense.net"  # Your cluster hostname
  server_api_key  = "your-api-key"
  server_port     = 443                      # Optional, defaults to 443
  server_protocol = "https"                  # Optional, defaults to "https"
}
```

### 3. Combined (Both Modes)

```hcl
provider "typesense" {
  # For cluster management
  cloud_management_api_key = "your-cloud-api-key"

  # For server resources
  server_host    = "xxx.a1.typesense.net"
  server_api_key = "your-api-key"
}
```

### Environment Variables (Alternative)

You can also configure via environment variables:

```bash
export TYPESENSE_CLOUD_MANAGEMENT_API_KEY="your-cloud-api-key"
export TYPESENSE_HOST="xxx.a1.typesense.net"
export TYPESENSE_API_KEY="your-api-key"
export TYPESENSE_PORT="443"
export TYPESENSE_PROTOCOL="https"
```

**Precedence**: Terraform configuration > Environment variables > Default values

---

## Usage Examples

### Example 1: Create a Search Collection

```hcl
resource "typesense_collection" "products" {
  name = "products"

  # Define schema fields (like database columns)
  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "title"
    type  = "string"
    facet = true     # Enable faceting for filtering
    index = true     # Index for full-text search
  }

  field {
    name = "price"
    type = "float"
    sort = true      # Allow sorting by this field
  }

  field {
    name     = "tags"
    type     = "string[]"  # Array of strings
    facet    = true
    optional = true        # Field is optional
  }

  default_sorting_field = "price"
}
```

### Example 2: Add Search Synonyms

```hcl
resource "typesense_synonym" "shoe_synonyms" {
  collection = typesense_collection.products.name
  name       = "shoe-synonyms"
  synonyms   = ["shoe", "sneaker", "trainer", "footwear"]
}
```

### Example 3: Create Search Override (Curated Results)

```hcl
resource "typesense_override" "featured_products" {
  collection = typesense_collection.products.name
  name       = "featured-iphone"

  # When user searches for "iphone"
  rule {
    query = "iphone"
    match = "exact"
  }

  # Pin these documents to the top
  includes {
    id       = "product-123"
    position = 1
  }

  includes {
    id       = "product-456"
    position = 2
  }

  # Hide these documents from results
  excludes {
    id = "product-outdated"
  }
}
```

### Example 4: Manage API Keys

```hcl
resource "typesense_api_key" "search_only_key" {
  description = "Frontend search-only key"
  actions     = ["documents:search"]
  collections = [typesense_collection.products.name]
  expires_at  = 0  # Never expires (use Unix timestamp for expiration)
}

# Use the generated key (sensitive value)
output "search_api_key" {
  value     = typesense_api_key.search_only_key.value
  sensitive = true
}
```

### Example 5: Create a Cloud Cluster

```hcl
resource "typesense_cluster" "production" {
  name                    = "prod-cluster"
  memory                  = "4_gb"
  vcpu                    = "2_vcpus"
  high_availability       = true
  typesense_server_version = "27.1"

  regions = ["us-east-1"]

  search_delivery_network = true
  auto_upgrade_capacity   = true
}

# Output cluster details
output "cluster_hostname" {
  value = typesense_cluster.production.load_balanced_hostname
}

output "admin_key" {
  value     = typesense_cluster.production.admin_api_key
  sensitive = true
}
```

### Example 6: Complete Workflow

```hcl
terraform {
  required_providers {
    typesense = {
      source = "alanm/typesense"
    }
  }
}

provider "typesense" {
  server_host    = "my-cluster.a1.typesense.net"
  server_api_key = var.admin_api_key
}

# 1. Create collection
resource "typesense_collection" "articles" {
  name = "articles"

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
    name  = "content"
    type  = "string"
    index = true
  }

  field {
    name = "published_at"
    type = "int64"
    sort = true
  }

  default_sorting_field = "published_at"
}

# 2. Add synonyms
resource "typesense_synonym" "tech_terms" {
  collection = typesense_collection.articles.name
  name       = "tech-synonyms"
  synonyms   = ["javascript", "js", "ecmascript"]
}

# 3. Add stopwords
resource "typesense_stopwords_set" "english" {
  name      = "english-stopwords"
  locale    = "en"
  stopwords = ["the", "a", "an", "and", "or", "but"]
}

# 4. Create search-only API key
resource "typesense_api_key" "public_search" {
  description = "Public search key"
  actions     = ["documents:search"]
  collections = [typesense_collection.articles.name]
}
```

---

## Available Resources

### Cloud Management Resources

| Resource | Purpose |
|----------|---------|
| `typesense_cluster` | Create and manage Typesense Cloud clusters |
| `typesense_cluster_config_change` | Schedule cluster configuration changes |

### Server Resources

| Resource | Purpose |
|----------|---------|
| `typesense_collection` | Define search collections with schema |
| `typesense_synonym` | Configure search synonyms |
| `typesense_override` | Curate search results (pin/hide documents) |
| `typesense_stopwords_set` | Define custom stopwords |
| `typesense_api_key` | Manage API keys with granular permissions |

### Detailed Resource Documentation

For detailed documentation on each resource, see the [`examples/resources/`](examples/resources/) directory or run:

```bash
terraform providers schema -json
```

---

## How It Works (For Non-Go Developers)

If you're coming from Python, JavaScript, Java, or similar languages, here's how this Go project works:

### 1. **Project Entry Point: `main.go`**

**Like**: `if __name__ == "__main__"` (Python) or `public static void main` (Java)

```go
func main() {
    // This function runs when the binary starts
    // It creates a plugin server that Terraform communicates with
}
```

### 2. **Package Structure: `internal/`**

**Like**: Private modules in Python or private packages in Java

Go uses directory names as package names. The `internal/` directory is special - code inside can only be imported by this project (enforced by Go).

### 3. **Provider: `internal/provider/provider.go`**

**Like**: A plugin interface or adapter pattern

This file implements the "Provider" interface required by Terraform. It:
- Defines configuration options (API keys, hosts, etc.)
- Initializes HTTP clients
- Registers available resources

**Key Concepts:**
```go
type TypesenseProvider struct{}

// Schema defines what users can configure
func (p *TypesenseProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    // Returns: cloud_management_api_key, server_host, etc.
}

// Configure runs when Terraform initializes the provider
func (p *TypesenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    // Creates HTTP clients and stores them for resources to use
}
```

### 4. **HTTP Clients: `internal/client/`**

**Like**: Axios (JavaScript), Requests (Python), HttpClient (Java)

Two HTTP clients that make REST API calls:

```go
type CloudClient struct {
    BaseURL    string
    APIKey     string
    HTTPClient *http.Client
}

func (c *CloudClient) CreateCluster(cluster *Cluster) (*Cluster, error) {
    // Makes POST request to https://cloud.typesense.org/api/v1/clusters
    // Returns the created cluster or an error
}
```

**Error Handling**: Go doesn't have try/catch. Instead, functions return `(result, error)`:
```go
cluster, err := client.CreateCluster(newCluster)
if err != nil {
    // Handle error
}
// Use cluster
```

### 5. **Resources: `internal/resources/`**

**Like**: Model classes with CRUD methods

Each resource file (e.g., `collection.go`) implements:

```go
type CollectionResource struct {
    client *ServerClient
}

// Schema defines the resource structure (what fields it has)
func (r *CollectionResource) Schema(...) schema.Schema {
    // Returns: name, fields, default_sorting_field, etc.
}

// Create runs when Terraform creates this resource
func (r *CollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // 1. Read configuration from Terraform
    // 2. Make API call to Typesense
    // 3. Save result to Terraform state
}

// Read runs to refresh the resource state
func (r *CollectionResource) Read(...) { }

// Update runs when configuration changes
func (r *CollectionResource) Update(...) { }

// Delete runs when resource is destroyed
func (r *CollectionResource) Delete(...) { }
```

### 6. **Data Flow**

```
User writes Terraform config (.tf files)
           ↓
Terraform CLI reads config
           ↓
Terraform calls provider plugin (this project)
           ↓
Provider's Create/Update/Delete methods run
           ↓
Provider makes HTTP API calls to Typesense
           ↓
Results saved to Terraform state file
```

### 7. **Key Go Concepts**

#### Structs (Like Classes/Objects)
```go
type Collection struct {
    Name   string
    Fields []Field
}

// Creating an instance
collection := Collection{
    Name: "products",
    Fields: []Field{ /* ... */ },
}
```

#### Pointers (Like References)
```go
func ModifyCollection(c *Collection) {
    // The * means "pointer to Collection"
    // Changes affect the original
    c.Name = "new-name"
}
```

#### Interfaces (Like Abstract Classes)
```go
type Resource interface {
    Create(...)
    Read(...)
    Update(...)
    Delete(...)
}

// Any struct with these methods implements Resource
```

#### Error Handling
```go
result, err := DoSomething()
if err != nil {
    return fmt.Errorf("failed: %w", err)  // Wrap and return error
}
// Success - use result
```

#### JSON Marshaling
```go
type Person struct {
    Name string `json:"name"`  // Maps to/from JSON "name" field
    Age  int    `json:"age"`
}

// Encode to JSON
data, _ := json.Marshal(person)

// Decode from JSON
json.Unmarshal(data, &person)
```

### 8. **Dependency Management: `go.mod`**

**Like**: `package.json` (Node.js), `requirements.txt` (Python), `pom.xml` (Java)

```
module github.com/alanm/typesense-terraform

go 1.21

require (
    github.com/hashicorp/terraform-plugin-framework v1.4.2
    // Other dependencies...
)
```

Run `go mod tidy` to install/update dependencies (like `npm install` or `pip install`).

### 9. **Building the Project**

```bash
# Compile Go code into a binary
go build -o terraform-provider-typesense

# The binary is now executable
./terraform-provider-typesense
```

**No runtime needed** - Go compiles to a native binary (unlike Python/Node.js which need interpreters).

### 10. **Testing (Go Way)**

```bash
# Run tests (files named *_test.go)
go test ./...

# With verbose output
go test -v ./...

# With coverage
go test -cover ./...
```

Test files sit alongside source files:
```
collection.go       # Implementation
collection_test.go  # Tests
```

---

## Development

### Running Locally

1. **Build the provider**:
   ```bash
   go build -o terraform-provider-typesense
   ```

2. **Install locally** (Linux/macOS):
   ```bash
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/alanm/typesense/0.1.0/linux_amd64
   cp terraform-provider-typesense ~/.terraform.d/plugins/registry.terraform.io/alanm/typesense/0.1.0/linux_amd64/
   ```

3. **Test with Terraform**:
   ```bash
   cd examples/complete
   terraform init
   terraform plan
   terraform apply
   ```

### Adding a New Resource

1. Create a new file in `internal/resources/` (e.g., `my_resource.go`)
2. Implement the `Resource` interface:
   - `Metadata()` - resource type name
   - `Schema()` - define attributes
   - `Create()`, `Read()`, `Update()`, `Delete()` - CRUD operations
   - `Configure()` - get client from provider
3. Register the resource in `internal/provider/provider.go`:
   ```go
   func (p *TypesenseProvider) Resources(ctx context.Context) []func() resource.Resource {
       return []func() resource.Resource{
           // Existing resources...
           resources.NewMyResource,  // Add your resource
       }
   }
   ```
4. Add an example in `examples/resources/typesense_my_resource/`

### Project Dependencies

Key libraries used:

- **terraform-plugin-framework**: The official Terraform plugin SDK
- **net/http**: Go's standard HTTP client library (no external HTTP library needed)
- **encoding/json**: JSON encoding/decoding (built-in)
- **context**: Request lifecycle management (built-in)

---

## Building from Source

### Requirements

- Go 1.21 or higher
- Git

### Steps

```bash
# Clone repository
git clone <repository-url>
cd typesense-terraform

# Download dependencies
go mod download

# Build binary
go build -o terraform-provider-typesense

# Verify build
./terraform-provider-typesense --version
```

### For Release (Cross-Platform)

Using GoReleaser (if configured):

```bash
goreleaser release --snapshot --clean
```

Or manually for multiple platforms:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o terraform-provider-typesense_linux_amd64

# macOS
GOOS=darwin GOARCH=amd64 go build -o terraform-provider-typesense_darwin_amd64
GOOS=darwin GOARCH=arm64 go build -o terraform-provider-typesense_darwin_arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o terraform-provider-typesense_windows_amd64.exe
```

---

## Contributing

Contributions are welcome! Please:

1. **Create a feature branch** for your work (see `CLAUDE.md` for git workflow)
2. **Make atomic commits** (one logical change per commit)
3. **Test your changes** locally with Terraform
4. **Update examples** if adding new features
5. **Submit a pull request**

---

## License

This project is licensed under the MPL-2.0 License.

---

## Resources

- [Typesense Documentation](https://typesense.org/docs/)
- [Typesense Cloud API](https://cloud.typesense.org/docs/)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [Go Programming Language](https://go.dev/doc/)

---

## Support

For issues and questions:
- Open an issue in the GitHub repository
- Check existing examples in the `examples/` directory
- Consult Typesense and Terraform documentation

---

**Happy Infrastructure-as-Coding!**
