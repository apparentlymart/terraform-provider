package tfprovider

import (
	"github.com/apparentlymart/terraform-provider/tfprovider/internal/common"
)

// The following are re-exports of symbols from our internal "common" package.
// We have this indirection so that the protocol-specific subdirectories can
// avoid depending on tfprovider and therefore creating a package dependency
// cycle.

type Schema = common.Schema

type ManagedResourceTypeSchema = common.Schema

type DataResourceTypeSchema = common.Schema

type Diagnostics = common.Diagnostics

type Diagnostic = common.Diagnostic

type DiagnosticSeverity = common.DiagnosticSeverity

const (
	Error   DiagnosticSeverity = common.Error
	Warning DiagnosticSeverity = common.Warning
)

// Config represents a provider configuration that has already been prepared
// using Provider.PrepareConfig, ready to be passed to Configure.
type Config = common.Config

type ManagedResourceType = common.ManagedResourceType

type DataResourceType = common.DataResourceType

type ManagedResourceReadRequest = common.ManagedResourceReadRequest

type ManagedResourceReadResponse = common.ManagedResourceReadResponse
