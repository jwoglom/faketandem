package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetDexcomG7PairingCodeRequest gives the pump the pairing code of a Dexcom
// G7 sensor: two bytes, then six of zero.
type SetDexcomG7PairingCodeRequest struct {
	messages.Base
	PairingCode int32 `pumpx2:"pairingCode"`
}

var setDexcomG7PairingCodeRequestProps = messages.Props{OpCode: -4, Size: 8, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetDexcomG7PairingCodeResponse"}

func init() {
	messages.Register("SetDexcomG7PairingCodeRequest", &SetDexcomG7PairingCodeRequest{}, setDexcomG7PairingCodeRequestProps,
		messages.Constructor{Params: "int pairingCode", Func: NewSetDexcomG7PairingCodeRequest},
	)
}

// NewSetDexcomG7PairingCodeRequest builds the request.
func NewSetDexcomG7PairingCodeRequest(pairingCode int32) *SetDexcomG7PairingCodeRequest {
	m := &SetDexcomG7PairingCodeRequest{PairingCode: pairingCode}
	m.SetCargo(helpers.Combine(helpers.FirstTwoBytesLittleEndian(pairingCode), make([]byte, 6)))
	return m
}

// Parse implements messages.Message.
func (m *SetDexcomG7PairingCodeRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setDexcomG7PairingCodeRequestProps, raw)
	if len(raw) != setDexcomG7PairingCodeRequestProps.Size {
		return messages.InvalidLength("SetDexcomG7PairingCodeRequest", len(raw), setDexcomG7PairingCodeRequestProps.Size)
	}
	m.SetCargo(raw)
	m.PairingCode = helpers.ReadShort(raw, 0)
	return nil
}
