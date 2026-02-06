package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// SERVER-SIDE DEFAULT CONSISTENCY TESTS
// =============================================================================
// These tests verify that Terraform correctly handles Typesense's server-side
// defaults. The key issue is that when users don't specify optional fields,
// Typesense may apply defaults that differ from Terraform's assumptions.
//
// Common pitfalls:
// - Numeric fields (int32, int64, float) default sort=true on server
// - Bool fields default sort=true on server
// - String fields default sort=false on server
// - If Terraform assumes false but server returns true, we get "inconsistent result"

// TestAccCollectionResource_minimalConfig tests collection creation with only
// required fields. This catches server-side default mismatches because we don't
// specify any optional field attributes.
func TestAccCollectionResource_minimalConfig(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-minimal")

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
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "1"),
					// Verify computed defaults are set (not unknown)
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.0.sort"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.0.facet"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.0.optional"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.0.index"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.0.infix"),
				),
			},
		},
	})
}

// TestAccCollectionResource_numericFieldTypes tests all numeric field types
// without explicit sort configuration. Typesense defaults sort=true for
// numeric types, which previously caused "inconsistent result" errors.
func TestAccCollectionResource_numericFieldTypes(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-numeric")

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

  # int32 without explicit sort - server defaults to sort=true
  field {
    name = "count_int32"
    type = "int32"
  }

  # int64 without explicit sort - server defaults to sort=true
  field {
    name = "count_int64"
    type = "int64"
  }

  # float without explicit sort - server defaults to sort=true
  field {
    name = "price_float"
    type = "float"
  }

  # bool without explicit sort - server defaults to sort=true
  field {
    name = "is_active"
    type = "bool"
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "5"),
					// Verify server-side defaults for numeric types
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "count_int32"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "count_int64"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.name", "price_float"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.4.name", "is_active"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.4.sort", "true"),
				),
			},
		},
	})
}

// TestAccCollectionResource_stringFieldTypes tests string field types
// without explicit sort configuration. Typesense defaults sort=false for
// string types.
func TestAccCollectionResource_stringFieldTypes(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-string")

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

  # string without explicit sort - server defaults to sort=false
  field {
    name = "title"
    type = "string"
  }

  # string[] without explicit sort - server defaults to sort=false
  field {
    name = "tags"
    type = "string[]"
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "3"),
					// Verify server-side defaults for string types
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "title"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.sort", "false"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.name", "tags"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.sort", "false"),
				),
			},
		},
	})
}

// TestAccCollectionResource_explicitSortFalse tests that explicitly setting
// sort=false works for numeric types (overriding server default of true).
func TestAccCollectionResource_explicitSortFalse(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-sort-false")

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

  # Explicitly disable sort on numeric field
  field {
    name = "count"
    type = "int32"
    sort = false
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "count"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.sort", "false"),
				),
			},
		},
	})
}

// TestAccCollectionResource_explicitSortTrue tests that explicitly setting
// sort=true works for string types (overriding server default of false).
func TestAccCollectionResource_explicitSortTrue(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-sort-true")

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

  # Explicitly enable sort on string field
  field {
    name = "title"
    type = "string"
    sort = true
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "title"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.sort", "true"),
				),
			},
		},
	})
}

// TestAccCollectionResource_allFieldAttributesUnset tests a field with all
// optional attributes unset. This catches any server-side default mismatches.
func TestAccCollectionResource_allFieldAttributesUnset(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-unset")

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

  # Only name and type specified - all other attributes use server defaults
  field {
    name = "description"
    type = "string"
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "2"),
					// Verify all computed attributes have values (not unknown)
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.name", "description"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.type", "string"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.1.facet"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.1.optional"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.1.index"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.1.sort"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.1.infix"),
				),
			},
		},
	})
}

// TestAccCollectionResource_mixedFieldTypes tests a realistic collection with
// various field types and attribute combinations to catch integration issues.
func TestAccCollectionResource_mixedFieldTypes(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-mixed")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name                  = %[1]q
  enable_nested_fields  = true

  field {
    name = "id"
    type = "string"
  }

  # String with facet
  field {
    name  = "category"
    type  = "string"
    facet = true
  }

  # Optional string
  field {
    name     = "description"
    type     = "string"
    optional = true
  }

  # Numeric without sort (uses server default)
  field {
    name = "view_count"
    type = "int64"
  }

  # Numeric with explicit sort=true
  field {
    name = "rating"
    type = "float"
    sort = true
  }

  # Numeric with explicit sort=false
  field {
    name = "sequence"
    type = "int32"
    sort = false
  }

  # Array type
  field {
    name  = "tags"
    type  = "string[]"
    facet = true
  }

  # Optional numeric
  field {
    name     = "price"
    type     = "float"
    optional = true
  }

  # String with infix
  field {
    name  = "title"
    type  = "string"
    infix = true
  }

  # Geopoint
  field {
    name     = "location"
    type     = "geopoint"
    optional = true
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "enable_nested_fields", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "10"),
					// Verify specific fields
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.facet", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.optional", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.3.sort", "true"), // int64 server default
					resource.TestCheckResourceAttr("typesense_collection.test", "field.4.sort", "true"), // explicit
					resource.TestCheckResourceAttr("typesense_collection.test", "field.5.sort", "false"), // explicit false
					resource.TestCheckResourceAttr("typesense_collection.test", "field.8.infix", "true"),
				),
			},
		},
	})
}

// TestAccCollectionResource_objectFields tests object field types which have
// their own set of server-side defaults.
func TestAccCollectionResource_objectFields(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-object")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "typesense_collection" "test" {
  name                  = %[1]q
  enable_nested_fields  = true

  field {
    name = "id"
    type = "string"
  }

  # Object field
  field {
    name     = "metadata"
    type     = "object"
    optional = true
  }

  # Object array field
  field {
    name = "items"
    type = "object[]"
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "3"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.type", "object"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.type", "object[]"),
					// Verify computed attributes are set
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.1.sort"),
					resource.TestCheckResourceAttrSet("typesense_collection.test", "field.2.sort"),
				),
			},
		},
	})
}

// =============================================================================
// GEOPOINT FIELD TESTS
// =============================================================================

// TestAccCollectionResource_geopointField tests geopoint fields which have
// their own server-side defaults (sort=true).
func TestAccCollectionResource_geopointField(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-geo")

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

  # Geopoint without explicit sort - server defaults to sort=true
  field {
    name = "location"
    type = "geopoint"
  }

  # Optional geopoint
  field {
    name     = "headquarters"
    type     = "geopoint"
    optional = true
  }
}
`, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.#", "3"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.type", "geopoint"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.1.sort", "true"),
					resource.TestCheckResourceAttr("typesense_collection.test", "field.2.optional", "true"),
				),
			},
		},
	})
}
