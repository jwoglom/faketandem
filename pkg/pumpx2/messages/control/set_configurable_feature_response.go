package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetConfigurableFeatureResponse is the pump's answer to a
// SetConfigurableFeatureRequest: Status 0 means the feature changed, and
// Reason says why it did not.
type SetConfigurableFeatureResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
	Reason int32 `pumpx2:"reason"`
}

var setConfigurableFeatureResponseProps = messages.Props{OpCode: 117, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetConfigurableFeatureRequest"}

func init() {
	messages.Register("SetConfigurableFeatureResponse", &SetConfigurableFeatureResponse{}, setConfigurableFeatureResponseProps,
		messages.Constructor{Params: "int status, int reason", Func: NewSetConfigurableFeatureResponse},
	)
}

// NewSetConfigurableFeatureResponse builds the response.
func NewSetConfigurableFeatureResponse(status, reason int32) *SetConfigurableFeatureResponse {
	m := &SetConfigurableFeatureResponse{Status: status, Reason: reason}
	m.SetCargo([]byte{byte(status), byte(reason)})
	return m
}

// Parse implements messages.Message.
func (m *SetConfigurableFeatureResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setConfigurableFeatureResponseProps, raw)
	if len(raw) != setConfigurableFeatureResponseProps.Size {
		return messages.InvalidLength("SetConfigurableFeatureResponse", len(raw), setConfigurableFeatureResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.Reason = int32(raw[1])
	return nil
}
