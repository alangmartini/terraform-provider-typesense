// Package types contains shared types used across the provider
package types

import "github.com/alanm/terraform-provider-typesense/internal/client"

// ProviderData contains configured API clients
type ProviderData struct {
	CloudClient  *client.CloudClient
	ServerClient *client.ServerClient
}
