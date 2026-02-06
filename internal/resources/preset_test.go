package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPresetResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-preset")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPresetResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_preset.test", "name", rName),
					resource.TestCheckResourceAttrSet("typesense_preset.test", "id"),
					resource.TestCheckResourceAttrSet("typesense_preset.test", "value"),
				),
			},
			{
				ResourceName:      "typesense_preset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPresetResource_withMultipleParams(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-preset")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPresetResourceConfig_multipleParams(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_preset.test", "name", rName),
					resource.TestCheckResourceAttrSet("typesense_preset.test", "value"),
				),
			},
			{
				ResourceName:      "typesense_preset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPresetResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-preset")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPresetResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_preset.test", "name", rName),
				),
			},
			{
				Config: testAccPresetResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_preset.test", "name", rName),
				),
			},
		},
	})
}

func testAccPresetResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "typesense_preset" "test" {
  name  = %[1]q
  value = jsonencode({
    q       = "*"
    sort_by = "popularity:desc"
  })
}
`, name)
}

func testAccPresetResourceConfig_multipleParams(name string) string {
	return fmt.Sprintf(`
resource "typesense_preset" "test" {
  name  = %[1]q
  value = jsonencode({
    q         = "*"
    query_by  = "title,description"
    sort_by   = "popularity:desc"
    filter_by = "status:active"
    per_page  = 20
  })
}
`, name)
}

func testAccPresetResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "typesense_preset" "test" {
  name  = %[1]q
  value = jsonencode({
    q        = "*"
    sort_by  = "created_at:desc"
    per_page = 50
  })
}
`, name)
}
