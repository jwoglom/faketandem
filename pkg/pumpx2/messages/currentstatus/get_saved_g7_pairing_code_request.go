package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// GetSavedG7PairingCodeRequest asks for the Dexcom G7 pairing code saved on
// the pump.
type GetSavedG7PairingCodeRequest struct {
	messages.Base
}

var getSavedG7PairingCodeRequestProps = messages.Props{OpCode: 116, Type: messages.Request, Response: "GetSavedG7PairingCodeResponse"}

func init() {
	messages.Register("GetSavedG7PairingCodeRequest", &GetSavedG7PairingCodeRequest{}, getSavedG7PairingCodeRequestProps,
		messages.Constructor{Func: NewGetSavedG7PairingCodeRequest},
	)
}

// NewGetSavedG7PairingCodeRequest is the request, which has no payload.
func NewGetSavedG7PairingCodeRequest() *GetSavedG7PairingCodeRequest {
	m := &GetSavedG7PairingCodeRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *GetSavedG7PairingCodeRequest) Parse(raw []byte) error {
	if len(raw) != getSavedG7PairingCodeRequestProps.Size {
		return messages.InvalidLength("GetSavedG7PairingCodeRequest", len(raw), getSavedG7PairingCodeRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
