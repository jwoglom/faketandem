package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// Jpake1aRequest opens the authorization a 6-digit pairing code completes:
// the first half of the app's EC-JPAKE round 1, 165 bytes after the app
// instance ID, which the pump answers with a Jpake1aResponse.
type Jpake1aRequest struct {
	messages.Base
	AppInstanceID    int32  `pumpx2:"appInstanceId"`
	CentralChallenge []byte `pumpx2:"centralChallenge"`
}

var jpake1aRequestProps = messages.Props{OpCode: 32, Size: 167, Type: messages.Request, Characteristic: messages.Authorization, MinAPI: "API_V3_2", Response: "Jpake1aResponse"}

func init() {
	messages.Register("Jpake1aRequest", &Jpake1aRequest{}, jpake1aRequestProps,
		messages.Constructor{Params: "byte[] rawCargo", Func: NewJpake1aRequestFromCargo},
		messages.Constructor{Params: "int appInstanceId, byte[] centralChallenge", Func: NewJpake1aRequest},
	)
}

// NewJpake1aRequestFromCargo parses a request's payload.
func NewJpake1aRequestFromCargo(rawCargo []byte) (*Jpake1aRequest, error) {
	m := &Jpake1aRequest{}
	return m, m.Parse(rawCargo)
}

// NewJpake1aRequest builds the request around a round-1 half,
// centralChallenge.
func NewJpake1aRequest(appInstanceID int32, centralChallenge []byte) *Jpake1aRequest {
	m := &Jpake1aRequest{AppInstanceID: appInstanceID, CentralChallenge: centralChallenge}
	m.SetCargo(jpakeRoundCargo(jpake1aRequestProps.Size, appInstanceID, centralChallenge))
	return m
}

// Parse implements messages.Message. Like pumpX2's, it takes a payload of
// any length past the app instance ID as the round.
func (m *Jpake1aRequest) Parse(raw []byte) error {
	appInstanceID, challenge, err := parseChallenge("Jpake1aRequest", raw)
	if err != nil {
		return err
	}
	m.SetCargo(raw)
	m.AppInstanceID = appInstanceID
	m.CentralChallenge = challenge
	return nil
}

// jpakeRoundCargo is the app instance ID and a JPAKE round in a size-byte
// payload. A round with a minimal-length ZKP scalar can be a byte or two
// short, and is zero-padded, which the peer's length-prefixed parser skips;
// a long one is truncated.
func jpakeRoundCargo(size int, appInstanceID int32, round []byte) []byte {
	return helpers.CopyOfRange(helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), round), 0, size)
}
