// Package generator provides functionality to generate Terraform configuration from Typesense resources
package generator

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// nonAlphanumericRegex matches characters that are not alphanumeric or underscore
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	// multipleUnderscoresRegex matches multiple consecutive underscores
	multipleUnderscoresRegex = regexp.MustCompile(`_+`)
)

// SanitizeResourceName converts a Typesense resource name to a valid Terraform resource name.
// Terraform resource names must:
// - Start with a letter or underscore
// - Contain only letters, digits, and underscores
func SanitizeResourceName(name string) string {
	if name == "" {
		return "_empty"
	}

	// Replace common separators with underscores
	result := strings.ReplaceAll(name, "-", "_")
	result = strings.ReplaceAll(result, ".", "_")
	result = strings.ReplaceAll(result, " ", "_")

	// Remove any remaining non-alphanumeric characters (except underscore)
	result = nonAlphanumericRegex.ReplaceAllString(result, "")

	// Collapse multiple underscores into one
	result = multipleUnderscoresRegex.ReplaceAllString(result, "_")

	// Trim leading/trailing underscores
	result = strings.Trim(result, "_")

	// If empty after sanitization, use a default
	if result == "" {
		return "_resource"
	}

	// If starts with a digit, prefix with underscore
	if unicode.IsDigit(rune(result[0])) {
		result = "_" + result
	}

	return result
}

// MakeUniqueResourceName generates a unique resource name by appending a suffix if needed
func MakeUniqueResourceName(baseName string, existingNames map[string]bool) string {
	name := SanitizeResourceName(baseName)
	if !existingNames[name] {
		existingNames[name] = true
		return name
	}

	// Try with numeric suffixes
	for i := 2; ; i++ {
		candidate := name + "_" + string(rune('0'+i%10))
		if i >= 10 {
			candidate = name + "_" + strings.Repeat("0", i/10) + string(rune('0'+i%10))
		}
		if !existingNames[candidate] {
			existingNames[candidate] = true
			return candidate
		}
	}
}
