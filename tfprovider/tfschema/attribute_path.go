package tfschema

import (
	"fmt"
)

type AttributePath []AttributePathStep

type AttributePathStep interface {
	attributePathStep()
	String() string
}

type AttributeName string

func (n AttributeName) attributePathStep() {}

func (n AttributeName) String() string {
	return fmt.Sprintf(".%s", string(n))
}

type MapKey string

func (k MapKey) attributePathStep() {}

func (k MapKey) String() string {
	return fmt.Sprintf("[%q]", string(k))
}

type ListIndex int64

func (i ListIndex) attributePathStep() {}

func (i ListIndex) String() string {
	return fmt.Sprintf("[%d]", int64(i))
}
