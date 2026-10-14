package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// The temp basal rates SetTempRateRequest takes. The pump refuses others.
const (
	MinTempRateMinutes = 15
	MaxTempRateMinutes = 72 * 60
	MinTempRatePercent = 0
	MaxTempRatePercent = 250
)

// SetTempRateRequest starts a temp basal rate: Percent of the profile's
// basal rate for Minutes. Control-IQ must be off. The pump answers with the
// temp rate's ID, which its history log records it under.
type SetTempRateRequest struct {
	messages.Base
	Minutes int32 `pumpx2:"minutes"`
	Percent int32 `pumpx2:"percent"`
}

var setTempRateRequestProps = messages.Props{OpCode: -92, Size: 6, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Response: "SetTempRateResponse"}

func init() {
	messages.Register("SetTempRateRequest", &SetTempRateRequest{}, setTempRateRequestProps,
		messages.Constructor{Params: "int minutes, int percent", Func: NewSetTempRateRequest},
		messages.Constructor{Params: "byte[] raw", Func: NewSetTempRateRequestFromCargo},
	)
}

// NewSetTempRateRequest fails for a duration outside MinTempRateMinutes to
// MaxTempRateMinutes or a percent outside MinTempRatePercent to
// MaxTempRatePercent.
func NewSetTempRateRequest(minutes, percent int32) (*SetTempRateRequest, error) {
	if minutes < MinTempRateMinutes || minutes > MaxTempRateMinutes {
		return nil, messages.Invalid("SetTempRateRequest: duration of temp rate must be between 15 and 4,320 minutes (72 hours), not %d", minutes)
	}
	if percent < MinTempRatePercent || percent > MaxTempRatePercent {
		return nil, messages.Invalid("SetTempRateRequest: percent temp rate must be between 0-250%%, not %d", percent)
	}
	m := &SetTempRateRequest{Minutes: minutes, Percent: percent}
	m.SetCargo(helpers.Combine(helpers.ToUint32(int64(minutes)*60*1000), helpers.FirstTwoBytesLittleEndian(percent)))
	return m, nil
}

// NewSetTempRateRequestFromCargo parses a request's payload.
func NewSetTempRateRequestFromCargo(raw []byte) (*SetTempRateRequest, error) {
	m := &SetTempRateRequest{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message. The duration is in milliseconds on the
// wire, and Minutes rounds it down.
func (m *SetTempRateRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setTempRateRequestProps, raw)
	if len(raw) != setTempRateRequestProps.Size {
		return messages.InvalidLength("SetTempRateRequest", len(raw), setTempRateRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Minutes = int32(helpers.ReadUint32(raw, 0) / 1000 / 60)
	m.Percent = helpers.ReadShort(raw, 4)
	return nil
}
