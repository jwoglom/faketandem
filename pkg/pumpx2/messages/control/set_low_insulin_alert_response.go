package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetLowInsulinAlertResponse is the pump's answer to a
// SetLowInsulinAlertRequest: Status 0 is ACK, 1 NACK (refused) and 2 UNKNOWN.
type SetLowInsulinAlertResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setLowInsulinAlertResponseProps = messages.Props{OpCode: -33, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetLowInsulinAlertRequest"}

func init() {
	messages.Register("SetLowInsulinAlertResponse", &SetLowInsulinAlertResponse{}, setLowInsulinAlertResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetLowInsulinAlertResponse},
	)
}

// NewSetLowInsulinAlertResponse builds the response.
func NewSetLowInsulinAlertResponse(status int32) *SetLowInsulinAlertResponse {
	m := &SetLowInsulinAlertResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetLowInsulinAlertResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setLowInsulinAlertResponseProps, raw)
	if len(raw) != setLowInsulinAlertResponseProps.Size {
		return messages.InvalidLength("SetLowInsulinAlertResponse", len(raw), setLowInsulinAlertResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
