package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAPIKeyResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-api-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "value"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "actions.#", "1"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "actions.0", "documents:search"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "collections.#", "1"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "collections.0", "*"),
				),
			},
			{
				ResourceName:            "typesense_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
		},
	})
}

func TestAccAPIKeyResource_full(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-api-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig_full(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "value"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "description", "Test API key"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "actions.#", "2"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "actions.0", "documents:search"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "actions.1", "documents:get"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "collections.#", "1"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "collections.0", "*"),
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "expires_at"),
				),
			},
			{
				ResourceName:            "typesense_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
		},
	})
}

func testAccAPIKeyResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "typesense_api_key" "test" {
  actions     = ["documents:search"]
  collections = ["*"]
}
`)
}

func testAccAPIKeyResourceConfig_full(name string) string {
	return fmt.Sprintf(`
resource "typesense_api_key" "test" {
  description = "Test API key"
  actions     = ["documents:search", "documents:get"]
  collections = ["*"]
  expires_at  = 9999999999
}
`)
}
