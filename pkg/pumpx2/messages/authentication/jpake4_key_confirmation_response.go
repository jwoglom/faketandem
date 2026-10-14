package authentication

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// Jpake4KeyConfirmationResponse is the pump's key confirmation, laid out as
// the app's Jpake4KeyConfirmationRequest: its 8-byte nonce, 8 reserved zero
// bytes and the HMAC-SHA256 of that nonce under the derived key. A digest
// that does not match the app's own computation means the pairing code was
// wrong.
type Jpake4KeyConfirmationResponse struct {
	messages.Base
	AppInstanceID int32  `pumpx2:"appInstanceId"`
	Nonce         []byte `pumpx2:"nonce"`
	Reserved      []byte `pumpx2:"reserved"`
	HashDigest    []byte `pumpx2:"hashDigest"`
}

var jpake4KeyConfirmationResponseProps = messages.Props{OpCode: 41, Size: 50, Type: messages.Response, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Request: "Jpake4KeyConfirmationRequest"}

func init() {
	messages.Register("Jpake4KeyConfirmationResponse", &Jpake4KeyConfirmationResponse{}, jpake4KeyConfirmationResponseProps,
		messages.Constructor{Params: "byte[] rawCargo", Func: NewJpake4KeyConfirmationResponseFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] nonce, byte[] reserved, byte[] hashDigest", Func: NewJpake4KeyConfirmationResponse},
	)
}

// NewJpake4KeyConfirmationResponseFromCargo parses a response's payload.
func NewJpake4KeyConfirmationResponseFromCargo(rawCargo []byte) (*Jpake4KeyConfirmationResponse, error) {
	m := &Jpake4KeyConfirmationResponse{}
	return m, m.Parse(rawCargo)
}

// NewJpake4KeyConfirmationResponse builds the response. It fails unless
// nonce and reserved are 8 bytes and hashDigest 32.
func NewJpake4KeyConfirmationResponse(appInstanceID int32, nonce, reserved, hashDigest []byte) (*Jpake4KeyConfirmationResponse, error) {
	cargo, err := keyConfirmationCargo("Jpake4KeyConfirmationResponse", appInstanceID, nonce, reserved, hashDigest)
	if err != nil {
		return nil, err
	}
	m := &Jpake4KeyConfirmationResponse{AppInstanceID: appInstanceID, Nonce: nonce, Reserved: reserved, HashDigest: hashDigest}
	m.SetCargo(cargo)
	return m, nil
}

// Parse implements messages.Message. Like pumpX2's, it reads the first 50
// bytes of a longer payload and keeps the whole payload as the cargo.
func (m *Jpake4KeyConfirmationResponse) Parse(raw []byte) error {
	appInstanceID, nonce, reserved, hashDigest, err := parseKeyConfirmation("Jpake4KeyConfirmationResponse", raw)
	if err != nil {
		return err
	}
	m.SetCargo(raw)
	m.AppInstanceID = appInstanceID
	m.Nonce = nonce
	m.Reserved = reserved
	m.HashDigest = hashDigest
	return nil
}
