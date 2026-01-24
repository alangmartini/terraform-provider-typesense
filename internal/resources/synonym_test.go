package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSynonymResource_multiWay(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")
	synonymName := acctest.RandomWithPrefix("test-synonym")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymResourceConfig_multiWay(rName, synonymName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_synonym.test", "collection", rName),
					resource.TestCheckResourceAttr("typesense_synonym.test", "name", synonymName),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.#", "3"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.0", "blazer"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.1", "coat"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.2", "jacket"),
					resource.TestCheckResourceAttrSet("typesense_synonym.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_synonym.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/%s", rName, synonymName),
			},
		},
	})
}

func TestAccSynonymResource_oneWay(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")
	synonymName := acctest.RandomWithPrefix("test-synonym")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymResourceConfig_oneWay(rName, synonymName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_synonym.test", "collection", rName),
					resource.TestCheckResourceAttr("typesense_synonym.test", "name", synonymName),
					resource.TestCheckResourceAttr("typesense_synonym.test", "root", "pants"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.#", "2"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.0", "trousers"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.1", "jeans"),
					resource.TestCheckResourceAttrSet("typesense_synonym.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_synonym.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/%s", rName, synonymName),
			},
		},
	})
}

func TestAccSynonymResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-collection")
	synonymName := acctest.RandomWithPrefix("test-synonym")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSynonymResourceConfig_multiWay(rName, synonymName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.#", "3"),
				),
			},
			{
				Config: testAccSynonymResourceConfig_updated(rName, synonymName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.#", "4"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.0", "blazer"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.1", "coat"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.2", "jacket"),
					resource.TestCheckResourceAttr("typesense_synonym.test", "synonyms.3", "parka"),
				),
			},
		},
	})
}

func testAccSynonymResourceConfig_multiWay(collectionName, synonymName string) string {
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

resource "typesense_synonym" "test" {
  collection = typesense_collection.test.name
  name       = %[2]q
  synonyms   = ["blazer", "coat", "jacket"]
}
`, collectionName, synonymName)
}

func testAccSynonymResourceConfig_oneWay(collectionName, synonymName string) string {
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

resource "typesense_synonym" "test" {
  collection = typesense_collection.test.name
  name       = %[2]q
  root       = "pants"
  synonyms   = ["trousers", "jeans"]
}
`, collectionName, synonymName)
}

func testAccSynonymResourceConfig_updated(collectionName, synonymName string) string {
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

resource "typesense_synonym" "test" {
  collection = typesense_collection.test.name
  name       = %[2]q
  synonyms   = ["blazer", "coat", "jacket", "parka"]
}
`, collectionName, synonymName)
}
