package version

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMajor  int
		wantMinor  int
		wantPatch  int
		wantPre    string
		wantErr    bool
	}{
		{
			name:      "simple two-part version",
			input:     "29.0",
			wantMajor: 29,
			wantMinor: 0,
		},
		{
			name:      "three-part version with patch",
			input:     "30.0.1",
			wantMajor: 30,
			wantMinor: 0,
			wantPatch: 1,
		},
		{
			name:      "version with pre-release",
			input:     "30.0.rc38",
			wantMajor: 30,
			wantMinor: 0,
			wantPre:   "rc38",
		},
		{
			name:      "version with alpha pre-release",
			input:     "31.0.alpha1",
			wantMajor: 31,
			wantMinor: 0,
			wantPre:   "alpha1",
		},
		{
			name:      "version with beta pre-release",
			input:     "31.0.beta2",
			wantMajor: 31,
			wantMinor: 0,
			wantPre:   "beta2",
		},
		{
			name:      "higher major version",
			input:     "100.5",
			wantMajor: 100,
			wantMinor: 5,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format - single number",
			input:   "29",
			wantErr: true,
		},
		{
			name:    "invalid format - text only",
			input:   "latest",
			wantErr: true,
		},
		{
			name:    "invalid format - v prefix",
			input:   "v29.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
				return
			}
			if v.Major != tt.wantMajor {
				t.Errorf("Parse(%q).Major = %d, want %d", tt.input, v.Major, tt.wantMajor)
			}
			if v.Minor != tt.wantMinor {
				t.Errorf("Parse(%q).Minor = %d, want %d", tt.input, v.Minor, tt.wantMinor)
			}
			if v.Patch != tt.wantPatch {
				t.Errorf("Parse(%q).Patch = %d, want %d", tt.input, v.Patch, tt.wantPatch)
			}
			if v.PreRelease != tt.wantPre {
				t.Errorf("Parse(%q).PreRelease = %q, want %q", tt.input, v.PreRelease, tt.wantPre)
			}
			if v.Raw != tt.input {
				t.Errorf("Parse(%q).Raw = %q, want %q", tt.input, v.Raw, tt.input)
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	// Test successful parse
	v := MustParse("29.0")
	if v.Major != 29 {
		t.Errorf("MustParse(29.0).Major = %d, want 29", v.Major)
	}

	// Test panic on invalid input
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParse with invalid input should panic")
		}
	}()
	MustParse("invalid")
}

func TestVersionString(t *testing.T) {
	v := MustParse("30.0.rc38")
	if v.String() != "30.0.rc38" {
		t.Errorf("String() = %q, want %q", v.String(), "30.0.rc38")
	}

	// Test nil version
	var nilV *Version
	if nilV.String() != "" {
		t.Errorf("nil.String() = %q, want empty string", nilV.String())
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"equal versions", "29.0", "29.0", 0},
		{"a < b major", "28.0", "29.0", -1},
		{"a > b major", "30.0", "29.0", 1},
		{"a < b minor", "29.0", "29.1", -1},
		{"a > b minor", "29.5", "29.1", 1},
		{"a < b patch", "29.0.1", "29.0.2", -1},
		{"a > b patch", "29.0.3", "29.0.1", 1},

		// Pre-release comparisons
		{"pre-release < release", "30.0.rc38", "30.0", -1},
		{"release > pre-release", "30.0", "30.0.rc38", 1},
		{"same pre-release", "30.0.rc38", "30.0.rc38", 0},
		{"earlier rc < later rc", "30.0.rc1", "30.0.rc38", -1},
		{"later rc > earlier rc", "30.0.rc38", "30.0.rc1", 1},
		{"alpha < beta", "30.0.alpha1", "30.0.beta1", -1},
		{"beta > alpha", "30.0.beta1", "30.0.alpha1", 1},
		{"rc > beta", "30.0.rc1", "30.0.beta1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			got := a.Compare(b)
			if got != tt.want {
				t.Errorf("%s.Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVersionCompareNil(t *testing.T) {
	v := MustParse("29.0")

	if v.Compare(nil) != 1 {
		t.Error("version.Compare(nil) should return 1")
	}

	var nilV *Version
	if nilV.Compare(v) != -1 {
		t.Error("nil.Compare(version) should return -1")
	}

	if nilV.Compare(nil) != 0 {
		t.Error("nil.Compare(nil) should return 0")
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal", "29.0", "29.0", true},
		{"greater major", "30.0", "29.0", true},
		{"lesser major", "28.0", "29.0", false},
		{"greater minor", "29.5", "29.0", true},
		{"lesser minor", "29.0", "29.5", false},
		{"pre-release vs release", "30.0.rc38", "30.0", false},
		{"release vs pre-release", "30.0", "30.0.rc38", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			if got := a.AtLeast(b); got != tt.want {
				t.Errorf("%s.AtLeast(%s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVersionLessThan(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal", "29.0", "29.0", false},
		{"lesser major", "28.0", "29.0", true},
		{"greater major", "30.0", "29.0", false},
		{"pre-release vs release", "30.0.rc38", "30.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := MustParse(tt.a)
			b := MustParse(tt.b)
			if got := a.LessThan(b); got != tt.want {
				t.Errorf("%s.LessThan(%s) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDefaultFeatureChecker(t *testing.T) {
	tests := []struct {
		name    string
		version string
		feature Feature
		want    bool
	}{
		// v29 features
		{"v29 supports per-collection synonyms", "29.0", FeaturePerCollectionSynonyms, true},
		{"v29 supports per-collection overrides", "29.0", FeaturePerCollectionOverrides, true},
		{"v29 does not support synonym sets", "29.0", FeatureSynonymSets, false},
		{"v29 does not support curation sets", "29.0", FeatureCurationSets, false},

		// v30 features
		{"v30 supports synonym sets", "30.0", FeatureSynonymSets, true},
		{"v30 supports curation sets", "30.0", FeatureCurationSets, true},
		{"v30 does not support per-collection synonyms", "30.0", FeaturePerCollectionSynonyms, false},
		{"v30 does not support per-collection overrides", "30.0", FeaturePerCollectionOverrides, false},

		// v30 RC (pre-release)
		{"v30.rc38 supports synonym sets", "30.0.rc38", FeatureSynonymSets, false}, // Pre-release is < 30.0
		{"v30.rc38 supports per-collection synonyms", "30.0.rc38", FeaturePerCollectionSynonyms, true},

		// v28 (older)
		{"v28 supports per-collection synonyms", "28.0", FeaturePerCollectionSynonyms, true},
		{"v28 does not support synonym sets", "28.0", FeatureSynonymSets, false},

		// Future version (v31)
		{"v31 supports synonym sets", "31.0", FeatureSynonymSets, true},
		{"v31 does not support per-collection synonyms", "31.0", FeaturePerCollectionSynonyms, false},

		// Conversation models (v26+)
		{"v25 does not support conversation models", "25.0", FeatureConversationModels, false},
		{"v26 supports conversation models", "26.0", FeatureConversationModels, true},
		{"v30 supports conversation models", "30.0", FeatureConversationModels, true},

		// Presets (v27+)
		{"v26 does not support presets", "26.0", FeaturePresets, false},
		{"v27 supports presets", "27.0", FeaturePresets, true},
		{"v30 supports presets", "30.0", FeaturePresets, true},

		// Stopwords (v27+)
		{"v26 does not support stopwords", "26.0", FeatureStopwords, false},
		{"v27 supports stopwords", "27.0", FeatureStopwords, true},
		{"v29 supports stopwords", "29.0", FeatureStopwords, true},

		// Analytics rules (v28+)
		{"v27 does not support analytics rules", "27.0", FeatureAnalyticsRules, false},
		{"v28 supports analytics rules", "28.0", FeatureAnalyticsRules, true},
		{"v30 supports analytics rules", "30.0", FeatureAnalyticsRules, true},

		// NL search models (v29+)
		{"v28 does not support NL search models", "28.0", FeatureNLSearchModels, false},
		{"v29 supports NL search models", "29.0", FeatureNLSearchModels, true},
		{"v30 supports NL search models", "30.0", FeatureNLSearchModels, true},

		// Stemming dictionaries (v29+)
		{"v28 does not support stemming dictionaries", "28.0", FeatureStemmingDictionaries, false},
		{"v29 supports stemming dictionaries", "29.0", FeatureStemmingDictionaries, true},
		{"v30 supports stemming dictionaries", "30.0", FeatureStemmingDictionaries, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := MustParse(tt.version)
			checker := NewFeatureChecker(v)
			if got := checker.SupportsFeature(tt.feature); got != tt.want {
				t.Errorf("FeatureChecker(%s).SupportsFeature(%s) = %v, want %v",
					tt.version, tt.feature, got, tt.want)
			}
		})
	}
}

func TestDefaultFeatureCheckerNilVersion(t *testing.T) {
	checker := NewFeatureChecker(nil)

	// With nil version, all features should return false
	features := []Feature{
		FeatureSynonymSets,
		FeatureCurationSets,
		FeaturePerCollectionSynonyms,
		FeaturePerCollectionOverrides,
		FeatureConversationModels,
		FeaturePresets,
		FeatureStopwords,
		FeatureAnalyticsRules,
		FeatureNLSearchModels,
		FeatureStemmingDictionaries,
	}

	for _, f := range features {
		if checker.SupportsFeature(f) {
			t.Errorf("FeatureChecker(nil).SupportsFeature(%s) = true, want false", f)
		}
	}

	if checker.GetVersion() != nil {
		t.Error("FeatureChecker(nil).GetVersion() should return nil")
	}
}

func TestDefaultFeatureCheckerGetVersion(t *testing.T) {
	v := MustParse("29.0")
	checker := NewFeatureChecker(v)

	got := checker.GetVersion()
	if got != v {
		t.Errorf("GetVersion() returned different version")
	}
}

func TestFallbackFeatureChecker(t *testing.T) {
	checker := NewFallbackFeatureChecker()

	// All features should return false
	features := []Feature{
		FeatureSynonymSets,
		FeatureCurationSets,
		FeaturePerCollectionSynonyms,
		FeaturePerCollectionOverrides,
		FeatureConversationModels,
		FeaturePresets,
		FeatureStopwords,
		FeatureAnalyticsRules,
		FeatureNLSearchModels,
		FeatureStemmingDictionaries,
	}

	for _, f := range features {
		if checker.SupportsFeature(f) {
			t.Errorf("FallbackFeatureChecker.SupportsFeature(%s) = true, want false", f)
		}
	}

	if checker.GetVersion() != nil {
		t.Error("FallbackFeatureChecker.GetVersion() should return nil")
	}
}

func TestWellKnownVersions(t *testing.T) {
	// Verify the well-known versions are correctly initialized
	if V26_0.Major != 26 || V26_0.Minor != 0 {
		t.Errorf("V26_0 = %v, want 26.0", V26_0)
	}
	if V27_0.Major != 27 || V27_0.Minor != 0 {
		t.Errorf("V27_0 = %v, want 27.0", V27_0)
	}
	if V28_0.Major != 28 || V28_0.Minor != 0 {
		t.Errorf("V28_0 = %v, want 28.0", V28_0)
	}
	if V29_0.Major != 29 || V29_0.Minor != 0 {
		t.Errorf("V29_0 = %v, want 29.0", V29_0)
	}
	if V30_0.Major != 30 || V30_0.Minor != 0 {
		t.Errorf("V30_0 = %v, want 30.0", V30_0)
	}
}

func TestCheckVersionRequirement(t *testing.T) {
	t.Run("returns error when version is too old", func(t *testing.T) {
		checker := NewFeatureChecker(MustParse("26.0"))
		diags := CheckVersionRequirement(checker, FeaturePresets, "typesense_preset")
		if !diags.HasError() {
			t.Fatal("expected error diagnostic, got none")
		}
		errMsg := diags[0].Detail()
		if !strings.Contains(errMsg, "v27.0+") {
			t.Errorf("error should mention required version v27.0+, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "v26.0") {
			t.Errorf("error should mention current server version v26.0, got: %s", errMsg)
		}
		if !strings.Contains(errMsg, "typesense_preset") {
			t.Errorf("error should mention resource name, got: %s", errMsg)
		}
	})

	t.Run("returns nil when version meets requirement", func(t *testing.T) {
		checker := NewFeatureChecker(MustParse("27.0"))
		diags := CheckVersionRequirement(checker, FeaturePresets, "typesense_preset")
		if diags.HasError() {
			t.Errorf("expected no error, got: %v", diags)
		}
	})

	t.Run("returns nil when version exceeds requirement", func(t *testing.T) {
		checker := NewFeatureChecker(MustParse("30.0"))
		diags := CheckVersionRequirement(checker, FeaturePresets, "typesense_preset")
		if diags.HasError() {
			t.Errorf("expected no error, got: %v", diags)
		}
	})

	t.Run("skips check when version is unknown (fallback)", func(t *testing.T) {
		checker := NewFallbackFeatureChecker()
		diags := CheckVersionRequirement(checker, FeaturePresets, "typesense_preset")
		if diags != nil {
			t.Errorf("expected nil diagnostics for fallback checker, got: %v", diags)
		}
	})

	t.Run("skips check when version is nil", func(t *testing.T) {
		checker := NewFeatureChecker(nil)
		diags := CheckVersionRequirement(checker, FeaturePresets, "typesense_preset")
		if diags != nil {
			t.Errorf("expected nil diagnostics for nil version, got: %v", diags)
		}
	})

	t.Run("error message for each feature type", func(t *testing.T) {
		featureTests := []struct {
			feature      Feature
			resource     string
			tooOld       string
			wantVersion  string
		}{
			{FeatureConversationModels, "typesense_conversation_model", "25.0", "v26.0+"},
			{FeaturePresets, "typesense_preset", "26.0", "v27.0+"},
			{FeatureStopwords, "typesense_stopwords_set", "26.0", "v27.0+"},
			{FeatureAnalyticsRules, "typesense_analytics_rule", "27.0", "v28.0+"},
			{FeatureNLSearchModels, "typesense_nl_search_model", "28.0", "v29.0+"},
			{FeatureStemmingDictionaries, "typesense_stemming_dictionary", "28.0", "v29.0+"},
		}

		for _, tt := range featureTests {
			t.Run(string(tt.feature), func(t *testing.T) {
				checker := NewFeatureChecker(MustParse(tt.tooOld))
				diags := CheckVersionRequirement(checker, tt.feature, tt.resource)
				if !diags.HasError() {
					t.Fatal("expected error diagnostic, got none")
				}
				errMsg := diags[0].Detail()
				if !strings.Contains(errMsg, tt.wantVersion) {
					t.Errorf("error should mention %s, got: %s", tt.wantVersion, errMsg)
				}
				if !strings.Contains(errMsg, tt.resource) {
					t.Errorf("error should mention %s, got: %s", tt.resource, errMsg)
				}
			})
		}
	})
}

func TestFeatureMinVersionString(t *testing.T) {
	tests := []struct {
		feature Feature
		want    string
	}{
		{FeatureConversationModels, "v26.0+"},
		{FeaturePresets, "v27.0+"},
		{FeatureStopwords, "v27.0+"},
		{FeatureAnalyticsRules, "v28.0+"},
		{FeatureNLSearchModels, "v29.0+"},
		{FeatureStemmingDictionaries, "v29.0+"},
		{FeatureSynonymSets, "v30.0+"},
		{FeatureCurationSets, "v30.0+"},
		{FeaturePerCollectionSynonyms, "unknown version"}, // nil min version
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			got := featureMinVersionString(tt.feature)
			if got != tt.want {
				t.Errorf("featureMinVersionString(%s) = %q, want %q", tt.feature, got, tt.want)
			}
		})
	}
}
