package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CentralChallengeRequest opens the legacy authorization, the one a
// 16-character pairing code completes: the app's instance ID and an 8-byte
// random challenge, which the pump answers with a CentralChallengeResponse.
type CentralChallengeRequest struct {
	messages.Base
	AppInstanceID    int32  `pumpx2:"appInstanceId"`
	CentralChallenge []byte `pumpx2:"centralChallenge"`
}

var centralChallengeRequestProps = messages.Props{OpCode: 16, Size: 10, Type: messages.Request, Characteristic: messages.Authorization, Response: "CentralChallengeResponse"}

func init() {
	messages.Register("CentralChallengeRequest", &CentralChallengeRequest{}, centralChallengeRequestProps,
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallenge", Func: NewCentralChallengeRequest},
	)
}

// NewCentralChallengeRequest builds the request. The payload takes the first
// 8 bytes of centralChallenge, and the request fails when it has fewer.
func NewCentralChallengeRequest(appInstanceID int32, centralChallenge []byte) (*CentralChallengeRequest, error) {
	cargo, err := fixedCargo("CentralChallengeRequest", centralChallengeRequestProps.Size, helpers.FirstTwoBytesLittleEndian(appInstanceID), centralChallenge)
	if err != nil {
		return nil, err
	}
	m := &CentralChallengeRequest{AppInstanceID: appInstanceID, CentralChallenge: centralChallenge}
	m.SetCargo(cargo)
	return m, nil
}

// Parse implements messages.Message. Like pumpX2's, it takes a payload of
// any length past the app instance ID as the challenge.
func (m *CentralChallengeRequest) Parse(raw []byte) error {
	appInstanceID, challenge, err := parseChallenge("CentralChallengeRequest", raw)
	if err != nil {
		return err
	}
	m.SetCargo(raw)
	m.AppInstanceID = appInstanceID
	m.CentralChallenge = challenge
	return nil
}

// parseChallenge reads the app instance ID and everything after it, as each
// authorization request that carries a challenge or a JPAKE round does.
func parseChallenge(name string, raw []byte) (appInstanceID int32, challenge []byte, err error) {
	if len(raw) < 2 {
		return 0, nil, messages.Invalid("%s: payload of %d bytes has no app instance ID", name, len(raw))
	}
	return helpers.ReadShort(raw, 0), helpers.CopyOfRange(raw, 2, len(raw)), nil
}

// fixedCargo is pumpX2's System.arraycopy of the parts, combined, into a new
// size-byte payload: it truncates them when longer and fails when shorter.
func fixedCargo(name string, size int, parts ...[]byte) ([]byte, error) {
	combined := helpers.Combine(parts...)
	if len(combined) < size {
		return nil, messages.Invalid("%s: %d bytes cannot fill its %d-byte payload", name, len(combined), size)
	}
	return helpers.CopyOfRange(combined, 0, size), nil
}
