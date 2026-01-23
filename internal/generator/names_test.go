package generator

import "testing"

func TestSanitizeResourceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "products",
			expected: "products",
		},
		{
			name:     "name with hyphens",
			input:    "my-collection",
			expected: "my_collection",
		},
		{
			name:     "name with dots",
			input:    "api.v1.products",
			expected: "api_v1_products",
		},
		{
			name:     "name with spaces",
			input:    "my collection",
			expected: "my_collection",
		},
		{
			name:     "name starting with digit",
			input:    "123products",
			expected: "_123products",
		},
		{
			name:     "name with special characters",
			input:    "products@#$%test",
			expected: "productstest",
		},
		{
			name:     "name with multiple special chars",
			input:    "my---collection...name",
			expected: "my_collection_name",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "_empty",
		},
		{
			name:     "only special characters",
			input:    "@#$%^&",
			expected: "_resource",
		},
		{
			name:     "name with underscores",
			input:    "my_collection_name",
			expected: "my_collection_name",
		},
		{
			name:     "mixed case",
			input:    "MyCollection",
			expected: "MyCollection",
		},
		{
			name:     "leading and trailing special chars",
			input:    "---products---",
			expected: "products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeResourceName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeResourceName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMakeUniqueResourceName(t *testing.T) {
	existing := make(map[string]bool)

	// First name should be used as-is
	name1 := MakeUniqueResourceName("products", existing)
	if name1 != "products" {
		t.Errorf("First name should be 'products', got %q", name1)
	}

	// Second use should get a suffix
	name2 := MakeUniqueResourceName("products", existing)
	if name2 == "products" {
		t.Errorf("Second name should not be 'products', got %q", name2)
	}
	if name2 == name1 {
		t.Errorf("Second name should be different from first, both are %q", name1)
	}

	// Third use should get another suffix
	name3 := MakeUniqueResourceName("products", existing)
	if name3 == name1 || name3 == name2 {
		t.Errorf("Third name should be unique, got %q (same as previous)", name3)
	}
}
