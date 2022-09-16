package common

import (
	"github.com/zclconf/go-cty/cty"
)

type ManagedResourceReadRequest struct {
	PreviousValue cty.Value
	OpaquePrivate []byte
}

type ManagedResourceReadResponse struct {
	RefreshedValue cty.Value
	OpaquePrivate  []byte
}
