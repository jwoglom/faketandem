package currentstatus

import (
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// TempRateStatusResponse is the temp basal rate in force: whether one is,
// its ID, its percentage of the profile rate, when it started (pump epoch
// seconds), the pump's seconds since reset, and its duration, in a unit
// pumpX2 does not document.
type TempRateStatusResponse struct {
	messages.Base
	Active                bool  `pumpx2:"active"`
	TempRateID            int32 `pumpx2:"tempRateId"`
	Percentage            int32 `pumpx2:"percentage"`
	StartTimeRaw          int64 `pumpx2:"startTimeRaw"`
	SecondsSincePumpReset int64 `pumpx2:"secondsSincePumpReset"`
	Duration              int64 `pumpx2:"duration"`
}

var tempRateStatusResponseProps = messages.Props{OpCode: 31, Size: 16, Type: messages.Response, Request: "TempRateStatusRequest"}

func init() {
	messages.Register("TempRateStatusResponse", &TempRateStatusResponse{}, tempRateStatusResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewTempRateStatusResponseFromCargo},
	)
}

// NewTempRateStatusResponseFromCargo parses a response's payload.
func NewTempRateStatusResponseFromCargo(raw []byte) (*TempRateStatusResponse, error) {
	m := &TempRateStatusResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *TempRateStatusResponse) Parse(raw []byte) error {
	if len(raw) != tempRateStatusResponseProps.Size {
		return messages.InvalidLength("TempRateStatusResponse", len(raw), tempRateStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Active = raw[0] != 0
	m.TempRateID = helpers.ReadShort(raw, 1)
	m.Percentage = int32(raw[3])
	m.StartTimeRaw = helpers.ReadUint32(raw, 4)
	m.SecondsSincePumpReset = helpers.ReadUint32(raw, 8)
	m.Duration = helpers.ReadUint32(raw, 12)
	return nil
}

// StartTimeInstant is when the temp rate started. It fails only for a
// StartTimeRaw beyond the range of a Java Instant.
func (m *TempRateStatusResponse) StartTimeInstant() (time.Time, error) {
	return helpers.FromJan12008EpochSecondsToDate(m.StartTimeRaw)
}
