// Package types contains shared types used across the provider
package types

import (
	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/alanm/terraform-provider-typesense/internal/version"
)

// ProviderData contains configured API clients and server version information
type ProviderData struct {
	CloudClient  *client.CloudClient
	ServerClient *client.ServerClient

	// ServerVersion is the parsed version of the Typesense server.
	// May be nil if version detection failed or server is not configured.
	ServerVersion *version.Version

	// FeatureChecker provides version-aware feature detection.
	// When ServerVersion is nil, this will be a FallbackFeatureChecker
	// that returns false for all features, triggering runtime detection.
	FeatureChecker version.FeatureChecker
}
