package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CurrentActiveIdpValuesResponse is the active insulin delivery profile's
// values in force now, from its current time segment.
//
// pumpX2 reads CurrentTargetBg as a uint16 at byte 5, overlapping
// CurrentInsulinDuration at byte 6, and so does this: pumpX2's own capture
// reads a target of 11264, where a uint16 at byte 4 would be 115 mg/dL.
type CurrentActiveIdpValuesResponse struct {
	messages.Base
	// CurrentCarbRatio is grams per unit, times 1000.
	CurrentCarbRatio int64 `pumpx2:"currentCarbRatio"`
	// CurrentTargetBg is in mg/dL.
	CurrentTargetBg int32 `pumpx2:"currentTargetBg"`
	// CurrentInsulinDuration is in minutes.
	CurrentInsulinDuration int32 `pumpx2:"currentInsulinDuration"`
	// CurrentIsf is mg/dL per unit.
	CurrentIsf int32 `pumpx2:"currentIsf"`
}

var currentActiveIdpValuesResponseProps = messages.Props{OpCode: -105, Size: 10, Type: messages.Response, Request: "CurrentActiveIdpValuesRequest"}

func init() {
	messages.Register("CurrentActiveIdpValuesResponse", &CurrentActiveIdpValuesResponse{}, currentActiveIdpValuesResponseProps,
		messages.Constructor{
			Params: "long currentCarbRatio, int currentTargetBg, int currentInsulinDuration, int currentIsf",
			Func:   NewCurrentActiveIdpValuesResponse,
		},
	)
}

// NewCurrentActiveIdpValuesResponse builds the response as pumpX2 does, with
// only the target's low byte, at byte 5.
func NewCurrentActiveIdpValuesResponse(currentCarbRatio int64, currentTargetBg, currentInsulinDuration, currentIsf int32) *CurrentActiveIdpValuesResponse {
	m := &CurrentActiveIdpValuesResponse{
		CurrentCarbRatio:       currentCarbRatio,
		CurrentTargetBg:        currentTargetBg,
		CurrentInsulinDuration: currentInsulinDuration,
		CurrentIsf:             currentIsf,
	}
	m.SetCargo(helpers.Combine(
		helpers.ToUint32(currentCarbRatio),
		[]byte{0, byte(currentTargetBg), byte(currentInsulinDuration), byte(currentInsulinDuration >> 8)},
		helpers.FirstTwoBytesLittleEndian(currentIsf),
	))
	return m
}

// Parse implements messages.Message.
func (m *CurrentActiveIdpValuesResponse) Parse(raw []byte) error {
	if len(raw) != currentActiveIdpValuesResponseProps.Size {
		return messages.InvalidLength("CurrentActiveIdpValuesResponse", len(raw), currentActiveIdpValuesResponseProps.Size)
	}
	m.SetCargo(raw)
	m.CurrentCarbRatio = helpers.ReadUint32(raw, 0)
	m.CurrentTargetBg = helpers.ReadShort(raw, 5)
	m.CurrentInsulinDuration = helpers.ReadShort(raw, 6)
	m.CurrentIsf = helpers.ReadShort(raw, 8)
	return nil
}
