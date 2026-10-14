package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// The range SetMaxBolusLimitRequest takes, in milliunits.
const (
	MinBolusLimitMilliunits = 1000
	MaxBolusLimitMilliunits = 25000
)

// SetMaxBolusLimitRequest sets the pump's maximum bolus, in milliunits.
type SetMaxBolusLimitRequest struct {
	messages.Base
	MaxBolusMilliunits int32 `pumpx2:"maxBolusMilliunits"`
}

var setMaxBolusLimitRequestProps = messages.Props{OpCode: -122, Size: 2, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetMaxBolusLimitResponse"}

func init() {
	messages.Register("SetMaxBolusLimitRequest", &SetMaxBolusLimitRequest{}, setMaxBolusLimitRequestProps,
		messages.Constructor{Params: "int maxBolusMilliunits", Func: NewSetMaxBolusLimitRequest},
	)
}

// NewSetMaxBolusLimitRequest fails outside MinBolusLimitMilliunits to
// MaxBolusLimitMilliunits.
func NewSetMaxBolusLimitRequest(maxBolusMilliunits int32) (*SetMaxBolusLimitRequest, error) {
	switch {
	case maxBolusMilliunits < MinBolusLimitMilliunits:
		return nil, messages.Invalid("SetMaxBolusLimitRequest: bolus limit must be greater than 1 unit")
	case maxBolusMilliunits > MaxBolusLimitMilliunits:
		return nil, messages.Invalid("SetMaxBolusLimitRequest: bolus limit must be less than or equal to 25 units")
	}
	m := &SetMaxBolusLimitRequest{MaxBolusMilliunits: maxBolusMilliunits}
	m.SetCargo(helpers.FirstTwoBytesLittleEndian(maxBolusMilliunits))
	return m, nil
}

// Parse implements messages.Message.
func (m *SetMaxBolusLimitRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setMaxBolusLimitRequestProps, raw)
	if len(raw) != setMaxBolusLimitRequestProps.Size {
		return messages.InvalidLength("SetMaxBolusLimitRequest", len(raw), setMaxBolusLimitRequestProps.Size)
	}
	m.SetCargo(raw)
	m.MaxBolusMilliunits = helpers.ReadShort(raw, 0)
	return nil
}
