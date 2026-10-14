package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetLowInsulinAlertRequest sets the low insulin alert's threshold. Only the
// threshold's low byte goes on the wire.
type SetLowInsulinAlertRequest struct {
	messages.Base
	InsulinThreshold int32 `pumpx2:"insulinThreshold"`
}

var setLowInsulinAlertRequestProps = messages.Props{OpCode: -34, Size: 1, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetLowInsulinAlertResponse"}

func init() {
	messages.Register("SetLowInsulinAlertRequest", &SetLowInsulinAlertRequest{}, setLowInsulinAlertRequestProps,
		messages.Constructor{Params: "int insulinThreshold", Func: NewSetLowInsulinAlertRequest},
	)
}

// NewSetLowInsulinAlertRequest builds the request.
func NewSetLowInsulinAlertRequest(insulinThreshold int32) *SetLowInsulinAlertRequest {
	m := &SetLowInsulinAlertRequest{InsulinThreshold: insulinThreshold}
	m.SetCargo([]byte{byte(insulinThreshold)})
	return m
}

// Parse implements messages.Message.
func (m *SetLowInsulinAlertRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setLowInsulinAlertRequestProps, raw)
	if len(raw) != setLowInsulinAlertRequestProps.Size {
		return messages.InvalidLength("SetLowInsulinAlertRequest", len(raw), setLowInsulinAlertRequestProps.Size)
	}
	m.SetCargo(raw)
	m.InsulinThreshold = int32(raw[0])
	return nil
}
