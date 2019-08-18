package tfschema

import (
	"github.com/zclconf/go-cty/cty"
)

// Provider represents the overall schema for a provider.
type Provider struct {
	ProviderConfig       *BlockType
	ManagedResourceTypes map[string]*ManagedResourceType
	DataResourceTypes    map[string]*DataResourceType
}

type ManagedResourceType struct {
	Version int64
	Content BlockType
}

type DataResourceType struct {
	Content BlockType
}

type BlockType struct {
	Attributes       map[string]*Attribute
	NestedBlockTypes map[string]*NestedBlockType
}

type Attribute struct {
	Name        AttributeName
	Type        cty.Type
	Description string

	Optional bool
	Required bool
	Computed bool

	Sensitive bool
}

type NestedBlockType struct {
	TypeName AttributeName
	Nesting  NestingMode
	Content  BlockType
}

type NestingMode int

const (
	nestingInvalid NestingMode = iota
	NestingSingle
	NestingGroup
	NestingList
	NestingSet
	NestingMap
)
