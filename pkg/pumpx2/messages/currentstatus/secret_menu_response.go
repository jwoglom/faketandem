package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SecretMenuResponse is what the pump's secret menu shows: when it last
// connected, in pump epoch seconds, and a value whose meaning is unknown.
type SecretMenuResponse struct {
	messages.Base
	TimeOfLastConnectionTimestampSeconds int64 `pumpx2:"timeOfLastConnectionTimestampSeconds"`
	ReservedValue                        int64 `pumpx2:"reservedValue"`
}

var secretMenuResponseProps = messages.Props{OpCode: -67, Size: 8, Type: messages.Response, Request: "SecretMenuRequest"}

func init() {
	messages.Register("SecretMenuResponse", &SecretMenuResponse{}, secretMenuResponseProps,
		messages.Constructor{Params: "long timeOfLastConnectionTimestampSeconds, long reservedValue", Func: NewSecretMenuResponse},
	)
}

// NewSecretMenuResponse builds the response.
func NewSecretMenuResponse(timeOfLastConnectionTimestampSeconds, reservedValue int64) *SecretMenuResponse {
	m := &SecretMenuResponse{TimeOfLastConnectionTimestampSeconds: timeOfLastConnectionTimestampSeconds, ReservedValue: reservedValue}
	m.SetCargo(helpers.Combine(helpers.ToUint32(timeOfLastConnectionTimestampSeconds), helpers.ToUint32(reservedValue)))
	return m
}

// Parse implements messages.Message.
func (m *SecretMenuResponse) Parse(raw []byte) error {
	if len(raw) != secretMenuResponseProps.Size {
		return messages.InvalidLength("SecretMenuResponse", len(raw), secretMenuResponseProps.Size)
	}
	m.SetCargo(raw)
	m.TimeOfLastConnectionTimestampSeconds = helpers.ReadUint32(raw, 0)
	m.ReservedValue = helpers.ReadUint32(raw, 4)
	return nil
}
