package generator

import (
	"fmt"
	"strconv"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ImportCommand represents a terraform import command
type ImportCommand struct {
	ResourceType string
	ResourceName string
	ImportID     string
}

// GenerateImportBlocks creates HCL import blocks for all resources.
// These use the Terraform 1.5+ import block syntax, allowing
// `terraform apply` to import existing resources declaratively.
func GenerateImportBlocks(commands []ImportCommand) *hclwrite.File {
	f := hclwrite.NewEmptyFile()

	f.Body().AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 4, Bytes: []byte("# Generated Terraform import blocks\n# Run 'terraform apply' after 'terraform init' to import existing resources.\n# Once imported, you can remove this file.\n\n")},
	})

	for i, cmd := range commands {
		block := hclwrite.NewBlock("import", nil)
		block.Body().SetAttributeValue("to", cty.StringVal(fmt.Sprintf("%s.%s", cmd.ResourceType, cmd.ResourceName)))
		block.Body().SetAttributeValue("id", cty.StringVal(cmd.ImportID))

		// The "to" attribute must be a resource reference, not a string.
		// hclwrite.SetAttributeValue wraps it in quotes, so we use raw tokens instead.
		block.Body().RemoveAttribute("to")
		block.Body().SetAttributeRaw("to", hclwrite.TokensForTraversal(hclAbsTraversal(cmd.ResourceType, cmd.ResourceName)))

		f.Body().AppendBlock(block)
		if i < len(commands)-1 {
			f.Body().AppendNewline()
		}
	}

	return f
}

// hclAbsTraversal builds a two-part traversal: resourceType.resourceName
func hclAbsTraversal(resourceType, resourceName string) hcl.Traversal {
	return hcl.Traversal{
		hcl.TraverseRoot{Name: resourceType},
		hcl.TraverseAttr{Name: resourceName},
	}
}

// CollectionImportID returns the import ID for a collection
func CollectionImportID(name string) string {
	return name
}

// SynonymImportID returns the import ID for a synonym
func SynonymImportID(collectionName, synonymID string) string {
	return fmt.Sprintf("%s/%s", collectionName, synonymID)
}

// OverrideImportID returns the import ID for an override
func OverrideImportID(collectionName, overrideID string) string {
	return fmt.Sprintf("%s/%s", collectionName, overrideID)
}

// StopwordsImportID returns the import ID for a stopwords set
func StopwordsImportID(id string) string {
	return id
}

// CollectionAliasImportID returns the import ID for a collection alias
func CollectionAliasImportID(name string) string {
	return name
}

// PresetImportID returns the import ID for a preset
func PresetImportID(name string) string {
	return name
}

// StemmingDictionaryImportID returns the import ID for a stemming dictionary
func StemmingDictionaryImportID(id string) string {
	return id
}

// ClusterImportID returns the import ID for a cluster
func ClusterImportID(id string) string {
	return id
}

// AnalyticsRuleImportID returns the import ID for an analytics rule
func AnalyticsRuleImportID(name string) string {
	return name
}

// APIKeyImportID returns the import ID for an API key
func APIKeyImportID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// NLSearchModelImportID returns the import ID for an NL search model
func NLSearchModelImportID(id string) string {
	return id
}

// ConversationModelImportID returns the import ID for a conversation model
func ConversationModelImportID(id string) string {
	return id
}
