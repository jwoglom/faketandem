package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Jpake2Response is the pump's EC-JPAKE round 2, 168 bytes after the app
// instance ID: three longer than the app's, for the curve parameters the
// server's round 2 leads with.
type Jpake2Response struct {
	messages.Base
	AppInstanceID        int32  `pumpx2:"appInstanceId"`
	CentralChallengeHash []byte `pumpx2:"centralChallengeHash"`
}

var jpake2ResponseProps = messages.Props{OpCode: 37, Size: 170, Type: messages.Response, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Request: "Jpake2Request"}

func init() {
	messages.Register("Jpake2Response", &Jpake2Response{}, jpake2ResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewJpake2ResponseFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallengeHash", Func: NewJpake2Response},
	)
}

// NewJpake2ResponseFromCargo parses a response's payload.
func NewJpake2ResponseFromCargo(raw []byte) (*Jpake2Response, error) {
	m := &Jpake2Response{}
	return m, m.Parse(raw)
}

// NewJpake2Response builds the response around round 2,
// centralChallengeHash, zero-padded to 168 bytes. It fails when the round is
// longer or appInstanceID does not fit two bytes.
func NewJpake2Response(appInstanceID int32, centralChallengeHash []byte) (*Jpake2Response, error) {
	m := &Jpake2Response{}
	if err := m.Parse(helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), centralChallengeHash)); err != nil {
		return nil, err
	}
	if m.AppInstanceID != appInstanceID {
		return nil, messages.Invalid("Jpake2Response: app instance ID %d does not fit two bytes", appInstanceID)
	}
	return m, nil
}

// Parse implements messages.Message.
func (m *Jpake2Response) Parse(raw []byte) error {
	cargo, appInstanceID, hash, err := parseJpakeRound("Jpake2Response", jpake2ResponseProps.Size, raw)
	if err != nil {
		return err
	}
	m.SetCargo(cargo)
	m.AppInstanceID = appInstanceID
	m.CentralChallengeHash = hash
	return nil
}
