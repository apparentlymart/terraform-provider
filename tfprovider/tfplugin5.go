package tfprovider

import (
	"context"

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
	client     tfplugin5.ProviderClient
	plugin     *rpcplugin.Plugin
	schema     *Schema
	configured bool
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

func (p *tfplugin5Provider) Close() error {
	return p.plugin.Close()
}
