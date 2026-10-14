package currentstatus

import (
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// TimeSinceResetResponse is the pump's clock, the time the user set, in pump
// epoch seconds (CurrentTime, and CurrentTimeInstant as a time), and how
// many seconds the pump has run since it was last reset, the clock signed
// messages carry.
type TimeSinceResetResponse struct {
	messages.Base
	CurrentTime        int64     `pumpx2:"currentTime"`
	CurrentTimeInstant time.Time `pumpx2:"currentTimeInstant"`
	PumpTimeSinceReset int64     `pumpx2:"pumpTimeSinceReset"`
}

var timeSinceResetResponseProps = messages.Props{OpCode: 55, Size: 8, Type: messages.Response, Request: "TimeSinceResetRequest"}

func init() {
	messages.Register("TimeSinceResetResponse", &TimeSinceResetResponse{}, timeSinceResetResponseProps,
		messages.Constructor{Params: "long currentTime, long pumpTimeSinceReset", Func: NewTimeSinceResetResponse},
	)
}

// NewTimeSinceResetResponse builds the response. It fails for a currentTime
// beyond the range of a Java Instant.
func NewTimeSinceResetResponse(currentTime, pumpTimeSinceReset int64) (*TimeSinceResetResponse, error) {
	instant, err := helpers.FromJan12008EpochSecondsToDate(currentTime)
	if err != nil {
		return nil, messages.Invalid("TimeSinceResetResponse: %v", err)
	}
	m := &TimeSinceResetResponse{CurrentTime: currentTime, CurrentTimeInstant: instant, PumpTimeSinceReset: pumpTimeSinceReset}
	m.SetCargo(helpers.Combine(helpers.ToUint32(currentTime), helpers.ToUint32(pumpTimeSinceReset)))
	return m, nil
}

// Parse implements messages.Message.
func (m *TimeSinceResetResponse) Parse(raw []byte) error {
	if len(raw) != timeSinceResetResponseProps.Size {
		return messages.InvalidLength("TimeSinceResetResponse", len(raw), timeSinceResetResponseProps.Size)
	}
	currentTime := helpers.ReadUint32(raw, 0)
	instant, err := helpers.FromJan12008EpochSecondsToDate(currentTime)
	if err != nil {
		return messages.Invalid("TimeSinceResetResponse: %v", err)
	}
	m.SetCargo(raw)
	m.CurrentTime = currentTime
	m.CurrentTimeInstant = instant
	m.PumpTimeSinceReset = helpers.ReadUint32(raw, 4)
	return nil
}
