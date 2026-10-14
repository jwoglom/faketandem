package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CurrentBasalStatusResponse is the basal rate the profile sets now and the
// rate in force, in milliunits per hour, and how the latter departs from the
// former. pumpX2 leaves BasalModifiedBitmask opaque; a pump suspended with no
// temp rate running has been captured reporting 1.
type CurrentBasalStatusResponse struct {
	messages.Base
	ProfileBasalRate     int64 `pumpx2:"profileBasalRate"`
	CurrentBasalRate     int64 `pumpx2:"currentBasalRate"`
	BasalModifiedBitmask int32 `pumpx2:"basalModifiedBitmask"`
}

var currentBasalStatusResponseProps = messages.Props{OpCode: 41, Size: 9, Type: messages.Response, Request: "CurrentBasalStatusRequest"}

func init() {
	messages.Register("CurrentBasalStatusResponse", &CurrentBasalStatusResponse{}, currentBasalStatusResponseProps,
		messages.Constructor{Params: "long profileBasalRate, long currentBasalRate, int basalModifiedBitmask", Func: NewCurrentBasalStatusResponse},
	)
}

// NewCurrentBasalStatusResponse builds the response.
func NewCurrentBasalStatusResponse(profileBasalRate, currentBasalRate int64, basalModifiedBitmask int32) *CurrentBasalStatusResponse {
	m := &CurrentBasalStatusResponse{ProfileBasalRate: profileBasalRate, CurrentBasalRate: currentBasalRate, BasalModifiedBitmask: basalModifiedBitmask}
	m.SetCargo(helpers.Combine(
		helpers.ToUint32(profileBasalRate),
		helpers.ToUint32(currentBasalRate),
		[]byte{byte(basalModifiedBitmask)},
	))
	return m
}

// Parse implements messages.Message.
func (m *CurrentBasalStatusResponse) Parse(raw []byte) error {
	if len(raw) != currentBasalStatusResponseProps.Size {
		return messages.InvalidLength("CurrentBasalStatusResponse", len(raw), currentBasalStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.ProfileBasalRate = helpers.ReadUint32(raw, 0)
	m.CurrentBasalRate = helpers.ReadUint32(raw, 4)
	m.BasalModifiedBitmask = int32(int8(raw[8]))
	return nil
}
