package protocol6

import (
	"context"

	"github.com/apparentlymart/terraform-provider/internal/tfplugin6"
	"github.com/apparentlymart/terraform-provider/tfprovider/internal/common"
)

type ManagedResourceType struct {
	client   tfplugin6.ProviderClient
	typeName string
	schema   *common.ManagedResourceTypeSchema
}

func (rt *ManagedResourceType) Read(ctx context.Context, req common.ManagedResourceReadRequest) (common.ManagedResourceReadResponse, common.Diagnostics) {
	resp := common.ManagedResourceReadResponse{}
	dv, diags := encodeDynamicValue(req.PreviousValue, rt.schema.Content)
	if diags.HasErrors() {
		return resp, diags
	}

	rawResp, err := rt.client.ReadResource(ctx, &tfplugin6.ReadResource_Request{
		TypeName:     rt.typeName,
		CurrentState: dv,
		Private:      req.OpaquePrivate,
	})
	diags = append(diags, common.RPCErrorDiagnostics(err)...)
	if err != nil {
		return resp, diags
	}
	diags = append(diags, decodeDiagnostics(rawResp.Diagnostics)...)

	if raw := rawResp.NewState; raw != nil {
		v, moreDiags := decodeDynamicValue(raw, rt.schema.Content)
		resp.RefreshedValue = v
		diags = append(diags, moreDiags...)
	}
	resp.OpaquePrivate = rawResp.Private
	return resp, diags
}

func (rt *ManagedResourceType) Sealed() common.Sealed {
	return common.Sealed{}
}
