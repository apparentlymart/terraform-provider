package tfprovider

import (
	"github.com/apparentlymart/terraform-schema-go/tfschema"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

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
