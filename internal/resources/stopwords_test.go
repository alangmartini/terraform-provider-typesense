package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStopwordsSetResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-stopwords")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStopwordsSetResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.#", "3"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.0", "the"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.1", "a"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.2", "an"),
					resource.TestCheckResourceAttrSet("typesense_stopwords_set.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_stopwords_set.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccStopwordsSetResource_withLocale(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-stopwords")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStopwordsSetResourceConfig_withLocale(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "locale", "en"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.#", "4"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.0", "the"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.1", "is"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.2", "at"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.3", "which"),
				),
			},
			{
				ResourceName:      "typesense_stopwords_set.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccStopwordsSetResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-stopwords")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStopwordsSetResourceConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.#", "3"),
				),
			},
			{
				Config: testAccStopwordsSetResourceConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.#", "5"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.0", "the"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.1", "a"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.2", "an"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.3", "of"),
					resource.TestCheckResourceAttr("typesense_stopwords_set.test", "stopwords.4", "in"),
				),
			},
		},
	})
}

func testAccStopwordsSetResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "typesense_stopwords_set" "test" {
  name = %[1]q
  stopwords = ["the", "a", "an"]
}
`, name)
}

func testAccStopwordsSetResourceConfig_withLocale(name string) string {
	return fmt.Sprintf(`
resource "typesense_stopwords_set" "test" {
  name      = %[1]q
  locale    = "en"
  stopwords = ["the", "is", "at", "which"]
}
`, name)
}

func testAccStopwordsSetResourceConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "typesense_stopwords_set" "test" {
  name = %[1]q
  stopwords = ["the", "a", "an", "of", "in"]
}
`, name)
}
