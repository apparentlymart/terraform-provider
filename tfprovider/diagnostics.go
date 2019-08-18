package tfprovider

import (
	"fmt"

	grpcStatus "google.golang.org/grpc/status"

	"github.com/apparentlymart/terraform-provider/tfprovider/tfattrs"
)

type Diagnostics []Diagnostic

type Diagnostic struct {
	Severity  DiagnosticSeverity
	Summary   string
	Detail    string
	Attribute tfattrs.Path
}

type DiagnosticSeverity rune

const (
	Error   DiagnosticSeverity = 'E'
	Warning DiagnosticSeverity = 'W'
)

func (diags Diagnostics) HasErrors() bool {
	for _, diag := range diags {
		if diag.Severity == Error {
			return true
		}
	}
	return false
}

func rpcErrorDiagnostics(err error) Diagnostics {
	var diags Diagnostics
	status, ok := grpcStatus.FromError(err)
	if !ok {
		diags = append(diags, Diagnostic{
			Severity: Error,
			Summary:  "Failed to call provider plugin",
			Detail:   fmt.Sprintf("Provider RPC call failed: %s.", err),
		})
	} else {
		diags = append(diags, Diagnostic{
			Severity: Error,
			Summary:  "Failed to call provider plugin",
			Detail:   fmt.Sprintf("Provider returned RPC error %s: %s.", status.Code(), status.Message()),
		})
	}
	return diags
}
