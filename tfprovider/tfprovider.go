// Package tfprovider is a client library for the Terraform provider plugin
// API, allowing Go programs to call into Terraform provider plugins without
// using code from Terraform itself.
//
// This package currently implements clients for protocol versions 5 and 6.
// In particular, that means it isn't compatible with provider plugins that
// are only compatible with Terraform v0.11 and earlier.
package tfprovider

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/apparentlymart/terraform-provider/tfprovider/internal/common"
	"github.com/apparentlymart/terraform-provider/tfprovider/internal/protocol5"
	"github.com/zclconf/go-cty/cty"
	"go.rpcplugin.org/rpcplugin"
)

// Provider represents a running provider plugin.
type Provider interface {
	// Schema retrieves the full schema for the provider.
	Schema(ctx context.Context) (*Schema, Diagnostics)

	// PrepareConfig validates and normalizes an object representing a provider
	// configuration, returning either the normalized object or error
	// diagnostics describing any problems with it.
	PrepareConfig(ctx context.Context, config cty.Value) (Config, Diagnostics)

	// Configure configures the provider using the given configuration.
	//
	// Each provider instance can be configured only once. If this method
	// is called more than once, subsequent calls will return errors.
	//
	// The given Config must have been prepared using PrepareConfig.
	Configure(ctx context.Context, config Config) Diagnostics

	// ValidateManagedResourceConfig runs the provider's validation logic
	// for a particular managed resource type.
	ValidateManagedResourceConfig(ctx context.Context, typeName string, config cty.Value) Diagnostics

	// ValidateDataResourceConfig runs the provider's validation logic
	// for a particular managed resource type.
	ValidateDataResourceConfig(ctx context.Context, typeName string, config cty.Value) Diagnostics

	// Close kills the child process for this provider plugin, rendering the
	// reciever unusable. Any further calls on the object after Close returns
	// cause undefined behavior.
	//
	// Calling Close also invalidates any associated objects such as
	// resource type objects.
	Close() error

	// Sealed is a do-nothing method that exists only to represent that this
	// interface may not be implemented by any type outside of this module,
	// to allow the interface to expand in future to support new provider
	// plugin protocol features.
	Sealed() common.Sealed
}

// ManagedResourceType represents a managed resource type belonging to a
// provider.
//
// This interface will grow in future versions of this module to support
// new protocol features, so no packages outside of this module should attempt
// to implement it.
type ManagedResourceType interface {
	// Sealed is a do-nothing method that exists only to represent that this
	// interface may not be implemented by any type outside of this module,
	// to allow the interface to expand in future to support new provider
	// plugin protocol features.
	Sealed() common.Sealed
}

// DataResourceType represents a data resource type (a data source) belonging
// to a provider.
//
// This interface will grow in future versions of this module to support
// new protocol features, so no packages outside of this module should attempt
// to implement it.
type DataResourceType interface {
	// Sealed is a do-nothing method that exists only to represent that this
	// interface may not be implemented by any type outside of this module,
	// to allow the interface to expand in future to support new provider
	// plugin protocol features.
	Sealed() common.Sealed
}

// Start executes the given command line as a Terraform provider plugin
// and returns an object representing it.
//
// The provider is initially unconfigured, meaning that it can only be used
// for object validation tasks. It must be configured (that is, it must be
// provided with a valid configuration object) before it can take any
// non-validation actions.
//
// Terraform providers run as child processes, so if this function returns
// successfully there will be a new child process beneath the calling process
// waiting to recieve provider commands. Be sure to call Close on the returned
// object when you no longer need the provider, so that the child process
// can be killed.
//
// Terraform provider executables conventionally have names starting with
// "terraform-provider-", because that is the prefix Terraform itself looks
// for in order to discover them automatically.
func Start(ctx context.Context, exe string, args ...string) (Provider, error) {
	plugin, err := rpcplugin.New(ctx, &rpcplugin.ClientConfig{
		Handshake: rpcplugin.HandshakeConfig{
			CookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
			CookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
		},
		Cmd: exec.Command(exe, args...),
		ProtoVersions: map[int]rpcplugin.ClientVersion{
			5: protocol5.PluginClient{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch provider plugin: %s", err)
	}

	protoVersion, clientProxy, err := plugin.Client(ctx)
	if err != nil {
		plugin.Close()
		return nil, fmt.Errorf("failed to create plugin client: %s", err)
	}

	switch protoVersion {
	case 5:
		return protocol5.NewProvider(ctx, plugin, clientProxy)
	default:
		// Should not be possible to get here because the above cases cover
		// all of the versions we listed in ProtoVersions; rpcplugin bug?
		panic(fmt.Sprintf("unsupported protocol version %d", protoVersion))
	}
}
