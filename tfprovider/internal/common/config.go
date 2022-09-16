package common

import (
	"github.com/zclconf/go-cty/cty"
)

// Config represents a provider configuration that has already been prepared
// using Provider.PrepareConfig, ready to be passed to Configure.
type Config struct {
	Value cty.Value
}
