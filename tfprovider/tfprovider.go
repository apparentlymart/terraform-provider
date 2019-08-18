// Package tfprovider is a client library for the Terraform provider plugin
// API, allowing Go programs to call into Terraform provider plugins without
// using code from Terraform itself.
//
// This package only implements Terraform provider protocol version 5. That
// means it is only compatible with providers that are themselves compatible
// with Terraform 0.12, where protocol version 5 was introduced.
package tfprovider

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/zclconf/go-cty/cty"
	"go.rpcplugin.org/rpcplugin"
)

// Provider represents a running provider plugin that hasn't
// been configured yet.
type Provider interface {
	// We don't allow external implementations because this interface might
	// grow in future versions if the Terraform provider API surface area
	// also grows.
	isProvider()

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

	// Close kills the child process for this provider plugin, rendering the
	// reciever unusable. Any further calls on the object after Close returns
	// cause undefined behavior.
	//
	// Calling Close also invalidates any associated Provider object that
	// was created by calling Configure.
	Close() error
}

// ManagedResourceType represents a managed resource type belonging to a
// provider.
type ManagedResourceType interface {
	isManagedResourceType()
}

// DataResourceType represents a data resource type (a data source) belonging
// to a provider.
type DataResourceType interface {
	isDataResourceType()
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
			5: tfplugin5Client{},
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
		return newTfplugin5Provider(ctx, plugin, clientProxy)
	default:
		// Should not be possible to get here because the above cases cover
		// all of the versions we listed in ProtoVersions; rpcplugin bug?
		panic(fmt.Sprintf("unsupported protocol version %d", protoVersion))
	}
}
