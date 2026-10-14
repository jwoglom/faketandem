package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// GlobalMaxBolusSettingsResponse is the pump's maximum bolus setting and its
// default, in milliunits.
type GlobalMaxBolusSettingsResponse struct {
	messages.Base
	MaxBolus        int32 `pumpx2:"maxBolus"`
	MaxBolusDefault int32 `pumpx2:"maxBolusDefault"`
}

var globalMaxBolusSettingsResponseProps = messages.Props{OpCode: -115, Size: 4, Type: messages.Response, Request: "GlobalMaxBolusSettingsRequest"}

func init() {
	messages.Register("GlobalMaxBolusSettingsResponse", &GlobalMaxBolusSettingsResponse{}, globalMaxBolusSettingsResponseProps,
		messages.Constructor{Params: "int maxBolus, int maxBolusDefault", Func: NewGlobalMaxBolusSettingsResponse},
	)
}

// NewGlobalMaxBolusSettingsResponse builds the response.
func NewGlobalMaxBolusSettingsResponse(maxBolus, maxBolusDefault int32) *GlobalMaxBolusSettingsResponse {
	m := &GlobalMaxBolusSettingsResponse{MaxBolus: maxBolus, MaxBolusDefault: maxBolusDefault}
	m.SetCargo(helpers.Combine(helpers.FirstTwoBytesLittleEndian(maxBolus), helpers.FirstTwoBytesLittleEndian(maxBolusDefault)))
	return m
}

// Parse implements messages.Message.
func (m *GlobalMaxBolusSettingsResponse) Parse(raw []byte) error {
	if len(raw) != globalMaxBolusSettingsResponseProps.Size {
		return messages.InvalidLength("GlobalMaxBolusSettingsResponse", len(raw), globalMaxBolusSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.MaxBolus = helpers.ReadShort(raw, 0)
	m.MaxBolusDefault = helpers.ReadShort(raw, 2)
	return nil
}
