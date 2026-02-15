package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStemmingDictionaryResource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-stemdict")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStemmingDictionaryConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "dictionary_id", rName),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.#", "2"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.0.word", "running"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.0.stem", "run"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.1.word", "jumping"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.1.stem", "jump"),
					resource.TestCheckResourceAttrSet("typesense_stemming_dictionary.test", "id"),
				),
			},
			{
				ResourceName:      "typesense_stemming_dictionary.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccStemmingDictionaryResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-stemdict")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStemmingDictionaryConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "dictionary_id", rName),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.#", "2"),
				),
			},
			{
				Config: testAccStemmingDictionaryConfig_updated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "dictionary_id", rName),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.#", "3"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.0.word", "running"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.0.stem", "run"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.1.word", "jumping"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.1.stem", "jump"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.2.word", "better"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.2.stem", "good"),
				),
			},
		},
	})
}

func TestAccStemmingDictionaryResource_multipleWords(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-stemdict")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStemmingDictionaryConfig_multipleWords(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "dictionary_id", rName),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.#", "4"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.0.word", "guitars"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.0.stem", "guitar"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.1.word", "drumming"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.1.stem", "drum"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.2.word", "singing"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.2.stem", "sing"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.3.word", "recordings"),
					resource.TestCheckResourceAttr("typesense_stemming_dictionary.test", "words.3.stem", "recording"),
				),
			},
		},
	})
}

func testAccStemmingDictionaryConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "typesense_stemming_dictionary" "test" {
  dictionary_id = %[1]q

  words {
    word = "running"
    stem = "run"
  }
  words {
    word = "jumping"
    stem = "jump"
  }
}
`, name)
}

func testAccStemmingDictionaryConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "typesense_stemming_dictionary" "test" {
  dictionary_id = %[1]q

  words {
    word = "running"
    stem = "run"
  }
  words {
    word = "jumping"
    stem = "jump"
  }
  words {
    word = "better"
    stem = "good"
  }
}
`, name)
}

func testAccStemmingDictionaryConfig_multipleWords(name string) string {
	return fmt.Sprintf(`
resource "typesense_stemming_dictionary" "test" {
  dictionary_id = %[1]q

  words {
    word = "guitars"
    stem = "guitar"
  }
  words {
    word = "drumming"
    stem = "drum"
  }
  words {
    word = "singing"
    stem = "sing"
  }
  words {
    word = "recordings"
    stem = "recording"
  }
}
`, name)
}
