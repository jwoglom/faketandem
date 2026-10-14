package authentication

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpChallengeResponse, also known as AuthenticationStatusResponse, ends the
// legacy authorization: Success is false when the pairing code's HMAC in the
// PumpChallengeRequest was wrong.
type PumpChallengeResponse struct {
	messages.Base
	AppInstanceID int32 `pumpx2:"appInstanceId"`
	Success       bool  `pumpx2:"success"`
}

var pumpChallengeResponseProps = messages.Props{OpCode: 19, Size: 3, Type: messages.Response, Characteristic: messages.Authorization, Request: "PumpChallengeRequest"}

func init() {
	messages.Register("PumpChallengeResponse", &PumpChallengeResponse{}, pumpChallengeResponseProps,
		messages.Constructor{Params: "int appInstanceId, boolean success", Func: NewPumpChallengeResponse},
	)
}

// NewPumpChallengeResponse builds the response. The app instance ID reads
// back as its low two bytes.
func NewPumpChallengeResponse(appInstanceID int32, success bool) *PumpChallengeResponse {
	raw := helpers.Combine(helpers.FirstTwoBytesLittleEndian(appInstanceID), []byte{helpers.Bool(success)})
	m := &PumpChallengeResponse{AppInstanceID: helpers.ReadShort(raw, 0), Success: success}
	m.SetCargo(raw)
	return m
}

// Parse implements messages.Message.
func (m *PumpChallengeResponse) Parse(raw []byte) error {
	if len(raw) != pumpChallengeResponseProps.Size {
		return messages.InvalidLength("PumpChallengeResponse", len(raw), pumpChallengeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.AppInstanceID = helpers.ReadShort(raw, 0)
	m.Success = raw[2] == 1
	return nil
}
