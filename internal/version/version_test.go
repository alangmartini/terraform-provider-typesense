package version

import (
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
	if V29_0.Major != 29 || V29_0.Minor != 0 {
		t.Errorf("V29_0 = %v, want 29.0", V29_0)
	}
	if V30_0.Major != 30 || V30_0.Minor != 0 {
		t.Errorf("V30_0 = %v, want 30.0", V30_0)
	}
}
