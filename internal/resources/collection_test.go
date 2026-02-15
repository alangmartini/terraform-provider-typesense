package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCollectionResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "2"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.0.name", "id"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.0.type", "string"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "title"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.type", "string"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "num_documents"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "created_at"),
				),
			},
			{
				ResourceName:            "typesense_collection.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"field"}, // Typesense treats 'id' as implicit and doesn't return it in schema
			},
		},
	})
}

func TestAccCollectionResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionResourceConfig_full(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "default_sorting_field", "rating"),
					resource.TestCheckResourceAttr("typesense_collection.test", "enable_nested_fields", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "5"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.0.name", "id"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "title"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.facet", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.infix", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "description"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.optional", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.name", "rating"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.type", "int32"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.4.name", "author"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.4.locale", "en"),
				),
			},
			{
				ResourceName:            "typesense_collection.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"field"}, // Typesense treats 'id' as implicit and doesn't return it in schema
			},
		},
	})
}

func TestAccCollectionResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "2"),
				),
			},
			{
				Config: testAccCollectionResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "3"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "author"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.type", "string"),
				),
			},
		},
	})
}

func testAccCollectionResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }
}
`, name)
}

func testAccCollectionResourceConfig_full(name string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name                   = %[1]q
  default_sorting_field  = "rating"
  enable_nested_fields   = true

  field {
    name = "id"
    type = "string"
  }

  field {
    name  = "title"
    type  = "string"
    facet = true
    infix = true
  }

  field {
    name     = "description"
    type     = "string"
    optional = true
  }

  field {
    name = "rating"
    type = "int32"
    sort = true
  }

  field {
    name   = "author"
    type   = "string"
    locale = "en"
  }
}
`, name)
}

func testAccCollectionResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }

  field {
    name = "author"
    type = "string"
  }
}
`, name)
}

// TestAccCollectionResource_numericWithoutSort tests that numeric fields
// without explicit sort configuration work correctly with Typesense's
// server-side defaults (sort=true for numeric types).
func TestAccCollectionResource_numericWithoutSort(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionResourceConfig_numericWithoutSort(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "4"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.0.name", "id"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "title"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "count"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.type", "int32"),
					// Typesense defaults sort=true for numeric types
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.name", "price"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.type", "float"),
					// Typesense defaults sort=true for numeric types
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.sort", "true"),
				),
			},
		},
	})
}

func testAccCollectionResourceConfig_numericWithoutSort(name string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }

  # Numeric field without explicit sort - Typesense defaults to sort=true
  field {
    name = "count"
    type = "int32"
  }

  # Float field without explicit sort - Typesense defaults to sort=true
  field {
    name     = "price"
    type     = "float"
    optional = true
  }
}
`, name)
}

// TestAccCollectionResource_vectorSearch tests creating a collection with
// vector search fields (num_dim, vec_dist).
func TestAccCollectionResource_vectorSearch(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-vector")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }

  field {
    name     = "embedding"
    type     = "float[]"
    num_dim  = 384
    vec_dist = "cosine"
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "3"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "embedding"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.type", "float[]"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.num_dim", "384"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.vec_dist", "cosine"),
				),
			},
		},
	})
}

// TestAccCollectionResource_stemRangeIndexStore tests creating a collection with
// stem, range_index, and store field attributes.
func TestAccCollectionResource_stemRangeIndexStore(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-attrs")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
    stem = true
  }

  field {
    name        = "price"
    type        = "float"
    range_index = true
  }

  field {
    name  = "raw_data"
    type  = "string"
    store = false
    index = false
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "4"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "title"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.stem", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "price"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.range_index", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.name", "raw_data"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.store", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.index", "false"),
				),
			},
		},
	})
}

// TestAccCollectionResource_fieldLevelSeparators tests creating a collection with
// field-level token_separators and symbols_to_index.
func TestAccCollectionResource_fieldLevelSeparators(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-seps")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name             = "sku"
    type             = "string"
    token_separators = ["-", "_"]
    symbols_to_index = ["#", "+"]
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "2"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "sku"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.token_separators.#", "2"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.token_separators.0", "-"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.token_separators.1", "_"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.symbols_to_index.#", "2"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.symbols_to_index.0", "#"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.symbols_to_index.1", "+"),
				),
			},
		},
	})
}

// TestAccCollectionResource_collectionMetadata tests creating a collection with
// collection-level metadata and voice_query_model.
func TestAccCollectionResource_collectionMetadata(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-meta")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name     = %[1]q
  metadata = jsonencode({ version = "1.0", team = "search" })

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "metadata"),
				),
			},
		},
	})
}

// TestAccCollectionResource_updateWithNewAttrs tests updating a collection to add
// a new field with the new attributes (stem, range_index).
func TestAccCollectionResource_updateWithNewAttrs(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-upd-attrs")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "2"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[1]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }

  field {
    name = "description"
    type = "string"
    stem = true
  }

  field {
    name        = "price"
    type        = "float"
    range_index = true
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "4"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "description"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.stem", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.name", "price"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.range_index", "true"),
				),
			},
		},
	})
}
