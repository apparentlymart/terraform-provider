package tfprovider

import (
	"context"
	"sync"

	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"

	"github.com/apparentlymart/terraform-provider/internal/tfplugin5"
	"github.com/zclconf/go-cty/cty"
)

// This file contains the Provider implementation and other related
// implementations for Terraform provider protocol major version 5,
// represented as the "tfplugin5" protobuf package.

type tfplugin5Client struct{}

func (c tfplugin5Client) ClientProxy(ctx context.Context, conn *grpc.ClientConn) (interface{}, error) {
	return tfplugin5.NewProviderClient(conn), nil
}

type tfplugin5Provider struct {
	client tfplugin5.ProviderClient
	plugin *rpcplugin.Plugin
	schema *Schema

	configured   bool
	configuredMu *sync.Mutex
}

func newTfplugin5Provider(ctx context.Context, plugin *rpcplugin.Plugin, clientProxy interface{}) (*tfplugin5Provider, error) {
	client := clientProxy.(tfplugin5.ProviderClient)

	// We proactively fetch the schema here because in practice there's no
	// practical thing you can do to a provider without it: we need it to
	// serialize any given values to msgpack.
	schema, err := tfplugin5LoadSchema(ctx, client)
	if err != nil {
		return nil, err
	}

	return &tfplugin5Provider{
		client:     client,
		plugin:     plugin,
		schema:     schema,
		configured: false,
	}, nil
}

func (p *tfplugin5Provider) isProvider() {}

func (p *tfplugin5Provider) Schema(ctx context.Context) (*Schema, Diagnostics) {
	return p.schema, nil
}

func (p *tfplugin5Provider) PrepareConfig(ctx context.Context, config cty.Value) (Config, Diagnostics) {
	dv, diags := tfplugin5EncodeDynamicValue(config, p.schema.ProviderConfig)
	if diags.HasErrors() {
		return Config{config}, diags
	}
	resp, err := p.client.PrepareProviderConfig(ctx, &tfplugin5.PrepareProviderConfig_Request{
		Config: dv,
	})
	diags = append(diags, rpcErrorDiagnostics(err)...)
	if err != nil {
		return Config{config}, diags
	}
	diags = append(diags, tfplugin5Diagnostics(resp.Diagnostics)...)
	if raw := resp.PreparedConfig; raw != nil {
		v, moreDiags := tfplugin5DecodeDynamicValue(raw, p.schema.ProviderConfig)
		diags = append(diags, moreDiags...)
		return Config{v}, diags
	}
	return Config{cty.DynamicVal}, diags
}

func (p *tfplugin5Provider) Configure(ctx context.Context, config Config) Diagnostics {
	p.configuredMu.Lock()
	defer p.configuredMu.Unlock()
	if p.configured {
		return Diagnostics{
			{
				Severity: Error,
				Summary:  "Provider already configured",
				Detail:   "This operation requires an unconfigured provider, but this provider was already configured.",
			},
		}
	}

	dv, diags := tfplugin5EncodeDynamicValue(config.Value, p.schema.ProviderConfig)
	if diags.HasErrors() {
		return diags
	}
	resp, err := p.client.Configure(ctx, &tfplugin5.Configure_Request{
		Config: dv,
	})
	diags = append(diags, rpcErrorDiagnostics(err)...)
	if err != nil {
		return diags
	}
	diags = append(diags, tfplugin5Diagnostics(resp.Diagnostics)...)
	if !diags.HasErrors() {
		p.configured = true
	}
	return diags
}

func (p *tfplugin5Provider) Close() error {
	return p.plugin.Close()
}

func (p *tfplugin5Provider) requireConfigured() Diagnostics {
	p.configuredMu.Lock()
	var diags Diagnostics
	if !p.configured {
		diags = append(diags, Diagnostic{
			Severity: Error,
			Summary:  "Provider unconfigured",
			Detail:   "This operation requires a configured provider, but this provider isn't configured yet.",
		})
	}
	p.configuredMu.Unlock()
	return diags
}
