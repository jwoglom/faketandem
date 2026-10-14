package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Jpake4KeyConfirmationRequest is the app's key confirmation, the last step
// of EC-JPAKE authorization: an 8-byte nonce, 8 reserved zero bytes and the
// 32-byte HMAC-SHA256 of the nonce under the key derived from the shared
// secret and the pump's session key nonce. The pump answers with its own, in
// a Jpake4KeyConfirmationResponse.
type Jpake4KeyConfirmationRequest struct {
	messages.Base
	AppInstanceID int32  `pumpx2:"appInstanceId"`
	HashDigest    []byte `pumpx2:"hashDigest"`
	Reserved      []byte `pumpx2:"reserved"`
	Nonce         []byte `pumpx2:"nonce"`
}

var jpake4KeyConfirmationRequestProps = messages.Props{OpCode: 40, Size: 50, Type: messages.Request, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Response: "Jpake4KeyConfirmationResponse"}

func init() {
	messages.Register("Jpake4KeyConfirmationRequest", &Jpake4KeyConfirmationRequest{}, jpake4KeyConfirmationRequestProps,
		messages.Constructor{Params: "byte[] rawCargo", Func: NewJpake4KeyConfirmationRequestFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] nonce, byte[] reserved, byte[] hashDigest", Func: NewJpake4KeyConfirmationRequest},
	)
}

// NewJpake4KeyConfirmationRequestFromCargo parses a request's payload.
func NewJpake4KeyConfirmationRequestFromCargo(rawCargo []byte) (*Jpake4KeyConfirmationRequest, error) {
	m := &Jpake4KeyConfirmationRequest{}
	return m, m.Parse(rawCargo)
}

// NewJpake4KeyConfirmationRequest builds the request. It fails unless nonce
// and reserved are 8 bytes and hashDigest 32.
func NewJpake4KeyConfirmationRequest(appInstanceID int32, nonce, reserved, hashDigest []byte) (*Jpake4KeyConfirmationRequest, error) {
	cargo, err := keyConfirmationCargo("Jpake4KeyConfirmationRequest", appInstanceID, nonce, reserved, hashDigest)
	if err != nil {
		return nil, err
	}
	m := &Jpake4KeyConfirmationRequest{AppInstanceID: appInstanceID, Nonce: nonce, Reserved: reserved, HashDigest: hashDigest}
	m.SetCargo(cargo)
	return m, nil
}

// Parse implements messages.Message. Like pumpX2's, it reads the first 50
// bytes of a longer payload and keeps the whole payload as the cargo.
func (m *Jpake4KeyConfirmationRequest) Parse(raw []byte) error {
	appInstanceID, nonce, reserved, hashDigest, err := parseKeyConfirmation("Jpake4KeyConfirmationRequest", raw)
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

// keyConfirmationCargo is a Jpake4KeyConfirmation request's or response's
// payload.
func keyConfirmationCargo(name string, appInstanceID int32, nonce, reserved, hashDigest []byte) ([]byte, error) {
	switch {
	case len(nonce) != 8:
		return nil, messages.Invalid("%s: nonce was %d not 8", name, len(nonce))
	case len(reserved) != 8:
		return nil, messages.Invalid("%s: reserved was %d not 8", name, len(reserved))
	case len(hashDigest) != 32:
		return nil, messages.Invalid("%s: hashDigest was %d not 32", name, len(hashDigest))
	}
	return helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), nonce, reserved, hashDigest), nil
}

// parseKeyConfirmation reads a Jpake4KeyConfirmation request's or response's
// payload. A digest cut short by a payload under 50 bytes is zero-padded, as
// pumpX2's is.
func parseKeyConfirmation(name string, raw []byte) (appInstanceID int32, nonce, reserved, hashDigest []byte, err error) {
	if len(raw) < 18 {
		return 0, nil, nil, nil, messages.Invalid("%s: payload of %d bytes ends before its digest", name, len(raw))
	}
	return helpers.ReadShort(raw, 0), helpers.CopyOfRange(raw, 2, 10), helpers.CopyOfRange(raw, 10, 18), helpers.CopyOfRange(raw, 18, 50), nil
}
