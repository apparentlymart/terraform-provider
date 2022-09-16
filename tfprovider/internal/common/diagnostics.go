package common

import (
	"fmt"
	"strings"

	grpcStatus "google.golang.org/grpc/status"

	"github.com/zclconf/go-cty/cty"
)

type Diagnostics []Diagnostic

type Diagnostic struct {
	Severity  DiagnosticSeverity
	Summary   string
	Detail    string
	Attribute cty.Path
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

func RPCErrorDiagnostics(err error) Diagnostics {
	if err == nil {
		return nil
	}
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

func ErrorDiagnostics(summary, detailPrefix string, err error) Diagnostics {
	switch err := err.(type) {
	case nil:
		return nil
	case cty.PathError:
		return Diagnostics{
			{
				Severity:  Error,
				Summary:   summary,
				Detail:    fmt.Sprintf("%s: %s.", detailPrefix, err.Error()),
				Attribute: err.Path,
			},
		}
	default:
		return Diagnostics{
			{
				Severity: Error,
				Summary:  summary,
				Detail:   fmt.Sprintf("%s: %s.", detailPrefix, err.Error()),
			},
		}
	}

}

func FormatError(err error) string {
	switch err := err.(type) {
	case cty.PathError:
		return fmt.Sprintf("%s: %s", FormatCtyPath(err.Path), err.Error())
	default:
		return err.Error()
	}
}

func FormatCtyPath(path cty.Path) string {
	var buf strings.Builder
	for _, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			buf.WriteString("." + step.Name)
		case cty.IndexStep:
			switch step.Key.Type() {
			case cty.String:
				fmt.Fprintf(&buf, "[%q]", step.Key.AsString())
			case cty.Number:
				fmt.Fprintf(&buf, "[%s]", step.Key.AsBigFloat().Text('f', 0))
			default:
				buf.WriteString("[...]")
			}
		default:
			buf.WriteString("[...]")
		}
	}
	return buf.String()
}
