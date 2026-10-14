package authentication

import (
	"bytes"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpChallengeRequest completes the legacy authorization: the 20-byte
// HMAC-SHA1 of the 16-character pairing code under the key the pump sent in
// its CentralChallengeResponse.
type PumpChallengeRequest struct {
	messages.Base
	AppInstanceID     int32  `pumpx2:"appInstanceId"`
	PumpChallengeHash []byte `pumpx2:"pumpChallengeHash"`
}

var pumpChallengeRequestProps = messages.Props{OpCode: 18, Size: 22, Type: messages.Request, Characteristic: messages.Authorization, Response: "PumpChallengeResponse"}

func init() {
	messages.Register("PumpChallengeRequest", &PumpChallengeRequest{}, pumpChallengeRequestProps,
		messages.Constructor{Params: "int appInstanceId, byte[] pumpChallengeHash", Func: NewPumpChallengeRequest},
	)
}

// NewPumpChallengeRequest builds the request. It fails unless
// pumpChallengeHash is 20 bytes and appInstanceID fits two bytes.
func NewPumpChallengeRequest(appInstanceID int32, pumpChallengeHash []byte) (*PumpChallengeRequest, error) {
	cargo, err := fixedCargo("PumpChallengeRequest", pumpChallengeRequestProps.Size, helpers.FirstTwoBytesLittleEndian(appInstanceID), pumpChallengeHash)
	if err != nil {
		return nil, err
	}
	m := &PumpChallengeRequest{}
	if err := m.Parse(cargo); err != nil {
		return nil, err
	}
	if m.AppInstanceID != appInstanceID || !bytes.Equal(m.PumpChallengeHash, pumpChallengeHash) {
		return nil, messages.Invalid("PumpChallengeRequest: app instance ID %d and a %d-byte hash do not read back", appInstanceID, len(pumpChallengeHash))
	}
	return m, nil
}

// Parse implements messages.Message. Like pumpX2's, it takes a payload of
// any length past the app instance ID as the hash.
func (m *PumpChallengeRequest) Parse(raw []byte) error {
	appInstanceID, hash, err := parseChallenge("PumpChallengeRequest", raw)
	if err != nil {
		return err
	}
	m.SetCargo(raw)
	m.AppInstanceID = appInstanceID
	m.PumpChallengeHash = hash
	return nil
}
