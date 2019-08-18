package tfprovider

import (
	"github.com/apparentlymart/terraform-provider/internal/tfplugin5"
	"github.com/apparentlymart/terraform-provider/tfprovider/tfschema"
)

func tfplugin5Diagnostics(raws []*tfplugin5.Diagnostic) Diagnostics {
	if len(raws) == 0 {
		return nil
	}
	diags := make(Diagnostics, 0, len(raws))
	for _, raw := range raws {
		diag := Diagnostic{
			Summary:   raw.Summary,
			Detail:    raw.Detail,
			Attribute: tfplugin5AttributePath(raw.Attribute),
		}

		switch raw.Severity {
		case tfplugin5.Diagnostic_ERROR:
			diag.Severity = Error
		case tfplugin5.Diagnostic_WARNING:
			diag.Severity = Warning
		}

		diags = append(diags, diag)
	}
	return diags
}

func tfplugin5AttributePath(raws *tfplugin5.AttributePath) tfschema.AttributePath {
	if raws == nil || len(raws.Steps) == 0 {
		return nil
	}
	ret := make(tfschema.AttributePath, 0, len(raws.Steps))
	for _, raw := range raws.Steps {
		switch s := raw.GetSelector().(type) {
		case *tfplugin5.AttributePath_Step_AttributeName:
			ret = append(ret, tfschema.AttributeName(s.AttributeName))
		case *tfplugin5.AttributePath_Step_ElementKeyString:
			ret = append(ret, tfschema.AttributeName(s.ElementKeyString))
		case *tfplugin5.AttributePath_Step_ElementKeyInt:
			ret = append(ret, tfschema.AttributeName(s.ElementKeyInt))
		default:
			ret = append(ret, nil)
		}
	}
	return ret
}
