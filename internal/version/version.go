// Package version provides Typesense version parsing and feature detection.
// Different Typesense versions have different API endpoints:
//   - v29 and earlier: /collections/{name}/synonyms/{id} (per-collection synonyms)
//   - v30+: /synonym_sets (system-level synonym sets), /curation_sets
package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Well-known version boundaries for feature detection
var (
	V26_0 = MustParse("26.0")
	V27_0 = MustParse("27.0")
	V28_0 = MustParse("28.0")
	V29_0 = MustParse("29.0")
	V30_0 = MustParse("30.0")
)

// Version represents a parsed Typesense version.
// Typesense uses semver-like versioning: "29.0", "30.0", "30.0.rc38"
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string // e.g., "rc38" for "30.0.rc38"
	Raw        string // Original version string
}

// versionRegex matches Typesense version strings like "29.0", "30.0.1", "30.0.rc38"
var versionRegex = regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+|[a-zA-Z]+\d*))?$`)

// Parse parses a Typesense version string into a Version struct.
// Handles formats like "29.0", "30.0", "30.0.1", "30.0.rc38"
func Parse(s string) (*Version, error) {
	if s == "" {
		return nil, fmt.Errorf("empty version string")
	}

	matches := versionRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %q", s)
	}

	v := &Version{Raw: s}

	var err error
	v.Major, err = strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %w", err)
	}

	v.Minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %w", err)
	}

	// Third component can be a patch number or a pre-release identifier
	if matches[3] != "" {
		// Try parsing as integer (patch version)
		if patch, err := strconv.Atoi(matches[3]); err == nil {
			v.Patch = patch
		} else {
			// It's a pre-release identifier like "rc38"
			v.PreRelease = matches[3]
		}
	}

	return v, nil
}

// MustParse parses a version string and panics if it fails.
// Useful for package-level version constants.
func MustParse(s string) *Version {
	v, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse version %q: %v", s, err))
	}
	return v
}

// String returns the original version string.
func (v *Version) String() string {
	if v == nil {
		return ""
	}
	return v.Raw
}

// Compare returns:
//
//	-1 if v < other
//	 0 if v == other
//	+1 if v > other
//
// Pre-release versions are considered less than the release version.
// For example: 30.0.rc38 < 30.0
func (v *Version) Compare(other *Version) int {
	if v == nil && other == nil {
		return 0
	}
	if v == nil {
		return -1
	}
	if other == nil {
		return 1
	}

	// Compare major
	if v.Major < other.Major {
		return -1
	}
	if v.Major > other.Major {
		return 1
	}

	// Compare minor
	if v.Minor < other.Minor {
		return -1
	}
	if v.Minor > other.Minor {
		return 1
	}

	// Compare patch
	if v.Patch < other.Patch {
		return -1
	}
	if v.Patch > other.Patch {
		return 1
	}

	// Handle pre-release: a pre-release version is less than a release version
	// e.g., 30.0.rc38 < 30.0
	vHasPreRelease := v.PreRelease != ""
	otherHasPreRelease := other.PreRelease != ""

	if vHasPreRelease && !otherHasPreRelease {
		return -1
	}
	if !vHasPreRelease && otherHasPreRelease {
		return 1
	}

	// Both have pre-release or neither do
	if vHasPreRelease && otherHasPreRelease {
		return comparePreRelease(v.PreRelease, other.PreRelease)
	}

	return 0
}

// comparePreRelease compares two pre-release strings.
// Handles formats like "rc38", "alpha1", "beta2"
func comparePreRelease(a, b string) int {
	// Extract numeric suffix if present
	aNum := extractTrailingNumber(a)
	bNum := extractTrailingNumber(b)

	aPrefix := strings.TrimRight(a, "0123456789")
	bPrefix := strings.TrimRight(b, "0123456789")

	// Compare prefixes alphabetically first
	if aPrefix < bPrefix {
		return -1
	}
	if aPrefix > bPrefix {
		return 1
	}

	// Same prefix, compare numbers
	if aNum < bNum {
		return -1
	}
	if aNum > bNum {
		return 1
	}

	return 0
}

// extractTrailingNumber extracts the trailing number from a string like "rc38"
func extractTrailingNumber(s string) int {
	numStr := ""
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] >= '0' && s[i] <= '9' {
			numStr = string(s[i]) + numStr
		} else {
			break
		}
	}
	if numStr == "" {
		return 0
	}
	n, _ := strconv.Atoi(numStr)
	return n
}

// AtLeast returns true if v >= other.
func (v *Version) AtLeast(other *Version) bool {
	return v.Compare(other) >= 0
}

// LessThan returns true if v < other.
func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

// Feature represents a Typesense feature that may or may not be available
// depending on the server version.
type Feature string

// Feature constants for version-dependent functionality
const (
	// FeatureSynonymSets indicates support for system-level synonym sets (/synonym_sets)
	// Available in v30.0+
	FeatureSynonymSets Feature = "synonym_sets"

	// FeatureCurationSets indicates support for system-level curation sets (/curation_sets)
	// Available in v30.0+
	FeatureCurationSets Feature = "curation_sets"

	// FeaturePerCollectionSynonyms indicates support for per-collection synonyms
	// (/collections/{name}/synonyms)
	// Available in v29 and earlier
	FeaturePerCollectionSynonyms Feature = "per_collection_synonyms"

	// FeaturePerCollectionOverrides indicates support for per-collection overrides
	// (/collections/{name}/overrides)
	// Available in v29 and earlier
	FeaturePerCollectionOverrides Feature = "per_collection_overrides"

	// FeatureConversationModels indicates support for conversation models (RAG)
	// Available in v26.0+
	FeatureConversationModels Feature = "conversation_models"

	// FeaturePresets indicates support for search presets
	// Available in v27.0+
	FeaturePresets Feature = "presets"

	// FeatureStopwords indicates support for stopwords sets
	// Available in v27.0+
	FeatureStopwords Feature = "stopwords"

	// FeatureAnalyticsRules indicates support for analytics rules
	// Available in v28.0+
	FeatureAnalyticsRules Feature = "analytics_rules"

	// FeatureNLSearchModels indicates support for natural language search models
	// Available in v29.0+
	FeatureNLSearchModels Feature = "nl_search_models"

	// FeatureStemmingDictionaries indicates support for stemming dictionaries
	// Available in v29.0+
	FeatureStemmingDictionaries Feature = "stemming_dictionaries"
)

// featureVersions maps features to their minimum required version.
// nil means the feature has no minimum version (always available).
var featureVersions = map[Feature]*Version{
	FeatureSynonymSets:            V30_0,
	FeatureCurationSets:           V30_0,
	FeaturePerCollectionSynonyms:  nil, // Available in older versions, removed in v30
	FeaturePerCollectionOverrides: nil, // Available in older versions, removed in v30
	FeatureConversationModels:     V26_0,
	FeaturePresets:                V27_0,
	FeatureStopwords:              V27_0,
	FeatureAnalyticsRules:         V28_0,
	FeatureNLSearchModels:         V29_0,
	FeatureStemmingDictionaries:   V29_0,
}

// featureMaxVersions maps features to their maximum supported version (exclusive).
// nil means no upper bound.
var featureMaxVersions = map[Feature]*Version{
	FeaturePerCollectionSynonyms:  V30_0, // Removed in v30
	FeaturePerCollectionOverrides: V30_0, // Removed in v30
}

// FeatureChecker provides version-aware feature detection.
type FeatureChecker interface {
	// SupportsFeature returns true if the server supports the given feature.
	SupportsFeature(feature Feature) bool

	// GetVersion returns the server version, or nil if unknown.
	GetVersion() *Version
}

// DefaultFeatureChecker implements FeatureChecker with a known server version.
type DefaultFeatureChecker struct {
	version *Version
}

// NewFeatureChecker creates a new FeatureChecker for the given version.
// If version is nil, SupportsFeature will return false for all features
// that require version detection.
func NewFeatureChecker(version *Version) FeatureChecker {
	return &DefaultFeatureChecker{version: version}
}

// SupportsFeature returns true if the server version supports the given feature.
func (c *DefaultFeatureChecker) SupportsFeature(feature Feature) bool {
	if c.version == nil {
		// Without version info, we can't reliably determine feature support.
		// Return false to trigger fallback behavior.
		return false
	}

	// Check minimum version
	minVersion, hasMin := featureVersions[feature]
	if hasMin && minVersion != nil {
		if c.version.LessThan(minVersion) {
			return false
		}
	}

	// Check maximum version (exclusive)
	maxVersion, hasMax := featureMaxVersions[feature]
	if hasMax && maxVersion != nil {
		if c.version.AtLeast(maxVersion) {
			return false
		}
	}

	return true
}

// GetVersion returns the server version.
func (c *DefaultFeatureChecker) GetVersion() *Version {
	return c.version
}

// FallbackFeatureChecker is used when version detection fails.
// It returns a neutral response that allows the caller to fall back to
// runtime detection (e.g., 404 handling).
type FallbackFeatureChecker struct{}

// NewFallbackFeatureChecker creates a FeatureChecker for when version is unknown.
func NewFallbackFeatureChecker() FeatureChecker {
	return &FallbackFeatureChecker{}
}

// SupportsFeature always returns false, signaling that the caller should
// use runtime detection (try the API and handle 404s).
func (c *FallbackFeatureChecker) SupportsFeature(feature Feature) bool {
	return false
}

// GetVersion returns nil for the fallback checker.
func (c *FallbackFeatureChecker) GetVersion() *Version {
	return nil
}

// featureMinVersionString returns a human-readable minimum version string for a feature.
func featureMinVersionString(feature Feature) string {
	if minVer, ok := featureVersions[feature]; ok && minVer != nil {
		return fmt.Sprintf("v%d.%d+", minVer.Major, minVer.Minor)
	}
	return "unknown version"
}

// CheckVersionRequirement checks if the server version supports the given feature
// and returns an error diagnostic if it does not. When the server version is unknown
// (FallbackFeatureChecker), the check is skipped to allow runtime detection.
func CheckVersionRequirement(checker FeatureChecker, feature Feature, resourceName string) diag.Diagnostics {
	// If version is unknown, skip the guard and let the API call fail naturally.
	// This allows runtime detection via 404 handling.
	if checker.GetVersion() == nil {
		return nil
	}

	if !checker.SupportsFeature(feature) {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic(
				fmt.Sprintf("%s requires a newer Typesense version", resourceName),
				fmt.Sprintf(
					"The %s resource requires Typesense %s. Your server is running v%s. "+
						"Please upgrade your Typesense server or remove this resource from your configuration.",
					resourceName, featureMinVersionString(feature), checker.GetVersion().String(),
				),
			),
		}
	}
	return nil
}
