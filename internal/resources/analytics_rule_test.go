package resources_test

import (
	"fmt"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAnalyticsRuleResource_popularQueries(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-analytics")
	collectionName := acctest.RandomWithPrefix("test-collection")
	destCollectionName := acctest.RandomWithPrefix("test-queries")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalyticsRuleResourceConfig_popularQueries(rName, collectionName, destCollectionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "type", "popular_queries"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "event_type", "search"),
					resource.TestCheckResourceAttrSet("typesense_analytics_rule.test", "id"),
					resource.TestCheckResourceAttrSet("typesense_analytics_rule.test", "params"),
				),
			},
			{
				ResourceName:            "typesense_analytics_rule.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"params"}, // API returns additional server-side defaults
			},
		},
	})
}

func TestAccAnalyticsRuleResource_nohitsQueries(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-analytics")
	collectionName := acctest.RandomWithPrefix("test-collection")
	destCollectionName := acctest.RandomWithPrefix("test-nohits")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalyticsRuleResourceConfig_nohitsQueries(rName, collectionName, destCollectionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "type", "nohits_queries"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "event_type", "search"),
				),
			},
		},
	})
}

func TestAccAnalyticsRuleResource_counter(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-analytics")
	collectionName := acctest.RandomWithPrefix("test-collection")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalyticsRuleResourceConfig_counter(rName, collectionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "name", rName),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "type", "counter"),
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "event_type", "click"),
				),
			},
		},
	})
}

func TestAccAnalyticsRuleResource_update(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-analytics")
	collectionName := acctest.RandomWithPrefix("test-collection")
	destCollectionName := acctest.RandomWithPrefix("test-queries")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAnalyticsRuleResourceConfig_popularQueries(rName, collectionName, destCollectionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "name", rName),
				),
			},
			{
				Config: testAccAnalyticsRuleResourceConfig_popularQueriesUpdated(rName, collectionName, destCollectionName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_analytics_rule.test", "name", rName),
				),
			},
		},
	})
}

func testAccAnalyticsRuleResourceConfig_popularQueries(ruleName, collectionName, destCollectionName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "source" {
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

resource "typesense_collection" "queries" {
  name = %[3]q

  field {
    name = "q"
    type = "string"
  }

  field {
    name = "count"
    type = "int32"
  }
}

resource "typesense_analytics_rule" "test" {
  name       = %[1]q
  type       = "popular_queries"
  event_type = "search"
  params = jsonencode({
    source = {
      collections = [typesense_collection.source.name]
    }
    destination = {
      collection = typesense_collection.queries.name
    }
    limit = 1000
  })
}
`, ruleName, collectionName, destCollectionName)
}

func testAccAnalyticsRuleResourceConfig_popularQueriesUpdated(ruleName, collectionName, destCollectionName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "source" {
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

resource "typesense_collection" "queries" {
  name = %[3]q

  field {
    name = "q"
    type = "string"
  }

  field {
    name = "count"
    type = "int32"
  }
}

resource "typesense_analytics_rule" "test" {
  name       = %[1]q
  type       = "popular_queries"
  event_type = "search"
  params = jsonencode({
    source = {
      collections = [typesense_collection.source.name]
    }
    destination = {
      collection = typesense_collection.queries.name
    }
    limit        = 500
    expand_query = true
  })
}
`, ruleName, collectionName, destCollectionName)
}

func testAccAnalyticsRuleResourceConfig_nohitsQueries(ruleName, collectionName, destCollectionName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "source" {
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

resource "typesense_collection" "nohits" {
  name = %[3]q

  field {
    name = "q"
    type = "string"
  }

  field {
    name = "count"
    type = "int32"
  }
}

resource "typesense_analytics_rule" "test" {
  name       = %[1]q
  type       = "nohits_queries"
  event_type = "search"
  params = jsonencode({
    source = {
      collections = [typesense_collection.source.name]
    }
    destination = {
      collection = typesense_collection.nohits.name
    }
    limit = 1000
  })
}
`, ruleName, collectionName, destCollectionName)
}

func testAccAnalyticsRuleResourceConfig_counter(ruleName, collectionName string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "source" {
  name = %[2]q

  field {
    name = "id"
    type = "string"
  }

  field {
    name = "title"
    type = "string"
  }

  field {
    name     = "popularity"
    type     = "int32"
    optional = true
  }
}

resource "typesense_analytics_rule" "test" {
  name       = %[1]q
  type       = "counter"
  event_type = "click"
  params = jsonencode({
    source = {
      collections = [typesense_collection.source.name]
      events = [
        {
          type   = "click"
          weight = 1
          name   = "click_event"
        },
        {
          type   = "conversion"
          weight = 5
          name   = "purchase_event"
        }
      ]
    }
    destination = {
      collection    = typesense_collection.source.name
      counter_field = "popularity"
    }
  })
}
`, ruleName, collectionName)
}
