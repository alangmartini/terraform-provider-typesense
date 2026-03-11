package resources

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestClusterSchemaMarksCreationTimeOnlyFieldsRequiresReplace(t *testing.T) {
	cluster := &ClusterResource{}
	var resp resource.SchemaResponse

	cluster.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	searchDeliveryNetworkAttr, ok := resp.Schema.Attributes["search_delivery_network"].(schema.StringAttribute)
	if !ok {
		t.Fatal("search_delivery_network should be a string attribute")
	}
	if !hasStringPlanModifier(searchDeliveryNetworkAttr.PlanModifiers, stringplanmodifier.RequiresReplace()) {
		t.Fatal("search_delivery_network should require replacement")
	}

	regionsAttr, ok := resp.Schema.Attributes["regions"].(schema.ListAttribute)
	if !ok {
		t.Fatal("regions should be a list attribute")
	}
	if !hasListPlanModifier(regionsAttr.PlanModifiers, listplanmodifier.RequiresReplace()) {
		t.Fatal("regions should require replacement")
	}
}

func TestClusterSchemaKeepsMutableFieldsInPlace(t *testing.T) {
	cluster := &ClusterResource{}
	var resp resource.SchemaResponse

	cluster.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	nameAttr, ok := resp.Schema.Attributes["name"].(schema.StringAttribute)
	if !ok {
		t.Fatal("name should be a string attribute")
	}
	if hasStringPlanModifier(nameAttr.PlanModifiers, stringplanmodifier.RequiresReplace()) {
		t.Fatal("name should not require replacement")
	}

	memoryAttr, ok := resp.Schema.Attributes["memory"].(schema.StringAttribute)
	if !ok {
		t.Fatal("memory should be a string attribute")
	}
	if hasStringPlanModifier(memoryAttr.PlanModifiers, stringplanmodifier.RequiresReplace()) {
		t.Fatal("memory should not require replacement")
	}

	autoUpgradeCapacityAttr, ok := resp.Schema.Attributes["auto_upgrade_capacity"].(schema.BoolAttribute)
	if !ok {
		t.Fatal("auto_upgrade_capacity should be a bool attribute")
	}
	if len(autoUpgradeCapacityAttr.PlanModifiers) != 0 {
		t.Fatal("auto_upgrade_capacity should not require replacement")
	}
}

func TestClusterSchemaRequiresReplacementWhenDisablingHighAvailability(t *testing.T) {
	cluster := &ClusterResource{}
	var schemaResp resource.SchemaResponse

	cluster.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	highAvailabilityAttr, ok := schemaResp.Schema.Attributes["high_availability"].(schema.StringAttribute)
	if !ok {
		t.Fatal("high_availability should be a string attribute")
	}
	if len(highAvailabilityAttr.PlanModifiers) == 0 {
		t.Fatal("high_availability should have a plan modifier")
	}

	testSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"high_availability": schema.StringAttribute{},
		},
	}

	makePlan := func(value types.String) tfsdk.Plan {
		tfValue, err := value.ToTerraformValue(context.Background())
		if err != nil {
			t.Fatalf("plan ToTerraformValue error: %s", err)
		}

		return tfsdk.Plan{
			Schema: testSchema,
			Raw: tftypes.NewValue(
				testSchema.Type().TerraformType(context.Background()),
				map[string]tftypes.Value{
					"high_availability": tfValue,
				},
			),
		}
	}

	makeState := func(value types.String) tfsdk.State {
		tfValue, err := value.ToTerraformValue(context.Background())
		if err != nil {
			t.Fatalf("state ToTerraformValue error: %s", err)
		}

		return tfsdk.State{
			Schema: testSchema,
			Raw: tftypes.NewValue(
				testSchema.Type().TerraformType(context.Background()),
				map[string]tftypes.Value{
					"high_availability": tfValue,
				},
			),
		}
	}

	resp := &planmodifier.StringResponse{
		PlanValue: types.StringValue("no"),
	}

	highAvailabilityAttr.PlanModifiers[0].PlanModifyString(context.Background(), planmodifier.StringRequest{
		Plan:       makePlan(types.StringValue("no")),
		PlanValue:  types.StringValue("no"),
		State:      makeState(types.StringValue("yes")),
		StateValue: types.StringValue("yes"),
	}, resp)

	if !resp.RequiresReplace {
		t.Fatal("disabling high_availability should require replacement")
	}
}

func TestClusterPlanWarnings(t *testing.T) {
	regionsState, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"us-east-1"})
	if diags.HasError() {
		t.Fatalf("state regions diagnostics: %v", diags)
	}

	regionsPlan, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"us-east-1", "us-west-2"})
	if diags.HasError() {
		t.Fatalf("plan regions diagnostics: %v", diags)
	}

	warnings := clusterPlanWarnings(
		ClusterResourceModel{
			SearchDeliveryNetwork: types.StringValue("off"),
			Regions:               regionsState,
			HighAvailability:      types.StringValue("yes"),
		},
		ClusterResourceModel{
			SearchDeliveryNetwork: types.StringValue("on"),
			Regions:               regionsPlan,
			HighAvailability:      types.StringValue("no"),
		},
	)

	want := []clusterPlanWarning{
		{
			Attribute: "search_delivery_network",
			Summary:   "Cluster Replacement Required",
			Detail:    "Typesense Cloud only accepts `search_delivery_network` when a cluster is created. Terraform will replace this cluster to apply the new value.",
		},
		{
			Attribute: "regions",
			Summary:   "Cluster Replacement Required",
			Detail:    "Typesense Cloud only accepts `regions` when a cluster is created. Terraform will replace this cluster to apply the new region set.",
		},
		{
			Attribute: "high_availability",
			Summary:   "Cluster Replacement Required",
			Detail:    "Typesense Cloud does not allow disabling high availability on an existing cluster. Terraform will replace this cluster to apply this change.",
		},
	}

	if diff := cmp.Diff(want, warnings); diff != "" {
		t.Fatalf("unexpected warnings diff: %s", diff)
	}
}

func hasStringPlanModifier(modifiers []planmodifier.String, want planmodifier.String) bool {
	wantType := reflect.TypeOf(want)
	for _, modifier := range modifiers {
		if reflect.TypeOf(modifier) == wantType {
			return true
		}
	}

	return false
}

func hasListPlanModifier(modifiers []planmodifier.List, want planmodifier.List) bool {
	wantType := reflect.TypeOf(want)
	for _, modifier := range modifiers {
		if reflect.TypeOf(modifier) == wantType {
			return true
		}
	}

	return false
}
