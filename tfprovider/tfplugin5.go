package tfprovider

import (
	"context"

	"go.rpcplugin.org/rpcplugin"
	"google.golang.org/grpc"

	"github.com/apparentlymart/terraform-provider/internal/tfplugin5"
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

func (p *tfplugin5Provider) Schema(ctx context.Context) (*Schema, Diagnostics) {
	resp, err := p.client.GetSchema(ctx, &tfplugin5.GetProviderSchema_Request{})
	if err != nil {
		return nil, rpcErrorDiagnostics(err)
	}
	diags := tfplugin5Diagnostics(resp.Diagnostics)
	var ret Schema
	ret.ProviderConfig = tfplugin5ProviderSchemaBlock(resp.Provider.Block)
	ret.ManagedResourceTypes = make(map[string]*ManagedResourceTypeSchema)
	for name, raw := range resp.ResourceSchemas {
		ret.ManagedResourceTypes[name] = &ManagedResourceTypeSchema{
			Version: raw.Version,
			Content: tfplugin5ProviderSchemaBlock(raw.Block),
		}
	}
	ret.DataResourceTypes = make(map[string]*DataResourceTypeSchema)
	for name, raw := range resp.DataSourceSchemas {
		ret.DataResourceTypes[name] = &DataResourceTypeSchema{
			Content: tfplugin5ProviderSchemaBlock(raw.Block),
		}
	}
	return &ret, diags
}

func (p *tfplugin5Provider) Close() error {
	return nil
}
