package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// BasalLimitSettingsResponse is the pump's maximum basal rate setting and
// its default, in milliunits per hour (5000 is 5 U/hr).
type BasalLimitSettingsResponse struct {
	messages.Base
	BasalLimit        int64 `pumpx2:"basalLimit"`
	BasalLimitDefault int64 `pumpx2:"basalLimitDefault"`
}

var basalLimitSettingsResponseProps = messages.Props{OpCode: -117, Size: 8, Type: messages.Response, Request: "BasalLimitSettingsRequest"}

func init() {
	messages.Register("BasalLimitSettingsResponse", &BasalLimitSettingsResponse{}, basalLimitSettingsResponseProps,
		messages.Constructor{Params: "long basalLimit, long basalLimitDefault", Func: NewBasalLimitSettingsResponse},
	)
}

// NewBasalLimitSettingsResponse builds the response.
func NewBasalLimitSettingsResponse(basalLimit, basalLimitDefault int64) *BasalLimitSettingsResponse {
	m := &BasalLimitSettingsResponse{BasalLimit: basalLimit, BasalLimitDefault: basalLimitDefault}
	m.SetCargo(helpers.Combine(helpers.ToUint32(basalLimit), helpers.ToUint32(basalLimitDefault)))
	return m
}

// Parse implements messages.Message.
func (m *BasalLimitSettingsResponse) Parse(raw []byte) error {
	if len(raw) != basalLimitSettingsResponseProps.Size {
		return messages.InvalidLength("BasalLimitSettingsResponse", len(raw), basalLimitSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BasalLimit = helpers.ReadUint32(raw, 0)
	m.BasalLimitDefault = helpers.ReadUint32(raw, 4)
	return nil
}
