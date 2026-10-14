package currentstatus

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CGMStatusCResponse is a Mobi's CGM session, laid out as CgmStatusV2Response
// with the sensor's warmup appended: how long it lasts and how much is left.
// Timestamps are pump epoch seconds and durations seconds.
type CGMStatusCResponse struct {
	messages.Base
	SessionStateID                 int32                              `pumpx2:"sessionStateId"`
	SessionState                   CGMStatusCSessionState             `pumpx2:"sessionState"`
	LastCalibrationTimestamp       int64                              `pumpx2:"lastCalibrationTimestamp"`
	SensorStartedTimestamp         int64                              `pumpx2:"sensorStartedTimestamp"`
	TransmitterBatteryStatusID     int32                              `pumpx2:"transmitterBatteryStatusId"`
	TransmitterBatteryStatus       CGMStatusCTransmitterBatteryStatus `pumpx2:"transmitterBatteryStatus"`
	SessionDurationSeconds         int64                              `pumpx2:"sessionDurationSeconds"`
	SessionTimeRemainingSeconds    int64                              `pumpx2:"sessionTimeRemainingSeconds"`
	CgmSensorTypeID                int32                              `pumpx2:"cgmSensorTypeId"`
	CgmSensorType                  CGMStatusCSensorType               `pumpx2:"cgmSensorType"`
	GracePeriod                    bool                               `pumpx2:"gracePeriod"`
	SensorWarmupDurationSeconds    int32                              `pumpx2:"sensorWarmupDurationSeconds"`
	RemainingWarmupDurationSeconds int32                              `pumpx2:"remainingWarmupDurationSeconds"`
}

var cgmStatusCResponseProps = messages.Props{OpCode: -51, Size: 24, Type: messages.Response, MinAPI: "MOBI_API_V3_8", Request: "CGMStatusCRequest"}

func init() {
	messages.Register("CGMStatusCResponse", &CGMStatusCResponse{}, cgmStatusCResponseProps,
		messages.Constructor{
			Params: "int sessionStateId, long lastCalibrationTimestamp, long sensorStartedTimestamp, int transmitterBatteryStatusId, " +
				"long sessionDurationSeconds, long sessionTimeRemainingSeconds, int cgmSensorTypeId, boolean gracePeriod, " +
				"int sensorWarmupDurationSeconds, int remainingWarmupDurationSeconds",
			Func: NewCGMStatusCResponse,
		},
	)
}

// NewCGMStatusCResponse builds the response. Each enum is given by its wire
// ID, and reads back as UNKNOWN when it is not one.
func NewCGMStatusCResponse(sessionStateID int32, lastCalibrationTimestamp, sensorStartedTimestamp int64, transmitterBatteryStatusID int32,
	sessionDurationSeconds, sessionTimeRemainingSeconds int64, cgmSensorTypeID int32, gracePeriod bool,
	sensorWarmupDurationSeconds, remainingWarmupDurationSeconds int32) *CGMStatusCResponse {
	m := &CGMStatusCResponse{}
	m.decode(helpers.Combine(
		[]byte{byte(sessionStateID)},
		helpers.ToUint32(lastCalibrationTimestamp),
		helpers.ToUint32(sensorStartedTimestamp),
		[]byte{byte(transmitterBatteryStatusID)},
		helpers.ToUint32(sessionDurationSeconds),
		helpers.ToUint32(sessionTimeRemainingSeconds),
		[]byte{byte(cgmSensorTypeID), helpers.Bool(gracePeriod)},
		helpers.FirstTwoBytesLittleEndian(sensorWarmupDurationSeconds),
		helpers.FirstTwoBytesLittleEndian(remainingWarmupDurationSeconds),
	))
	return m
}

// Parse implements messages.Message.
func (m *CGMStatusCResponse) Parse(raw []byte) error {
	if len(raw) != cgmStatusCResponseProps.Size {
		return messages.InvalidLength("CGMStatusCResponse", len(raw), cgmStatusCResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *CGMStatusCResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.SessionStateID = int32(raw[0])
	m.SessionState = CGMStatusCSessionStateFromID(m.SessionStateID)
	m.LastCalibrationTimestamp = helpers.ReadUint32(raw, 1)
	m.SensorStartedTimestamp = helpers.ReadUint32(raw, 5)
	m.TransmitterBatteryStatusID = int32(raw[9])
	m.TransmitterBatteryStatus = CGMStatusCTransmitterBatteryStatusFromID(m.TransmitterBatteryStatusID)
	m.SessionDurationSeconds = helpers.ReadUint32(raw, 10)
	m.SessionTimeRemainingSeconds = helpers.ReadUint32(raw, 14)
	m.CgmSensorTypeID = int32(raw[18])
	m.CgmSensorType = CGMStatusCSensorTypeFromID(m.CgmSensorTypeID)
	m.GracePeriod = raw[19] != 0
	m.SensorWarmupDurationSeconds = helpers.ReadShort(raw, 20)
	m.RemainingWarmupDurationSeconds = helpers.ReadShort(raw, 22)
}

// LastCalibrationTime is LastCalibrationTimestamp as pumpX2 reads it: the
// pump's clock, as UTC.
func (m *CGMStatusCResponse) LastCalibrationTime() time.Time {
	return time.Unix(helpers.FromJan12008ToUnixEpochSeconds(m.LastCalibrationTimestamp), 0).UTC()
}

// SensorStartedTime is SensorStartedTimestamp as pumpX2 reads it: the pump's
// clock, as UTC.
func (m *CGMStatusCResponse) SensorStartedTime() time.Time {
	return time.Unix(helpers.FromJan12008ToUnixEpochSeconds(m.SensorStartedTimestamp), 0).UTC()
}

// CGMStatusCSessionState is a CGM session's state, numbered as the Tandem
// app's CGMStatusBState.
type CGMStatusCSessionState int32

// The session states.
const (
	CGMStatusCSessionStateStatusStopped CGMStatusCSessionState = iota
	CGMStatusCSessionStateStatusStartPending
	CGMStatusCSessionStateSessionActive
	CGMStatusCSessionStateSessionStopPending
	CGMStatusCSessionStateUnknown
)

var cgmStatusCSessionStateNames = []string{"STATUS_STOPPED", "STATUS_START_PENDING", "SESSION_ACTIVE", "SESSION_STOP_PENDING", "UNKNOWN"}

// CGMStatusCSessionStateFromID is the state with wire ID id, or Unknown.
func CGMStatusCSessionStateFromID(id int32) CGMStatusCSessionState {
	if id >= 0 && int(id) < len(cgmStatusCSessionStateNames) {
		return CGMStatusCSessionState(id)
	}
	return CGMStatusCSessionStateUnknown
}

func (s CGMStatusCSessionState) String() string {
	if s >= 0 && int(s) < len(cgmStatusCSessionStateNames) {
		return cgmStatusCSessionStateNames[s]
	}
	return fmt.Sprintf("CGMStatusCSessionState(%d)", int32(s))
}

// CGMStatusCTransmitterBatteryStatus is the CGM transmitter's battery, as
// the Tandem app numbers it: OKAY where CGMStatusResponse says OK.
type CGMStatusCTransmitterBatteryStatus int32

// The transmitter battery statuses.
const (
	CGMStatusCTransmitterBatteryStatusUnavailable CGMStatusCTransmitterBatteryStatus = iota
	CGMStatusCTransmitterBatteryStatusError
	CGMStatusCTransmitterBatteryStatusExpired
	CGMStatusCTransmitterBatteryStatusOkay
	CGMStatusCTransmitterBatteryStatusOutOfRange
	CGMStatusCTransmitterBatteryStatusUnknown
)

var cgmStatusCTransmitterBatteryStatusNames = []string{"UNAVAILABLE", "ERROR", "EXPIRED", "OKAY", "OUT_OF_RANGE", "UNKNOWN"}

// CGMStatusCTransmitterBatteryStatusFromID is the status with wire ID id, or
// Unknown.
func CGMStatusCTransmitterBatteryStatusFromID(id int32) CGMStatusCTransmitterBatteryStatus {
	if id >= 0 && int(id) < len(cgmStatusCTransmitterBatteryStatusNames) {
		return CGMStatusCTransmitterBatteryStatus(id)
	}
	return CGMStatusCTransmitterBatteryStatusUnknown
}

func (s CGMStatusCTransmitterBatteryStatus) String() string {
	if s >= 0 && int(s) < len(cgmStatusCTransmitterBatteryStatusNames) {
		return cgmStatusCTransmitterBatteryStatusNames[s]
	}
	return fmt.Sprintf("CGMStatusCTransmitterBatteryStatus(%d)", int32(s))
}

// CGMStatusCSensorType is the kind of CGM sensor, numbered as the Tandem
// app's CGMSensorType.
type CGMStatusCSensorType int32

// The sensor types.
const (
	CGMStatusCSensorTypeNotApplicable CGMStatusCSensorType = iota
	CGMStatusCSensorTypeDexcomG6
	CGMStatusCSensorTypeFSL2
	CGMStatusCSensorTypeDexcomG7
	CGMStatusCSensorTypeUnknown
)

var cgmStatusCSensorTypeNames = []string{"NOT_APPLICABLE", "DEXCOM_G6", "FSL2", "DEXCOM_G7", "UNKNOWN"}

// CGMStatusCSensorTypeFromID is the sensor type with wire ID id, or Unknown.
func CGMStatusCSensorTypeFromID(id int32) CGMStatusCSensorType {
	if id >= 0 && int(id) < len(cgmStatusCSensorTypeNames) {
		return CGMStatusCSensorType(id)
	}
	return CGMStatusCSensorTypeUnknown
}

func (s CGMStatusCSensorType) String() string {
	if s >= 0 && int(s) < len(cgmStatusCSensorTypeNames) {
		return cgmStatusCSensorTypeNames[s]
	}
	return fmt.Sprintf("CGMStatusCSensorType(%d)", int32(s))
}
