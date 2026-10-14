package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// NonControlIQIOBResponse is the insulin on board of a pump without
// Control-IQ. Its payload is laid out as the first twelve bytes of
// ControlIQIOBResponse's, so IOB and TotalIOB are presumably in milliunits
// and TimeRemaining in seconds.
type NonControlIQIOBResponse struct {
	messages.Base
	IOB           int64 `pumpx2:"iob"`
	TimeRemaining int64 `pumpx2:"timeRemaining"`
	TotalIOB      int64 `pumpx2:"totalIOB"`
}

var nonControlIQIOBResponseProps = messages.Props{OpCode: 39, Size: 12, Type: messages.Response, Request: "NonControlIQIOBRequest"}

func init() {
	messages.Register("NonControlIQIOBResponse", &NonControlIQIOBResponse{}, nonControlIQIOBResponseProps,
		messages.Constructor{Params: "long iob, long timeRemaining, long totalIOB", Func: NewNonControlIQIOBResponse},
	)
}

// NewNonControlIQIOBResponse builds the response.
func NewNonControlIQIOBResponse(iob, timeRemaining, totalIOB int64) *NonControlIQIOBResponse {
	m := &NonControlIQIOBResponse{IOB: iob, TimeRemaining: timeRemaining, TotalIOB: totalIOB}
	m.SetCargo(helpers.Combine(helpers.ToUint32(iob), helpers.ToUint32(timeRemaining), helpers.ToUint32(totalIOB)))
	return m
}

// Parse implements messages.Message.
func (m *NonControlIQIOBResponse) Parse(raw []byte) error {
	if len(raw) != nonControlIQIOBResponseProps.Size {
		return messages.InvalidLength("NonControlIQIOBResponse", len(raw), nonControlIQIOBResponseProps.Size)
	}
	m.IOB = helpers.ReadUint32(raw, 0)
	m.TimeRemaining = helpers.ReadUint32(raw, 4)
	m.TotalIOB = helpers.ReadUint32(raw, 8)
	m.SetCargo(raw)
	return nil
}
