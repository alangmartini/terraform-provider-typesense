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

// TestAccConversationModelResource_timestampInt32 tests that Typesense accepts
// int32 type for the timestamp field in history collections.
func TestAccConversationModelResource_timestampInt32(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-history")
	modelID := acctest.RandomWithPrefix("test-model")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// History collection with timestamp as int32 (should work)
				Config: testAccConversationModelConfig_timestampAsInt32(rName, modelID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("typesense_conversation_model.test", "id", modelID),
					resource.TestCheckResourceAttr("typesense_conversation_model.test", "model_name", "openai/gpt-4o-mini"),
				),
			},
		},
	})
}

// TestAccConversationModelResource_timestampInt64 tests that Typesense rejects
// int64 type for the timestamp field - it requires int32 specifically.
func TestAccConversationModelResource_timestampInt64(t *testing.T) {
	rName := acctest.RandomWithPrefix("test-history")
	modelID := acctest.RandomWithPrefix("test-model")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provider.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// History collection with timestamp as int64 (wrong - needs int32)
				Config: testAccConversationModelConfig_timestampAsInt64(rName, modelID),
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
    name = "model_id"
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

  # WRONG: timestamp as string instead of int32
  # Typesense requires timestamp to be int32 specifically
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

// testAccConversationModelConfig_timestampAsInt32 creates a configuration
// with a history collection where timestamp is defined as int32 type.
// This should succeed - int32 is the required integer type for timestamp.
func testAccConversationModelConfig_timestampAsInt32(historyCollection, modelID string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "history" {
  name = %[1]q

  field {
    name = "conversation_id"
    type = "string"
  }

  field {
    name = "model_id"
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

  # CORRECT: timestamp as int32 (required integer type)
  field {
    name = "timestamp"
    type = "int32"
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

// testAccConversationModelConfig_timestampAsInt64 creates a configuration
// with a history collection where timestamp is defined as int64 type.
// NOTE: Typesense requires int32 specifically for timestamp, not int64.
// This test expects an error.
func testAccConversationModelConfig_timestampAsInt64(historyCollection, modelID string) string {
	return fmt.Sprintf(`
resource "typesense_collection" "history" {
  name = %[1]q

  field {
    name = "conversation_id"
    type = "string"
  }

  field {
    name = "model_id"
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

  # WRONG: timestamp as int64 - Typesense requires int32
  field {
    name = "timestamp"
    type = "int64"
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
