package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BolusPermissionRequest asks the pump for permission to bolus, which it
// grants with a bolus ID for the InitiateBolusRequest. It has no payload.
type BolusPermissionRequest struct {
	messages.Base
}

var bolusPermissionRequestProps = messages.Props{OpCode: -94, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	MinAPI: "API_V2_5", Response: "BolusPermissionResponse"}

func init() {
	messages.Register("BolusPermissionRequest", &BolusPermissionRequest{}, bolusPermissionRequestProps,
		messages.Constructor{Func: NewBolusPermissionRequest},
	)
}

// NewBolusPermissionRequest builds the request.
func NewBolusPermissionRequest() *BolusPermissionRequest {
	m := &BolusPermissionRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *BolusPermissionRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(bolusPermissionRequestProps, raw)
	if len(raw) != bolusPermissionRequestProps.Size {
		return messages.InvalidLength("BolusPermissionRequest", len(raw), bolusPermissionRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
