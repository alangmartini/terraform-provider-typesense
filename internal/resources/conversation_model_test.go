package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// CONVERSATION MODEL VALIDATION TESTS
// =============================================================================
// These tests verify that Typesense properly validates history collection
// schema when creating a conversation model. The history collection requires
// specific fields with correct types.

// TestAccConversationModelResource_timestampMustBeInteger tests that Typesense
// rejects history collections where the timestamp field is not an integer type.
// This reproduces the error: "`timestamp` field must be an integer"
func TestAccConversationModelResource_timestampMustBeInteger(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-history")
	modelID := acctest.RandomWithPrefix("test-model")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// History collection with timestamp as STRING (wrong type)
				// Typesense requires timestamp to be int32 or int64
				Config: testAccConversationModelConfig_timestampAsString(rName, modelID),
				ExpectError: regexp.MustCompile(
					`timestamp.*field must be an integer`,
				),
			},
		},
	})
}

// testAccConversationModelConfig_timestampAsString creates a configuration
// with a history collection where timestamp is defined as string type.
// This should trigger validation error from Typesense.
func testAccConversationModelConfig_timestampAsString(historyCollection, modelID string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "history" {
  name = %[1]q

  field {
    name = "conversation_id"
    type = "string"
  }

  field {
    name = "role"
    type = "string"
  }

  field {
    name = "message"
    type = "string"
  }

  # WRONG: timestamp as string instead of int64
  # Typesense requires timestamp to be an integer type
  field {
    name = "timestamp"
    type = "string"
  }
}

resource "typesense_conversation_model" "test" {
  id                 = %[2]q
  model_name         = "openai/gpt-4o-mini"
  api_key            = "test-api-key"
  history_collection = typesense_collection.history.name
  system_prompt      = "You are a helpful assistant."
  max_bytes          = 16000

  depends_on = [typesense_collection.history]
}
`, historyCollection, modelID)
}
