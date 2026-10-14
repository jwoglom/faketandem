package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CgmStatusV2Response is a Mobi's CGM session: its state, when the sensor
// started and was last calibrated (pump epoch seconds), how long it has run
// and has left, the transmitter battery, the sensor type and whether the
// session is in its grace period.
type CgmStatusV2Response struct {
	messages.Base
	SessionStateID              int32                               `pumpx2:"sessionStateId"`
	SessionState                CgmStatusV2SessionState             `pumpx2:"sessionState"`
	LastCalibrationTimestamp    int64                               `pumpx2:"lastCalibrationTimestamp"`
	SensorStartedTimestamp      int64                               `pumpx2:"sensorStartedTimestamp"`
	TransmitterBatteryStatusID  int32                               `pumpx2:"transmitterBatteryStatusId"`
	TransmitterBatteryStatus    CgmStatusV2TransmitterBatteryStatus `pumpx2:"transmitterBatteryStatus"`
	SessionDurationSeconds      int64                               `pumpx2:"sessionDurationSeconds"`
	SessionTimeRemainingSeconds int64                               `pumpx2:"sessionTimeRemainingSeconds"`
	CgmSensorTypeID             int32                               `pumpx2:"cgmSensorTypeId"`
	CgmSensorType               CgmStatusV2SensorType               `pumpx2:"cgmSensorType"`
	GracePeriod                 bool                                `pumpx2:"gracePeriod"`
}

var cgmStatusV2ResponseProps = messages.Props{OpCode: -65, Size: 20, Type: messages.Response, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Request: "CgmStatusV2Request"}

func init() {
	messages.Register("CgmStatusV2Response", &CgmStatusV2Response{}, cgmStatusV2ResponseProps,
		messages.Constructor{
			Params: "int sessionStateId, long lastCalibrationTimestamp, long sensorStartedTimestamp, int transmitterBatteryStatusId, " +
				"long sessionDurationSeconds, long sessionTimeRemainingSeconds, int cgmSensorTypeId, boolean gracePeriod",
			Func: NewCgmStatusV2Response,
		},
	)
}

// NewCgmStatusV2Response builds the response. Each enum is given by its wire
// ID, and reads back as UNKNOWN when it is not one.
func NewCgmStatusV2Response(sessionStateID int32, lastCalibrationTimestamp, sensorStartedTimestamp int64, transmitterBatteryStatusID int32,
	sessionDurationSeconds, sessionTimeRemainingSeconds int64, cgmSensorTypeID int32, gracePeriod bool) *CgmStatusV2Response {
	m := &CgmStatusV2Response{}
	m.decode(helpers.Combine(
		[]byte{byte(sessionStateID)},
		helpers.ToUint32(lastCalibrationTimestamp),
		helpers.ToUint32(sensorStartedTimestamp),
		[]byte{byte(transmitterBatteryStatusID)},
		helpers.ToUint32(sessionDurationSeconds),
		helpers.ToUint32(sessionTimeRemainingSeconds),
		[]byte{byte(cgmSensorTypeID)},
		[]byte{helpers.Bool(gracePeriod)},
	))
	return m
}

// Parse implements messages.Message.
func (m *CgmStatusV2Response) Parse(raw []byte) error {
	if len(raw) != cgmStatusV2ResponseProps.Size {
		return messages.InvalidLength("CgmStatusV2Response", len(raw), cgmStatusV2ResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *CgmStatusV2Response) decode(raw []byte) {
	m.SetCargo(raw)
	m.SessionStateID = int32(raw[0])
	m.SessionState = CgmStatusV2SessionStateFromID(m.SessionStateID)
	m.LastCalibrationTimestamp = helpers.ReadUint32(raw, 1)
	m.SensorStartedTimestamp = helpers.ReadUint32(raw, 5)
	m.TransmitterBatteryStatusID = int32(raw[9])
	m.TransmitterBatteryStatus = CgmStatusV2TransmitterBatteryStatusFromID(m.TransmitterBatteryStatusID)
	m.SessionDurationSeconds = helpers.ReadUint32(raw, 10)
	m.SessionTimeRemainingSeconds = helpers.ReadUint32(raw, 14)
	m.CgmSensorTypeID = int32(raw[18])
	m.CgmSensorType = CgmStatusV2SensorTypeFromID(m.CgmSensorTypeID)
	m.GracePeriod = raw[19] != 0
}

// CgmStatusV2SessionState is a CGM session's state, numbered as the Tandem
// app's CGMStatusBState.
type CgmStatusV2SessionState int32

// The session states.
const (
	CgmStatusV2SessionStateStatusStopped CgmStatusV2SessionState = iota
	CgmStatusV2SessionStateStatusStartPending
	CgmStatusV2SessionStateSessionActive
	CgmStatusV2SessionStateSessionStopPending
	CgmStatusV2SessionStateUnknown
)

var cgmStatusV2SessionStateNames = []string{"STATUS_STOPPED", "STATUS_START_PENDING", "SESSION_ACTIVE", "SESSION_STOP_PENDING", "UNKNOWN"}

// CgmStatusV2SessionStateFromID is the state with wire ID id, or Unknown.
func CgmStatusV2SessionStateFromID(id int32) CgmStatusV2SessionState {
	if id >= 0 && int(id) < len(cgmStatusV2SessionStateNames) {
		return CgmStatusV2SessionState(id)
	}
	return CgmStatusV2SessionStateUnknown
}

func (s CgmStatusV2SessionState) String() string {
	if s >= 0 && int(s) < len(cgmStatusV2SessionStateNames) {
		return cgmStatusV2SessionStateNames[s]
	}
	return fmt.Sprintf("CgmStatusV2SessionState(%d)", int32(s))
}

// CgmStatusV2TransmitterBatteryStatus is the CGM transmitter's battery, as
// the Tandem app numbers it: OKAY where CGMStatusResponse says OK.
type CgmStatusV2TransmitterBatteryStatus int32

// The transmitter battery statuses.
const (
	CgmStatusV2TransmitterBatteryStatusUnavailable CgmStatusV2TransmitterBatteryStatus = iota
	CgmStatusV2TransmitterBatteryStatusError
	CgmStatusV2TransmitterBatteryStatusExpired
	CgmStatusV2TransmitterBatteryStatusOkay
	CgmStatusV2TransmitterBatteryStatusOutOfRange
	CgmStatusV2TransmitterBatteryStatusUnknown
)

var cgmStatusV2TransmitterBatteryStatusNames = []string{"UNAVAILABLE", "ERROR", "EXPIRED", "OKAY", "OUT_OF_RANGE", "UNKNOWN"}

// CgmStatusV2TransmitterBatteryStatusFromID is the status with wire ID id,
// or Unknown.
func CgmStatusV2TransmitterBatteryStatusFromID(id int32) CgmStatusV2TransmitterBatteryStatus {
	if id >= 0 && int(id) < len(cgmStatusV2TransmitterBatteryStatusNames) {
		return CgmStatusV2TransmitterBatteryStatus(id)
	}
	return CgmStatusV2TransmitterBatteryStatusUnknown
}

func (s CgmStatusV2TransmitterBatteryStatus) String() string {
	if s >= 0 && int(s) < len(cgmStatusV2TransmitterBatteryStatusNames) {
		return cgmStatusV2TransmitterBatteryStatusNames[s]
	}
	return fmt.Sprintf("CgmStatusV2TransmitterBatteryStatus(%d)", int32(s))
}

// CgmStatusV2SensorType is the kind of CGM sensor, numbered as the Tandem
// app's CGMSensorType.
type CgmStatusV2SensorType int32

// The sensor types.
const (
	CgmStatusV2SensorTypeNotApplicable CgmStatusV2SensorType = iota
	CgmStatusV2SensorTypeDexcomG6
	CgmStatusV2SensorTypeFSL2
	CgmStatusV2SensorTypeDexcomG7
	CgmStatusV2SensorTypeUnknown
)

var cgmStatusV2SensorTypeNames = []string{"NOT_APPLICABLE", "DEXCOM_G6", "FSL2", "DEXCOM_G7", "UNKNOWN"}

// CgmStatusV2SensorTypeFromID is the sensor type with wire ID id, or
// Unknown.
func CgmStatusV2SensorTypeFromID(id int32) CgmStatusV2SensorType {
	if id >= 0 && int(id) < len(cgmStatusV2SensorTypeNames) {
		return CgmStatusV2SensorType(id)
	}
	return CgmStatusV2SensorTypeUnknown
}

// ID is the sensor type's wire ID.
func (s CgmStatusV2SensorType) ID() int32 { return int32(s) }

func (s CgmStatusV2SensorType) String() string {
	if s >= 0 && int(s) < len(cgmStatusV2SensorTypeNames) {
		return cgmStatusV2SensorTypeNames[s]
	}
	return fmt.Sprintf("CgmStatusV2SensorType(%d)", int32(s))
}
