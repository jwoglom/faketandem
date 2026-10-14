package authentication

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// Jpake1bRequest is the second half of the app's EC-JPAKE round 1, laid out
// as a Jpake1aRequest; the pump answers with a Jpake1bResponse.
type Jpake1bRequest struct {
	messages.Base
	AppInstanceID    int32  `pumpx2:"appInstanceId"`
	CentralChallenge []byte `pumpx2:"centralChallenge"`
}

var jpake1bRequestProps = messages.Props{OpCode: 34, Size: 167, Type: messages.Request, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Response: "Jpake1bResponse"}

func init() {
	messages.Register("Jpake1bRequest", &Jpake1bRequest{}, jpake1bRequestProps,
		messages.Constructor{Params: "byte[] rawCargo", Func: NewJpake1bRequestFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallenge", Func: NewJpake1bRequest},
	)
}

// NewJpake1bRequestFromCargo parses a request's payload.
func NewJpake1bRequestFromCargo(rawCargo []byte) (*Jpake1bRequest, error) {
	m := &Jpake1bRequest{}
	return m, m.Parse(rawCargo)
}

// NewJpake1bRequest builds the request around a round-1 half,
// centralChallenge.
func NewJpake1bRequest(appInstanceID int32, centralChallenge []byte) *Jpake1bRequest {
	m := &Jpake1bRequest{AppInstanceID: appInstanceID, CentralChallenge: centralChallenge}
	m.SetCargo(jpakeRoundCargo(jpake1bRequestProps.Size, appInstanceID, centralChallenge))
	return m
}

// Parse implements messages.Message. Like pumpX2's, it takes a payload of
// any length past the app instance ID as the round.
func (m *Jpake1bRequest) Parse(raw []byte) error {
	appInstanceID, challenge, err := parseChallenge("Jpake1bRequest", raw)
	if err != nil {
		return err
	}
	m.SetCargo(raw)
	m.AppInstanceID = appInstanceID
	m.CentralChallenge = challenge
	return nil
}
