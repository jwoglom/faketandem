package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CurrentEGVGuiDataResponse is the latest CGM reading as the pump's screen
// shows it: when it was taken (pump epoch seconds), the reading (mg/dL), its
// status (an EGVStatus ID) and the trend rate.
type CurrentEGVGuiDataResponse struct {
	messages.Base
	BgReadingTimestampSeconds int64      `pumpx2:"bgReadingTimestampSeconds"`
	CgmReading                int32      `pumpx2:"cgmReading"`
	EgvStatusID               int32      `pumpx2:"egvStatusId"`
	EgvStatus                 *EGVStatus `pumpx2:"egvStatus"`
	TrendRate                 int32      `pumpx2:"trendRate"`
}

var currentEGVGuiDataResponseProps = messages.Props{OpCode: 35, Size: 8, Type: messages.Response, Request: "CurrentEGVGuiDataRequest"}

func init() {
	messages.Register("CurrentEGVGuiDataResponse", &CurrentEGVGuiDataResponse{}, currentEGVGuiDataResponseProps,
		messages.Constructor{Params: "long bgReadingTimestampSeconds, int cgmReading, int egvStatusId, int trendRate", Func: NewCurrentEGVGuiDataResponse},
	)
}

// NewCurrentEGVGuiDataResponse builds the response.
func NewCurrentEGVGuiDataResponse(bgReadingTimestampSeconds int64, cgmReading, egvStatusID, trendRate int32) *CurrentEGVGuiDataResponse {
	m := &CurrentEGVGuiDataResponse{
		BgReadingTimestampSeconds: bgReadingTimestampSeconds,
		CgmReading:                cgmReading,
		EgvStatusID:               egvStatusID,
		EgvStatus:                 EGVStatusFromID(egvStatusID),
		TrendRate:                 trendRate,
	}
	m.SetCargo(egvGuiDataCargo(bgReadingTimestampSeconds, cgmReading, egvStatusID, trendRate))
	return m
}

// egvGuiDataCargo is the layout both CurrentEGVGuiDataResponse and
// CurrentEgvGuiDataV2Response have.
func egvGuiDataCargo(bgReadingTimestampSeconds int64, cgmReading, egvStatusID, trendRate int32) []byte {
	return helpers.Combine(
		helpers.ToUint32(bgReadingTimestampSeconds),
		helpers.FirstTwoBytesLittleEndian(cgmReading),
		[]byte{byte(egvStatusID), byte(trendRate)},
	)
}

// Parse implements messages.Message.
func (m *CurrentEGVGuiDataResponse) Parse(raw []byte) error {
	if len(raw) != currentEGVGuiDataResponseProps.Size {
		return messages.InvalidLength("CurrentEGVGuiDataResponse", len(raw), currentEGVGuiDataResponseProps.Size)
	}
	m.SetCargo(raw)
	m.BgReadingTimestampSeconds = helpers.ReadUint32(raw, 0)
	m.CgmReading = helpers.ReadShort(raw, 4)
	m.EgvStatusID = int32(int8(raw[6]))
	m.EgvStatus = EGVStatusFromID(m.EgvStatusID)
	m.TrendRate = int32(int8(raw[7]))
	return nil
}

// EGVStatus is the status of a CGM reading (an estimated glucose value).
type EGVStatus int32

// The reading statuses.
const (
	EGVStatusInvalid EGVStatus = iota
	EGVStatusValid
	EGVStatusLow
	EGVStatusHigh
	EGVStatusUnavailable
)

var egvStatusNames = []string{"INVALID", "VALID", "LOW", "HIGH", "UNAVAILABLE"}

// EGVStatusFromID is the status with wire ID id, or nil for an ID that is
// none.
func EGVStatusFromID(id int32) *EGVStatus {
	if id < 0 || int(id) >= len(egvStatusNames) {
		return nil
	}
	s := EGVStatus(id)
	return &s
}

// ID is the status's wire ID.
func (s EGVStatus) ID() int32 { return int32(s) }

func (s EGVStatus) String() string {
	if s >= 0 && int(s) < len(egvStatusNames) {
		return egvStatusNames[s]
	}
	return fmt.Sprintf("EGVStatus(%d)", int32(s))
}
