package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// The range SetMaxBasalLimitRequest takes, in milliunits per hour.
const (
	MinBasalLimitMilliunits = 1000
	MaxBasalLimitMilliunits = 15000
)

// SetMaxBasalLimitRequest sets the pump's maximum hourly basal rate, in
// milliunits per hour.
type SetMaxBasalLimitRequest struct {
	messages.Base
	MaxHourlyBasalMilliunits int64 `pumpx2:"maxHourlyBasalMilliunits"`
}

var setMaxBasalLimitRequestProps = messages.Props{OpCode: -120, Size: 4, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetMaxBasalLimitResponse"}

func init() {
	messages.Register("SetMaxBasalLimitRequest", &SetMaxBasalLimitRequest{}, setMaxBasalLimitRequestProps,
		messages.Constructor{Params: "long maxHourlyBasalMilliunits", Func: NewSetMaxBasalLimitRequest},
	)
}

// NewSetMaxBasalLimitRequest fails outside MinBasalLimitMilliunits to
// MaxBasalLimitMilliunits.
func NewSetMaxBasalLimitRequest(maxHourlyBasalMilliunits int64) (*SetMaxBasalLimitRequest, error) {
	switch {
	case maxHourlyBasalMilliunits < MinBasalLimitMilliunits:
		return nil, messages.Invalid("SetMaxBasalLimitRequest: hourly basal limit must be greater than 1 unit")
	case maxHourlyBasalMilliunits > MaxBasalLimitMilliunits:
		return nil, messages.Invalid("SetMaxBasalLimitRequest: basal limit must be less than or equal to 15 units")
	}
	m := &SetMaxBasalLimitRequest{MaxHourlyBasalMilliunits: maxHourlyBasalMilliunits}
	m.SetCargo(helpers.ToUint32(maxHourlyBasalMilliunits))
	return m, nil
}

// Parse implements messages.Message.
func (m *SetMaxBasalLimitRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setMaxBasalLimitRequestProps, raw)
	if len(raw) != setMaxBasalLimitRequestProps.Size {
		return messages.InvalidLength("SetMaxBasalLimitRequest", len(raw), setMaxBasalLimitRequestProps.Size)
	}
	m.SetCargo(raw)
	m.MaxHourlyBasalMilliunits = helpers.ReadUint32(raw, 0)
	return nil
}
