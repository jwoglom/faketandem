package authentication

import (
	"bytes"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Jpake3SessionKeyResponse is the pump's 8-byte nonce, the HKDF salt with
// which both sides derive the key confirmation key from the EC-JPAKE secret,
// and 8 reserved bytes, all zero.
type Jpake3SessionKeyResponse struct {
	messages.Base
	AppInstanceID     int32  `pumpx2:"appInstanceId"`
	DeviceKeyNonce    []byte `pumpx2:"deviceKeyNonce"`
	DeviceKeyReserved []byte `pumpx2:"deviceKeyReserved"`
}

var jpake3SessionKeyResponseProps = messages.Props{OpCode: 39, Size: 18, Type: messages.Response, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Request: "Jpake3SessionKeyRequest"}

func init() {
	messages.Register("Jpake3SessionKeyResponse", &Jpake3SessionKeyResponse{}, jpake3SessionKeyResponseProps,
		messages.Constructor{Params: "int appInstanceId, byte[] nonce, byte[] reserved", Func: NewJpake3SessionKeyResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewJpake3SessionKeyResponseFromCargo},
	)
}

// NewJpake3SessionKeyResponse builds the response. It fails unless nonce and
// reserved are 8 bytes each and appInstanceID fits two bytes.
func NewJpake3SessionKeyResponse(appInstanceID int32, nonce, reserved []byte) (*Jpake3SessionKeyResponse, error) {
	m := &Jpake3SessionKeyResponse{}
	if err := m.Parse(helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), nonce, reserved)); err != nil {
		return nil, err
	}
	if m.AppInstanceID != appInstanceID || !bytes.Equal(m.DeviceKeyNonce, nonce) || !bytes.Equal(m.DeviceKeyReserved, reserved) {
		return nil, messages.Invalid("Jpake3SessionKeyResponse: app instance ID %d, a %d-byte nonce and %d reserved bytes do not read back", appInstanceID, len(nonce), len(reserved))
	}
	return m, nil
}

// NewJpake3SessionKeyResponseFromCargo parses a response's payload.
func NewJpake3SessionKeyResponseFromCargo(raw []byte) (*Jpake3SessionKeyResponse, error) {
	m := &Jpake3SessionKeyResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *Jpake3SessionKeyResponse) Parse(raw []byte) error {
	if len(raw) != jpake3SessionKeyResponseProps.Size {
		return messages.InvalidLength("Jpake3SessionKeyResponse", len(raw), jpake3SessionKeyResponseProps.Size)
	}
	m.SetCargo(raw)
	m.AppInstanceID = helpers.ReadShort(raw, 0)
	m.DeviceKeyNonce = helpers.CopyOfRange(raw, 2, 10)
	m.DeviceKeyReserved = helpers.CopyOfRange(raw, 10, 18)
	return nil
}
