package tfprovider

import (
	"context"
	"fmt"

	"github.com/apparentlymart/terraform-schema-go/tfschema"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/apparentlymart/terraform-provider/internal/tfplugin5"
)

func tfplugin5ProviderSchemaBlock(raw *tfplugin5.Schema_Block) *tfschema.Block {
	var ret tfschema.Block
	if raw == nil {
		return &ret
	}

	ret.Attributes = make(map[string]*tfschema.Attribute)
	ret.BlockTypes = make(map[string]*tfschema.NestedBlock)

	for _, rawAttr := range raw.Attributes {
		rawType := rawAttr.Type
		ty, err := ctyjson.UnmarshalType(rawType)
		if err != nil {
			// If the provider sends us an invalid type then we'll just
			// replace it with dynamic, since the provider is misbehaving.
			ty = cty.DynamicPseudoType
		}

		ret.Attributes[rawAttr.Name] = &tfschema.Attribute{
			Type:        ty,
			Description: rawAttr.Description,

			Required:  rawAttr.Required,
			Optional:  rawAttr.Optional,
			Computed:  rawAttr.Computed,
			Sensitive: rawAttr.Sensitive,
		}
	}

	for _, rawBlock := range raw.BlockTypes {
		var mode tfschema.NestingMode
		switch rawBlock.Nesting {
		case tfplugin5.Schema_NestedBlock_SINGLE:
			mode = tfschema.NestingSingle
		case tfplugin5.Schema_NestedBlock_GROUP:
			mode = tfschema.NestingGroup
		case tfplugin5.Schema_NestedBlock_LIST:
			mode = tfschema.NestingList
		case tfplugin5.Schema_NestedBlock_SET:
			mode = tfschema.NestingSet
		case tfplugin5.Schema_NestedBlock_MAP:
			mode = tfschema.NestingMap
		}

		content := tfplugin5ProviderSchemaBlock(rawBlock.Block)

		ret.BlockTypes[rawBlock.TypeName] = &tfschema.NestedBlock{
			Nesting: mode,
			Block:   *content,
		}
	}

	return &ret
}

func tfplugin5LoadSchema(ctx context.Context, client tfplugin5.ProviderClient) (*Schema, error) {
	resp, err := client.GetSchema(ctx, &tfplugin5.GetProviderSchema_Request{})
	if err != nil {
		return nil, err
	}
	diags := tfplugin5Diagnostics(resp.Diagnostics)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to retrieve provider schema")
	}
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
	return &ret, nil
}

func tfplugin5EncodeDynamicValue(val cty.Value, schema *tfschema.Block) (*tfplugin5.DynamicValue, Diagnostics) {
	ty := schema.ImpliedType()
	raw, err := ctymsgpack.Marshal(val, ty)
	if err != nil {
		return nil, errorDiagnostics(
			"Invalid object",
			"Value does not have the required type",
			err,
		)
	}
	return &tfplugin5.DynamicValue{
		Msgpack: raw,
	}, nil
}

func tfplugin5DecodeDynamicValue(raw *tfplugin5.DynamicValue, schema *tfschema.Block) (cty.Value, Diagnostics) {
	ty := schema.ImpliedType()
	switch {
	case len(raw.Json) > 0:
		val, err := ctyjson.Unmarshal(raw.Json, ty)
		if err != nil {
			return cty.DynamicVal, errorDiagnostics(
				"Provider returned invalid object",
				"Provider's JSON response does not conform to the expected type",
				err,
			)
		}
		return val, nil
	case len(raw.Msgpack) > 0:
		val, err := ctymsgpack.Unmarshal(raw.Json, ty)
		if err != nil {
			return cty.DynamicVal, errorDiagnostics(
				"Provider returned invalid object",
				"Provider's msgpack response does not conform to the expected type",
				err,
			)
		}
		return val, nil
	default:
		return cty.DynamicVal, Diagnostics{
			{
				Severity: Error,
				Summary:  "Provider using unsupported response format",
				Detail:   "Provider's response is not in either JSON or msgpack format",
			},
		}
	}
}
