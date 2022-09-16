package tfproviderinst

import (
	"github.com/apparentlymart/go-versions/versions"
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

type ProviderVersion struct {
	Provider           tfaddr.Provider
	Version            versions.Version
	SupportedPlatforms []Platform
	SupportedProtocols []ProtocolVersion
}
