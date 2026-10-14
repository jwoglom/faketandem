package currentstatus

import (
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// TempRateResponse is the temp basal rate in force: whether one is, its
// percentage of the profile rate, when it started (pump epoch seconds) and
// its duration, in a unit pumpX2 does not document.
type TempRateResponse struct {
	messages.Base
	Active       bool  `pumpx2:"active"`
	Percentage   int32 `pumpx2:"percentage"`
	StartTimeRaw int64 `pumpx2:"startTimeRaw"`
	Duration     int64 `pumpx2:"duration"`
}

var tempRateResponseProps = messages.Props{OpCode: 43, Size: 10, Type: messages.Response, Request: "TempRateRequest"}

func init() {
	messages.Register("TempRateResponse", &TempRateResponse{}, tempRateResponseProps,
		messages.Constructor{Params: "boolean active, int percentage, long startTimeRaw, long duration", Func: NewTempRateResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewTempRateResponseFromCargo},
	)
}

// NewTempRateResponse builds the response.
func NewTempRateResponse(active bool, percentage int32, startTimeRaw, duration int64) *TempRateResponse {
	m := &TempRateResponse{Active: active, Percentage: percentage, StartTimeRaw: startTimeRaw, Duration: duration}
	m.SetCargo(helpers.Combine([]byte{helpers.Bool(active), byte(percentage)}, helpers.ToUint32(startTimeRaw), helpers.ToUint32(duration)))
	return m
}

// NewTempRateResponseFromCargo parses a response's payload.
func NewTempRateResponseFromCargo(raw []byte) (*TempRateResponse, error) {
	m := &TempRateResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *TempRateResponse) Parse(raw []byte) error {
	if len(raw) != tempRateResponseProps.Size {
		return messages.InvalidLength("TempRateResponse", len(raw), tempRateResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Active = raw[0] != 0
	m.Percentage = int32(raw[1])
	m.StartTimeRaw = helpers.ReadUint32(raw, 2)
	m.Duration = helpers.ReadUint32(raw, 6)
	return nil
}

// StartTimeInstant is when the temp rate started. It fails only for a
// StartTimeRaw beyond the range of a Java Instant.
func (m *TempRateResponse) StartTimeInstant() (time.Time, error) {
	return helpers.FromJan12008EpochSecondsToDate(m.StartTimeRaw)
}
