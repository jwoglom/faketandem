package currentstatus

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CGMStatusResponse is the CGM session: its state, when the sensor was last
// calibrated and when it started (pump epoch seconds), and the transmitter
// battery. A Mobi reports more in CgmStatusV2Response and CGMStatusCResponse.
type CGMStatusResponse struct {
	messages.Base
	SessionStateID             int32                              `pumpx2:"sessionStateId"`
	SessionState               *CGMStatusSessionState             `pumpx2:"sessionState"`
	LastCalibrationTimestamp   int64                              `pumpx2:"lastCalibrationTimestamp"`
	SensorStartedTimestamp     int64                              `pumpx2:"sensorStartedTimestamp"`
	TransmitterBatteryStatusID int32                              `pumpx2:"transmitterBatteryStatusId"`
	TransmitterBatteryStatus   *CGMStatusTransmitterBatteryStatus `pumpx2:"transmitterBatteryStatus"`
}

var cgmStatusResponseProps = messages.Props{OpCode: 81, Size: 10, Type: messages.Response, Request: "CGMStatusRequest"}

func init() {
	messages.Register("CGMStatusResponse", &CGMStatusResponse{}, cgmStatusResponseProps,
		messages.Constructor{
			Params: "int sessionStateId, long lastCalibrationTimestamp, long sensorStartedTimestamp, int transmitterBatteryStatusId",
			Func:   NewCGMStatusResponse,
		},
	)
}

// NewCGMStatusResponse builds the response. Each enum is given by its wire
// ID, and reads back as nil when it is not one.
func NewCGMStatusResponse(sessionStateID int32, lastCalibrationTimestamp, sensorStartedTimestamp int64, transmitterBatteryStatusID int32) *CGMStatusResponse {
	m := &CGMStatusResponse{}
	m.decode(helpers.Combine(
		[]byte{byte(sessionStateID)},
		helpers.ToUint32(lastCalibrationTimestamp),
		helpers.ToUint32(sensorStartedTimestamp),
		[]byte{byte(transmitterBatteryStatusID)},
	))
	return m
}

// Parse implements messages.Message.
func (m *CGMStatusResponse) Parse(raw []byte) error {
	if len(raw) != cgmStatusResponseProps.Size {
		return messages.InvalidLength("CGMStatusResponse", len(raw), cgmStatusResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *CGMStatusResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.SessionStateID = int32(int8(raw[0]))
	m.SessionState = CGMStatusSessionStateFromID(m.SessionStateID)
	m.LastCalibrationTimestamp = helpers.ReadUint32(raw, 1)
	m.SensorStartedTimestamp = helpers.ReadUint32(raw, 5)
	m.TransmitterBatteryStatusID = int32(int8(raw[9]))
	m.TransmitterBatteryStatus = CGMStatusTransmitterBatteryStatusFromID(m.TransmitterBatteryStatusID)
}

// LastCalibrationTime is LastCalibrationTimestamp as pumpX2 reads it: the
// pump's clock, as UTC.
func (m *CGMStatusResponse) LastCalibrationTime() time.Time {
	return time.Unix(helpers.FromJan12008ToUnixEpochSeconds(m.LastCalibrationTimestamp), 0).UTC()
}

// SensorStartedTime is SensorStartedTimestamp as pumpX2 reads it: the pump's
// clock, as UTC.
func (m *CGMStatusResponse) SensorStartedTime() time.Time {
	return time.Unix(helpers.FromJan12008ToUnixEpochSeconds(m.SensorStartedTimestamp), 0).UTC()
}

// CGMStatusSessionState is a CGM session's state.
type CGMStatusSessionState int32

// The session states.
const (
	CGMStatusSessionStateSessionStopped CGMStatusSessionState = iota
	CGMStatusSessionStateSessionStartPending
	CGMStatusSessionStateSessionActive
	CGMStatusSessionStateSessionStopPending
)

var cgmStatusSessionStateNames = []string{"SESSION_STOPPED", "SESSION_START_PENDING", "SESSION_ACTIVE", "SESSION_STOP_PENDING"}

// CGMStatusSessionStateFromID is the state with wire ID id, or nil.
func CGMStatusSessionStateFromID(id int32) *CGMStatusSessionState {
	if id < 0 || int(id) >= len(cgmStatusSessionStateNames) {
		return nil
	}
	s := CGMStatusSessionState(id)
	return &s
}

func (s CGMStatusSessionState) String() string {
	if s >= 0 && int(s) < len(cgmStatusSessionStateNames) {
		return cgmStatusSessionStateNames[s]
	}
	return fmt.Sprintf("CGMStatusSessionState(%d)", int32(s))
}

// CGMStatusTransmitterBatteryStatus is the CGM transmitter's battery.
type CGMStatusTransmitterBatteryStatus int32

// The transmitter battery statuses.
const (
	CGMStatusTransmitterBatteryStatusUnavailable CGMStatusTransmitterBatteryStatus = iota
	CGMStatusTransmitterBatteryStatusError
	CGMStatusTransmitterBatteryStatusExpired
	CGMStatusTransmitterBatteryStatusOK
	CGMStatusTransmitterBatteryStatusOutOfRange
)

var cgmStatusTransmitterBatteryStatusNames = []string{"UNAVAILABLE", "ERROR", "EXPIRED", "OK", "OUT_OF_RANGE"}

// CGMStatusTransmitterBatteryStatusFromID is the status with wire ID id, or
// nil.
func CGMStatusTransmitterBatteryStatusFromID(id int32) *CGMStatusTransmitterBatteryStatus {
	if id < 0 || int(id) >= len(cgmStatusTransmitterBatteryStatusNames) {
		return nil
	}
	s := CGMStatusTransmitterBatteryStatus(id)
	return &s
}

func (s CGMStatusTransmitterBatteryStatus) String() string {
	if s >= 0 && int(s) < len(cgmStatusTransmitterBatteryStatusNames) {
		return cgmStatusTransmitterBatteryStatusNames[s]
	}
	return fmt.Sprintf("CGMStatusTransmitterBatteryStatus(%d)", int32(s))
}
