package datasources_test

import (
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerInfoDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "typesense_server_info" "current" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.typesense_server_info.current", "version"),
					resource.TestCheckResourceAttrSet("data.typesense_server_info.current", "state"),
				),
			},
		},
	})
}
