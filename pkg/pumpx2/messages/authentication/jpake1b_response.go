package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Jpake1bResponse is the second half of the pump's EC-JPAKE round 1, laid
// out as a Jpake1aResponse.
type Jpake1bResponse struct {
	messages.Base
	AppInstanceID        int32  `pumpx2:"appInstanceId"`
	CentralChallengeHash []byte `pumpx2:"centralChallengeHash"`
}

var jpake1bResponseProps = messages.Props{OpCode: 35, Size: 167, Type: messages.Response, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Request: "Jpake1bRequest"}

func init() {
	messages.Register("Jpake1bResponse", &Jpake1bResponse{}, jpake1bResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewJpake1bResponseFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallengeHash", Func: NewJpake1bResponse},
	)
}

// NewJpake1bResponseFromCargo parses a response's payload.
func NewJpake1bResponseFromCargo(raw []byte) (*Jpake1bResponse, error) {
	m := &Jpake1bResponse{}
	return m, m.Parse(raw)
}

// NewJpake1bResponse builds the response around a round-1 half,
// centralChallengeHash, zero-padded to 165 bytes. It fails when the round is
// longer or appInstanceID does not fit two bytes.
func NewJpake1bResponse(appInstanceID int32, centralChallengeHash []byte) (*Jpake1bResponse, error) {
	m := &Jpake1bResponse{}
	if err := m.Parse(helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), centralChallengeHash)); err != nil {
		return nil, err
	}
	if m.AppInstanceID != appInstanceID {
		return nil, messages.Invalid("Jpake1bResponse: app instance ID %d does not fit two bytes", appInstanceID)
	}
	return m, nil
}

// Parse implements messages.Message.
func (m *Jpake1bResponse) Parse(raw []byte) error {
	cargo, appInstanceID, hash, err := parseJpakeRound("Jpake1bResponse", jpake1bResponseProps.Size, raw)
	if err != nil {
		return err
	}
	m.SetCargo(cargo)
	m.AppInstanceID = appInstanceID
	m.CentralChallengeHash = hash
	return nil
}
