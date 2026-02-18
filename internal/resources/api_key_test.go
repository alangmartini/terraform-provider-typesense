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
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "value_prefix"),
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
				ImportStateVerifyIgnore: []string{"value", "autodelete"},
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
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "value_prefix"),
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
				ImportStateVerifyIgnore: []string{"value", "autodelete"},
			},
		},
	})
}

func TestAccAPIKeyResource_userProvidedValue(t *testing.T) {
	// Bug: user-provided key values weren't supported, making multi-env deployments painful
	rName := acctest.RandomWithPrefix("test-api-key")
	keyValue := acctest.RandString(32)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig_userProvidedValue(rName, keyValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "id"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "value", keyValue),
					resource.TestCheckResourceAttr("typesense_api_key.test", "value_prefix", keyValue[:4]),
					resource.TestCheckResourceAttr("typesense_api_key.test", "description", "User-provided value test"),
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
				ImportStateVerifyIgnore: []string{"value", "autodelete"},
			},
		},
	})
}

func TestAccAPIKeyResource_autodelete(t *testing.T) {
	// Verify autodelete flag is sent correctly with expires_at
	rName := acctest.RandomWithPrefix("test-api-key")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAPIKeyResourceConfig_autodelete(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "value"),
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "value_prefix"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "description", "Autodelete test key"),
					resource.TestCheckResourceAttr("typesense_api_key.test", "autodelete", "true"),
					resource.TestCheckResourceAttrSet("typesense_api_key.test", "expires_at"),
				),
			},
			{
				ResourceName:            "typesense_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value", "autodelete"},
			},
		},
	})
}

func testAccAPIKeyResourceConfig_basic(_ string) string {
	return `
resource "typesense_api_key" "test" {
  actions     = ["documents:search"]
  collections = ["*"]
}
`
}

func testAccAPIKeyResourceConfig_full(_ string) string {
	return `
resource "typesense_api_key" "test" {
  description = "Test API key"
  actions     = ["documents:search", "documents:get"]
  collections = ["*"]
  expires_at  = 9999999999
}
`
}

func testAccAPIKeyResourceConfig_userProvidedValue(_ string, value string) string {
	return fmt.Sprintf(`
resource "typesense_api_key" "test" {
  description = "User-provided value test"
  value       = %q
  actions     = ["documents:search"]
  collections = ["*"]
}
`, value)
}

func testAccAPIKeyResourceConfig_autodelete(_ string) string {
	return `
resource "typesense_api_key" "test" {
  description = "Autodelete test key"
  actions     = ["documents:search"]
  collections = ["*"]
  expires_at  = 9999999999
  autodelete  = true
}
`
}
