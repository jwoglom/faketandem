package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ChangeControlIQSettingsRequest turns Control-IQ on or off and sets its
// weight, in pounds, and total daily insulin, in units.
type ChangeControlIQSettingsRequest struct {
	messages.Base
	Enabled   bool  `pumpx2:"enabled"`
	WeightLbs int32 `pumpx2:"weightLbs"`
	// TotalDailyInsulinUnits is read as a signed byte, so a total over 127
	// reads back negative.
	TotalDailyInsulinUnits int32 `pumpx2:"totalDailyInsulinUnits"`
}

var changeControlIQSettingsRequestProps = messages.Props{OpCode: -54, Size: 6, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "ChangeControlIQSettingsResponse"}

func init() {
	messages.Register("ChangeControlIQSettingsRequest", &ChangeControlIQSettingsRequest{}, changeControlIQSettingsRequestProps,
		messages.Constructor{Params: "boolean enabled, int weightLbs, int totalDailyInsulinUnits", Func: NewChangeControlIQSettingsRequest},
		messages.Constructor{Params: "byte[] raw", Func: NewChangeControlIQSettingsRequestFromCargo},
	)
}

// NewChangeControlIQSettingsRequest builds the request. Its fields are what
// the payload reads back as.
func NewChangeControlIQSettingsRequest(enabled bool, weightLbs, totalDailyInsulinUnits int32) *ChangeControlIQSettingsRequest {
	m := &ChangeControlIQSettingsRequest{}
	// pumpX2 always writes 1 to bytes 3 and 5, which it does not name.
	m.decode(helpers.Combine(
		[]byte{helpers.Bool(enabled)},
		helpers.FirstTwoBytesLittleEndian(weightLbs),
		[]byte{1, byte(totalDailyInsulinUnits), 1},
	))
	return m
}

// NewChangeControlIQSettingsRequestFromCargo parses a request's payload.
func NewChangeControlIQSettingsRequestFromCargo(raw []byte) (*ChangeControlIQSettingsRequest, error) {
	m := &ChangeControlIQSettingsRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *ChangeControlIQSettingsRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(changeControlIQSettingsRequestProps, raw)
	if len(raw) != changeControlIQSettingsRequestProps.Size {
		return messages.InvalidLength("ChangeControlIQSettingsRequest", len(raw), changeControlIQSettingsRequestProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *ChangeControlIQSettingsRequest) decode(raw []byte) {
	m.SetCargo(raw)
	m.Enabled = raw[0] == 1
	m.WeightLbs = helpers.ReadShort(raw, 1)
	m.TotalDailyInsulinUnits = int32(int8(raw[4]))
}
