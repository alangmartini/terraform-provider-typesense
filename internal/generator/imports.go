package generator

import (
	"fmt"
	"strconv"
	"strings"
)

// ImportCommand represents a terraform import command
type ImportCommand struct {
	ResourceType string
	ResourceName string
	ImportID     string
}

// GenerateImportScript creates a shell script containing all terraform import commands
func GenerateImportScript(commands []ImportCommand) string {
	var sb strings.Builder

	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("# Generated Terraform import commands\n")
	sb.WriteString("# Run this script after 'terraform init' to import existing resources\n")
	sb.WriteString("\n")
	sb.WriteString("set -e\n")
	sb.WriteString("\n")

	for _, cmd := range commands {
		sb.WriteString(fmt.Sprintf("terraform import %s.%s %q\n",
			cmd.ResourceType, cmd.ResourceName, cmd.ImportID))
	}

	sb.WriteString("\n")
	sb.WriteString("echo \"Import complete!\"\n")

	return sb.String()
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
