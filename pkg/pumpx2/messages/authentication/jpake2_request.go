package authentication

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// Jpake2Request is the app's EC-JPAKE round 2, laid out as a Jpake1aRequest;
// the round is half as long as round 1, so it fits one message. The pump
// answers with a Jpake2Response.
type Jpake2Request struct {
	messages.Base
	AppInstanceID    int32  `pumpx2:"appInstanceId"`
	CentralChallenge []byte `pumpx2:"centralChallenge"`
}

var jpake2RequestProps = messages.Props{OpCode: 36, Size: 167, Type: messages.Request, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Response: "Jpake2Response"}

func init() {
	messages.Register("Jpake2Request", &Jpake2Request{}, jpake2RequestProps,
		messages.Constructor{Params: "byte[] rawCargo", Func: NewJpake2RequestFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallenge", Func: NewJpake2Request},
	)
}

// NewJpake2RequestFromCargo parses a request's payload.
func NewJpake2RequestFromCargo(rawCargo []byte) (*Jpake2Request, error) {
	m := &Jpake2Request{}
	return m, m.Parse(rawCargo)
}

// NewJpake2Request builds the request around round 2, centralChallenge.
func NewJpake2Request(appInstanceID int32, centralChallenge []byte) *Jpake2Request {
	m := &Jpake2Request{AppInstanceID: appInstanceID, CentralChallenge: centralChallenge}
	m.SetCargo(jpakeRoundCargo(jpake2RequestProps.Size, appInstanceID, centralChallenge))
	return m
}

// Parse implements messages.Message. Like pumpX2's, it takes a payload of
// any length past the app instance ID as the round.
func (m *Jpake2Request) Parse(raw []byte) error {
	appInstanceID, challenge, err := parseChallenge("Jpake2Request", raw)
	if err != nil {
		return err
	}
	m.SetCargo(raw)
	m.AppInstanceID = appInstanceID
	m.CentralChallenge = challenge
	return nil
}
