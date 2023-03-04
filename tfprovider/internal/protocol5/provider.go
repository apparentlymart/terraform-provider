package protocol5

import (
	"context"
	"fmt"
	"sync"

	"github.com/apparentlymart/terraform-provider/internal/tfplugin5"
	"github.com/apparentlymart/terraform-provider/tfprovider/internal/common"
	"github.com/zclconf/go-cty/cty"
	"go.rpcplugin.org/rpcplugin"
)

// Provider is the implementation of tfprovider.Provider for provider plugin
// protocol version 5.
type Provider struct {
	client tfplugin5.ProviderClient
	plugin *rpcplugin.Plugin
	schema *common.Schema

	configured   bool
	configuredMu *sync.Mutex
}

func NewProvider(ctx context.Context, plugin *rpcplugin.Plugin, clientProxy interface{}) (*Provider, error) {
	client := clientProxy.(tfplugin5.ProviderClient)

	// We proactively fetch the schema here because you can't really do anything
	// useful to a provider without it: we need it to serialize any values given
	// in msgpack format.
	schema, err := loadSchema(ctx, client)
	if err != nil {
		return nil, err
	}

	return &Provider{
		client:       client,
		plugin:       plugin,
		schema:       schema,
		configured:   false,
		configuredMu: &sync.Mutex{},
	}, nil
}

func (p *Provider) Sealed() common.Sealed {
	return common.Sealed{}
}

func (p *Provider) Schema(ctx context.Context) (*common.Schema, common.Diagnostics) {
	return p.schema, nil
}

func (p *Provider) PrepareConfig(ctx context.Context, config cty.Value) (common.Config, common.Diagnostics) {
	dv, diags := encodeDynamicValue(config, p.schema.ProviderConfig)
	if diags.HasErrors() {
		return common.Config{config}, diags
	}
	resp, err := p.client.PrepareProviderConfig(ctx, &tfplugin5.PrepareProviderConfig_Request{
		Config: dv,
	})
	diags = append(diags, common.RPCErrorDiagnostics(err)...)
	if err != nil {
		return common.Config{config}, diags
	}
	diags = append(diags, decodeDiagnostics(resp.Diagnostics)...)
	if raw := resp.PreparedConfig; raw != nil {
		v, moreDiags := decodeDynamicValue(raw, p.schema.ProviderConfig)
		diags = append(diags, moreDiags...)
		return common.Config{v}, diags
	}
	return common.Config{cty.DynamicVal}, diags
}

func (p *Provider) Configure(ctx context.Context, config common.Config) common.Diagnostics {
	p.configuredMu.Lock()
	defer p.configuredMu.Unlock()
	if p.configured {
		return common.Diagnostics{
			{
				Severity: common.Error,
				Summary:  "Provider already configured",
				Detail:   "This operation requires an unconfigured provider, but this provider was already configured.",
			},
		}
	}

	dv, diags := encodeDynamicValue(config.Value, p.schema.ProviderConfig)
	if diags.HasErrors() {
		return diags
	}
	resp, err := p.client.Configure(ctx, &tfplugin5.Configure_Request{
		Config: dv,
	})
	diags = append(diags, common.RPCErrorDiagnostics(err)...)
	if err != nil {
		return diags
	}
	diags = append(diags, decodeDiagnostics(resp.Diagnostics)...)
	if !diags.HasErrors() {
		p.configured = true
	}
	return diags
}

func (p *Provider) ValidateManagedResourceConfig(ctx context.Context, typeName string, config cty.Value) common.Diagnostics {
	dv, diags := encodeDynamicValue(config, p.schema.ProviderConfig)
	if diags.HasErrors() {
		return diags
	}
	resp, err := p.client.ValidateResourceTypeConfig(ctx, &tfplugin5.ValidateResourceTypeConfig_Request{
		Config: dv,
	})
	diags = append(diags, common.RPCErrorDiagnostics(err)...)
	if err != nil {
		return diags
	}
	diags = append(diags, decodeDiagnostics(resp.Diagnostics)...)
	return diags
}

func (p *Provider) ValidateDataResourceConfig(ctx context.Context, typeName string, config cty.Value) common.Diagnostics {
	dv, diags := encodeDynamicValue(config, p.schema.ProviderConfig)
	if diags.HasErrors() {
		return diags
	}
	resp, err := p.client.ValidateDataSourceConfig(ctx, &tfplugin5.ValidateDataSourceConfig_Request{
		Config: dv,
	})
	diags = append(diags, common.RPCErrorDiagnostics(err)...)
	if err != nil {
		return diags
	}
	diags = append(diags, decodeDiagnostics(resp.Diagnostics)...)
	return diags
}

func (p *Provider) ManagedResourceType(typeName string) common.ManagedResourceType {
	p.configuredMu.Lock()
	if !p.configured {
		return nil
	}
	p.configuredMu.Unlock()

	schema, ok := p.schema.ManagedResourceTypes[typeName]
	if !ok {
		return nil
	}
	return &ManagedResourceType{
		client:   p.client,
		typeName: typeName,
		schema:   schema,
	}
}

func (p *Provider) ImportManagedResourceState(ctx context.Context, typeName string, id string) ([]common.ImportedResource, common.Diagnostics) {
	p.configuredMu.Lock()
	if !p.configured {
		return nil, nil
	}
	p.configuredMu.Unlock()

	var diags common.Diagnostics
	resp, err := p.client.ImportResourceState(ctx, &tfplugin5.ImportResourceState_Request{
		TypeName: typeName,
		Id:       id,
	})
	diags = append(diags, common.RPCErrorDiagnostics(err)...)
	if err != nil {
		return nil, diags
	}
	diags = append(diags, decodeDiagnostics(resp.Diagnostics)...)

	var resources []common.ImportedResource
	for _, raw := range resp.ImportedResources {
		resource := common.ImportedResource{
			TypeName:      raw.TypeName,
			OpaquePrivate: raw.Private,
		}
		schema, ok := p.schema.ManagedResourceTypes[raw.TypeName]
		if !ok {
			diags = append(diags, common.RPCErrorDiagnostics(fmt.Errorf("unknown resource type %q", raw.TypeName))...)
			continue
		}
		state, moreDiags := decodeDynamicValue(raw.State, schema.Content)
		resource.State = state
		diags = append(diags, moreDiags...)
		resources = append(resources, resource)
	}
	return resources, diags
}

func (p *Provider) Close() error {
	return p.plugin.Close()
}

func (p *Provider) requireConfigured() common.Diagnostics {
	p.configuredMu.Lock()
	var diags common.Diagnostics
	if !p.configured {
		diags = append(diags, common.Diagnostic{
			Severity: common.Error,
			Summary:  "Provider unconfigured",
			Detail:   "This operation requires a configured provider, but the provider isn't configured yet.",
		})
	}
	p.configuredMu.Unlock()
	return diags
}
