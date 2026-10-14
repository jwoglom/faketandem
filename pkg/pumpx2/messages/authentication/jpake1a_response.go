package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Jpake1aResponse is the first half of the pump's EC-JPAKE round 1, 165
// bytes after the app instance ID.
type Jpake1aResponse struct {
	messages.Base
	AppInstanceID        int32  `pumpx2:"appInstanceId"`
	CentralChallengeHash []byte `pumpx2:"centralChallengeHash"`
}

var jpake1aResponseProps = messages.Props{OpCode: 33, Size: 167, Type: messages.Response, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Request: "Jpake1aRequest"}

func init() {
	messages.Register("Jpake1aResponse", &Jpake1aResponse{}, jpake1aResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewJpake1aResponseFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallengeHash", Func: NewJpake1aResponse},
	)
}

// NewJpake1aResponseFromCargo parses a response's payload.
func NewJpake1aResponseFromCargo(raw []byte) (*Jpake1aResponse, error) {
	m := &Jpake1aResponse{}
	return m, m.Parse(raw)
}

// NewJpake1aResponse builds the response around a round-1 half,
// centralChallengeHash, zero-padded to 165 bytes. It fails when the round is
// longer or appInstanceID does not fit two bytes.
func NewJpake1aResponse(appInstanceID int32, centralChallengeHash []byte) (*Jpake1aResponse, error) {
	m := &Jpake1aResponse{}
	if err := m.Parse(helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), centralChallengeHash)); err != nil {
		return nil, err
	}
	if m.AppInstanceID != appInstanceID {
		return nil, messages.Invalid("Jpake1aResponse: app instance ID %d does not fit two bytes", appInstanceID)
	}
	return m, nil
}

// Parse implements messages.Message.
func (m *Jpake1aResponse) Parse(raw []byte) error {
	cargo, appInstanceID, hash, err := parseJpakeRound("Jpake1aResponse", jpake1aResponseProps.Size, raw)
	if err != nil {
		return err
	}
	m.SetCargo(cargo)
	m.AppInstanceID = appInstanceID
	m.CentralChallengeHash = hash
	return nil
}

// parseJpakeRound reads a pump's JPAKE round response of size bytes. A round
// with a minimal-length ZKP scalar, about 1 in 256 per scalar, can be a byte
// or two short of size; it is zero-padded back to size, which leaves the
// length-prefixed fields inside it readable. A longer one fails.
func parseJpakeRound(name string, size int, raw []byte) (cargo []byte, appInstanceID int32, hash []byte, err error) {
	if len(raw) > size {
		return nil, 0, nil, messages.Invalid("%s: size is %d not %d", name, len(raw), size)
	}
	cargo = helpers.CopyOfRange(raw, 0, size)
	return cargo, helpers.ReadShort(cargo, 0), helpers.CopyOfRange(cargo, 2, size), nil
}
