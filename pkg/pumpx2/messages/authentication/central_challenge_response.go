package authentication

import (
	"bytes"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CentralChallengeResponse is the pump's answer to a CentralChallengeRequest:
// a 20-byte hash of the app's challenge and the 8-byte key the app HMACs the
// pairing code with for its PumpChallengeRequest.
type CentralChallengeResponse struct {
	messages.Base
	AppInstanceID        int32  `pumpx2:"appInstanceId"`
	CentralChallengeHash []byte `pumpx2:"centralChallengeHash"`
	HmacKey              []byte `pumpx2:"hmacKey"`
}

var centralChallengeResponseProps = messages.Props{OpCode: 17, Size: 30, Type: messages.Response, Characteristic: messages.Authorization, Request: "CentralChallengeRequest"}

func init() {
	messages.Register("CentralChallengeResponse", &CentralChallengeResponse{}, centralChallengeResponseProps,
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallengeHash, byte[] hmacKey", Func: NewCentralChallengeResponse},
	)
}

// NewCentralChallengeResponse builds the response. It fails unless
// centralChallengeHash is 20 bytes, hmacKey 8 and appInstanceID fits two
// bytes.
func NewCentralChallengeResponse(appInstanceID int32, centralChallengeHash, hmacKey []byte) (*CentralChallengeResponse, error) {
	m := &CentralChallengeResponse{}
	if err := m.Parse(helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), centralChallengeHash, hmacKey)); err != nil {
		return nil, err
	}
	if m.AppInstanceID != appInstanceID || !bytes.Equal(m.CentralChallengeHash, centralChallengeHash) || !bytes.Equal(m.HmacKey, hmacKey) {
		return nil, messages.Invalid("CentralChallengeResponse: app instance ID %d, a %d-byte hash and a %d-byte key do not read back", appInstanceID, len(centralChallengeHash), len(hmacKey))
	}
	return m, nil
}

// Parse implements messages.Message.
func (m *CentralChallengeResponse) Parse(raw []byte) error {
	if len(raw) != centralChallengeResponseProps.Size {
		return messages.InvalidLength("CentralChallengeResponse", len(raw), centralChallengeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.AppInstanceID = helpers.ReadShort(raw, 0)
	m.CentralChallengeHash = helpers.CopyOfRange(raw, 2, 22)
	m.HmacKey = helpers.CopyOfRange(raw, 22, 30)
	return nil
}
