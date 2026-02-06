package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCollectionAliasResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-alias")
	collectionName := acctest.RandomWithPrefix("test-collection")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionAliasResourceConfig_basic(rName, collectionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection_alias.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection_alias.test", "collection_name", collectionName),
					resource.TestCheckResourceAttrSet("typesense_collection_alias.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_collection_alias.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccCollectionAliasResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-alias")
	collectionName1 := acctest.RandomWithPrefix("test-collection-1")
	collectionName2 := acctest.RandomWithPrefix("test-collection-2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCollectionAliasResourceConfig_basic(rName, collectionName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection_alias.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection_alias.test", "collection_name", collectionName1),
				),
			},
			{
				Config: testAccCollectionAliasResourceConfig_twoCollections(rName, collectionName1, collectionName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_collection_alias.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_collection_alias.test", "collection_name", collectionName2),
				),
			},
		},
	})
}

func testAccCollectionAliasResourceConfig_basic(aliasName, collectionName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[2]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }
}

resource "typesense_collection_alias" "test" {
  name            = %[1]q
  collection_name = typesense_collection.test.name
}
`, aliasName, collectionName)
}

func testAccCollectionAliasResourceConfig_twoCollections(aliasName, collectionName1, collectionName2 string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "test" {
  name = %[2]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }
}

resource "typesense_collection" "test2" {
  name = %[3]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }
}

resource "typesense_collection_alias" "test" {
  name            = %[1]q
  collection_name = typesense_collection.test2.name
}
`, aliasName, collectionName1, collectionName2)
}
