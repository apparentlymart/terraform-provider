package tfprovider

import (
	"context"

	"github.com/apparentlymart/terraform-provider/internal/tfplugin5"
	"github.com/apparentlymart/terraform-provider/tfprovider/tfschema"
	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"
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
}

func newTfplugin5Provider(plugin *rpcplugin.Plugin, clientProxy interface{}) *tfplugin5Provider {
	return &tfplugin5Provider{
		client: clientProxy.(tfplugin5.ProviderClient),
		plugin: plugin,
	}
}

func (p *tfplugin5Provider) isProvider() {}

func (p *tfplugin5Provider) Schema(ctx context.Context) (*tfschema.Provider, Diagnostics) {
	resp, err := p.client.GetSchema(ctx, &tfplugin5.GetProviderSchema_Request{})
	if err != nil {
		return nil, rpcErrorDiagnostics(err)
	}
	diags := tfplugin5Diagnostics(resp.Diagnostics)
	var ret tfschema.Provider
	ret.ProviderConfig = tfplugin5ProviderSchemaBlock(resp.Provider.Block)
	ret.ManagedResourceTypes = make(map[string]*tfschema.ManagedResourceType)
	for name, raw := range resp.ResourceSchemas {
		block := tfplugin5ProviderSchemaBlock(raw.Block)
		ret.ManagedResourceTypes[name] = &tfschema.ManagedResourceType{
			Version: raw.Version,
			Content: *block,
		}
	}
	ret.DataResourceTypes = make(map[string]*tfschema.DataResourceType)
	for name, raw := range resp.DataSourceSchemas {
		block := tfplugin5ProviderSchemaBlock(raw.Block)
		ret.DataResourceTypes[name] = &tfschema.DataResourceType{
			Content: *block,
		}
	}
	return &ret, diags
}

func (p *tfplugin5Provider) Close() error {
	return nil
}
