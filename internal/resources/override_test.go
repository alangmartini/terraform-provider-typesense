package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOverrideResource_includes(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")
	overrideName := acctest.RandomWithPrefix("test-override")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOverrideResourceConfig_includes(rName, overrideName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_override.test", "collection", rName),
					resource.TestCheckResourceAttr("typesense_override.test", "name", overrideName),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.query", "apple"),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.match", "exact"),
					resource.TestCheckResourceAttr("typesense_override.test", "includes.#", "2"),
					resource.TestCheckResourceAttr("typesense_override.test", "includes.0.id", "100"),
					resource.TestCheckResourceAttr("typesense_override.test", "includes.0.position", "1"),
					resource.TestCheckResourceAttr("typesense_override.test", "includes.1.id", "200"),
					resource.TestCheckResourceAttr("typesense_override.test", "includes.1.position", "2"),
					resource.TestCheckResourceAttrSet("typesense_override.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_override.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/%s", rName, overrideName),
			},
		},
	})
}

func TestAccOverrideResource_excludes(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")
	overrideName := acctest.RandomWithPrefix("test-override")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOverrideResourceConfig_excludes(rName, overrideName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_override.test", "collection", rName),
					resource.TestCheckResourceAttr("typesense_override.test", "name", overrideName),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.query", "phone"),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.match", "contains"),
					resource.TestCheckResourceAttr("typesense_override.test", "excludes.#", "2"),
					resource.TestCheckResourceAttr("typesense_override.test", "excludes.0.id", "300"),
					resource.TestCheckResourceAttr("typesense_override.test", "excludes.1.id", "400"),
					resource.TestCheckResourceAttrSet("typesense_override.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_override.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/%s", rName, overrideName),
			},
		},
	})
}

func TestAccOverrideResource_filterBy(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")
	overrideName := acctest.RandomWithPrefix("test-override")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOverrideResourceConfig_filterBy(rName, overrideName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_override.test", "collection", rName),
					resource.TestCheckResourceAttr("typesense_override.test", "name", overrideName),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.query", "laptop"),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.match", "exact"),
					resource.TestCheckResourceAttr("typesense_override.test", "filter_by", "category:electronics"),
					resource.TestCheckResourceAttrSet("typesense_override.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_override.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/%s", rName, overrideName),
			},
		},
	})
}

func TestAccOverrideResource_replaceQuery(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")
	overrideName := acctest.RandomWithPrefix("test-override")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOverrideResourceConfig_replaceQuery(rName, overrideName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_override.test", "collection", rName),
					resource.TestCheckResourceAttr("typesense_override.test", "name", overrideName),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.query", "iphone"),
					resource.TestCheckResourceAttr("typesense_override.test", "rule.match", "exact"),
					resource.TestCheckResourceAttr("typesense_override.test", "replace_query", "apple iphone smartphone"),
					resource.TestCheckResourceAttrSet("typesense_override.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_override.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/%s", rName, overrideName),
			},
		},
	})
}

func testAccOverrideResourceConfig_includes(collectionName, overrideName string) string {
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

resource "typesense_override" "test" {
  collection = typesense_collection.test.name
  name       = %[2]q

  rule {
    query = "apple"
    match = "exact"
  }

  includes {
    id       = "100"
    position = 1
  }

  includes {
    id       = "200"
    position = 2
  }
}
`, collectionName, overrideName)
}

func testAccOverrideResourceConfig_excludes(collectionName, overrideName string) string {
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

resource "typesense_override" "test" {
  collection = typesense_collection.test.name
  name       = %[2]q

  rule {
    query = "phone"
    match = "contains"
  }

  excludes {
    id = "300"
  }

  excludes {
    id = "400"
  }
}
`, collectionName, overrideName)
}

func testAccOverrideResourceConfig_filterBy(collectionName, overrideName string) string {
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
    name = "category"
    type = "string"
  }
}

resource "typesense_override" "test" {
  collection = typesense_collection.test.name
  name       = %[2]q

  rule {
    query = "laptop"
    match = "exact"
  }

  filter_by = "category:electronics"
}
`, collectionName, overrideName)
}

func testAccOverrideResourceConfig_replaceQuery(collectionName, overrideName string) string {
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

resource "typesense_override" "test" {
  collection = typesense_collection.test.name
  name       = %[2]q

  rule {
    query = "iphone"
    match = "exact"
  }

  replace_query = "apple iphone smartphone"
}
`, collectionName, overrideName)
}
