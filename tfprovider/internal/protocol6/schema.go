package protocol6

import (
	"context"
	"fmt"

	"github.com/apparentlymart/terraform-schema-go/tfschema"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/apparentlymart/terraform-provider/internal/tfplugin6"
	"github.com/apparentlymart/terraform-provider/tfprovider/internal/common"
)

func decodeProviderSchemaBlock(raw *tfplugin6.Schema_Block) *tfschema.Block {
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
		case tfplugin6.Schema_NestedBlock_SINGLE:
			mode = tfschema.NestingSingle
		case tfplugin6.Schema_NestedBlock_GROUP:
			mode = tfschema.NestingGroup
		case tfplugin6.Schema_NestedBlock_LIST:
			mode = tfschema.NestingList
		case tfplugin6.Schema_NestedBlock_SET:
			mode = tfschema.NestingSet
		case tfplugin6.Schema_NestedBlock_MAP:
			mode = tfschema.NestingMap
		}

		content := decodeProviderSchemaBlock(rawBlock.Block)

		ret.BlockTypes[rawBlock.TypeName] = &tfschema.NestedBlock{
			Nesting: mode,
			Block:   *content,
		}
	}

	return &ret
}

func loadSchema(ctx context.Context, client tfplugin6.ProviderClient) (*common.Schema, error) {
	resp, err := client.GetProviderSchema(ctx, &tfplugin6.GetProviderSchema_Request{})
	if err != nil {
		return nil, err
	}
	diags := decodeDiagnostics(resp.Diagnostics)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to retrieve provider schema")
	}
	var ret common.Schema
	ret.ProviderConfig = decodeProviderSchemaBlock(resp.Provider.Block)
	ret.ManagedResourceTypes = make(map[string]*common.ManagedResourceTypeSchema)
	for name, raw := range resp.ResourceSchemas {
		ret.ManagedResourceTypes[name] = &common.ManagedResourceTypeSchema{
			Version: raw.Version,
			Content: decodeProviderSchemaBlock(raw.Block),
		}
	}
	ret.DataResourceTypes = make(map[string]*common.DataResourceTypeSchema)
	for name, raw := range resp.DataSourceSchemas {
		ret.DataResourceTypes[name] = &common.DataResourceTypeSchema{
			Content: decodeProviderSchemaBlock(raw.Block),
		}
	}
	return &ret, nil
}

func encodeDynamicValue(val cty.Value, schema *tfschema.Block) (*tfplugin6.DynamicValue, common.Diagnostics) {
	ty := schema.ImpliedType()
	raw, err := ctymsgpack.Marshal(val, ty)
	if err != nil {
		return nil, common.ErrorDiagnostics(
			"Invalid object",
			"Value does not have the required type",
			err,
		)
	}
	return &tfplugin6.DynamicValue{
		Msgpack: raw,
	}, nil
}

func decodeDynamicValue(raw *tfplugin6.DynamicValue, schema *tfschema.Block) (cty.Value, common.Diagnostics) {
	ty := schema.ImpliedType()
	switch {
	case len(raw.Json) > 0:
		val, err := ctyjson.Unmarshal(raw.Json, ty)
		if err != nil {
			return cty.DynamicVal, common.ErrorDiagnostics(
				"Provider returned invalid object",
				"Provider's JSON response does not conform to the expected type",
				err,
			)
		}
		return val, nil
	case len(raw.Msgpack) > 0:
		val, err := ctymsgpack.Unmarshal(raw.Json, ty)
		if err != nil {
			return cty.DynamicVal, common.ErrorDiagnostics(
				"Provider returned invalid object",
				"Provider's msgpack response does not conform to the expected type",
				err,
			)
		}
		return val, nil
	default:
		return cty.DynamicVal, common.Diagnostics{
			{
				Severity: common.Error,
				Summary:  "Provider using unsupported response format",
				Detail:   "Provider's response is not in either JSON or msgpack format",
			},
		}
	}
}
