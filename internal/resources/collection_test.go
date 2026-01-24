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
				ResourceName:      "typesense_collection.test",
				ImportState:       true,
				ImportStateVerify: true,
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
				ResourceName:      "typesense_collection.test",
				ImportState:       true,
				ImportStateVerify: true,
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
