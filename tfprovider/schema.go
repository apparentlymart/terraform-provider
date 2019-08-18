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

func (s *Schema) HasManagedResourceType(name string) bool {
	_, ok := s.ManagedResourceTypes[name]
	return ok
}

func (s *Schema) HasDataResourceType(name string) bool {
	_, ok := s.DataResourceTypes[name]
	return ok
}
