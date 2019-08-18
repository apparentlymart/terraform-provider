package tfprovider

import (
	"github.com/apparentlymart/terraform-schema-go/tfschema"
)

type Schema struct {
	ProviderConfig       *tfschema.Block
	ManagedResourceTypes map[string]*ManagedResourceTypeSchema
	DataResourceTypes    map[string]*DataResourceTypeSchema
}

type ManagedResourceTypeSchema struct {
	Version int64
	Content *tfschema.Block
}

type DataResourceTypeSchema struct {
	Content *tfschema.Block
}
