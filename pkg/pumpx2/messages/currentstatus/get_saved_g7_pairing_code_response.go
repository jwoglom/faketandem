package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// GetSavedG7PairingCodeResponse is the Dexcom G7 pairing code saved on the
// pump.
type GetSavedG7PairingCodeResponse struct {
	messages.Base
	PairingCode int32 `pumpx2:"pairingCode"`
}

var getSavedG7PairingCodeResponseProps = messages.Props{OpCode: 117, Size: 2, Type: messages.Response, Request: "GetSavedG7PairingCodeRequest"}

func init() {
	messages.Register("GetSavedG7PairingCodeResponse", &GetSavedG7PairingCodeResponse{}, getSavedG7PairingCodeResponseProps,
		messages.Constructor{Params: "int pairingCode", Func: NewGetSavedG7PairingCodeResponse},
	)
}

// NewGetSavedG7PairingCodeResponse builds the response.
func NewGetSavedG7PairingCodeResponse(pairingCode int32) *GetSavedG7PairingCodeResponse {
	m := &GetSavedG7PairingCodeResponse{PairingCode: pairingCode}
	m.SetCargo(helpers.FirstTwoBytesLittleEndian(pairingCode))
	return m
}

// Parse implements messages.Message.
func (m *GetSavedG7PairingCodeResponse) Parse(raw []byte) error {
	if len(raw) != getSavedG7PairingCodeResponseProps.Size {
		return messages.InvalidLength("GetSavedG7PairingCodeResponse", len(raw), getSavedG7PairingCodeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.PairingCode = helpers.ReadShort(raw, 0)
	return nil
}
