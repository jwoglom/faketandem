package currentstatus

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// CGMAlertStatusResponse is the pump's active CGM alerts: a 64-bit mask with
// a CGMAlert at each bit. pumpX2 prints none of it but the cargo.
type CGMAlertStatusResponse struct {
	messages.Base
	// IntMap is the mask. Built from a long it keeps that long's sign, as
	// BigInteger.valueOf does; parsed it is unsigned.
	IntMap *big.Int
}

var cgmAlertStatusResponseProps = messages.Props{OpCode: 75, Size: 8, Type: messages.Response, Request: "CGMAlertStatusRequest"}

func init() {
	messages.Register("CGMAlertStatusResponse", &CGMAlertStatusResponse{}, cgmAlertStatusResponseProps,
		messages.Constructor{Params: "long cgmAlertBitmask", Func: NewCGMAlertStatusResponse},
	)
}

// NewCGMAlertStatusResponse is the response with mask cgmAlertBitmask.
func NewCGMAlertStatusResponse(cgmAlertBitmask int64) *CGMAlertStatusResponse {
	m := &CGMAlertStatusResponse{IntMap: big.NewInt(cgmAlertBitmask)}
	m.SetCargo(helpers.ToUint64(cgmAlertBitmask))
	return m
}

// Parse implements messages.Message.
func (m *CGMAlertStatusResponse) Parse(raw []byte) error {
	if len(raw) != cgmAlertStatusResponseProps.Size {
		return messages.InvalidLength("CGMAlertStatusResponse", len(raw), cgmAlertStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.IntMap = helpers.ReadUint64(raw, 0)
	return nil
}

// BitMap is the mask as a signed 64-bit number.
func (m *CGMAlertStatusResponse) BitMap() int64 { return helpers.LongValue(m.IntMap) }

// CGMAlerts are the active CGM alerts, in declaration order.
func (m *CGMAlertStatusResponse) CGMAlerts() []CGMAlert {
	if m.IntMap == nil {
		return []CGMAlert{}
	}
	return CGMAlertFromBitmask(m.IntMap)
}

// Size is how many CGM alerts are active.
func (m *CGMAlertStatusResponse) Size() int { return len(m.CGMAlerts()) }

// NotificationIDs are the active CGM alerts pumpX2 has a name for.
func (m *CGMAlertStatusResponse) NotificationIDs() []int32 {
	ids := []int32{}
	for _, a := range m.CGMAlerts() {
		if a.IsKnown() {
			ids = append(ids, int32(a))
		}
	}
	return ids
}

// CGMAlert is a CGM alert, as its bit in CGMAlertStatusResponse's mask. No
// alert has bit 54.
type CGMAlert int32

// The CGM alerts, by their bits.
const (
	CGMAlertDefaultCGMAlert0                CGMAlert = 0
	CGMAlertFixedLowCGMAlert                CGMAlert = 1
	CGMAlertHighCGMAlert                    CGMAlert = 2
	CGMAlertLowCGMAlert                     CGMAlert = 3
	CGMAlertCalibrationRequestCGMAlert      CGMAlert = 4
	CGMAlertRiseCGMAlert                    CGMAlert = 5
	CGMAlertRapidRiseCGMAlert               CGMAlert = 6
	CGMAlertFallCGMAlert                    CGMAlert = 7
	CGMAlertRapidFallCGMAlert               CGMAlert = 8
	CGMAlertLowCalibrationErrorCGMAlert     CGMAlert = 9
	CGMAlertHighCalibrationErrorCGMAlert    CGMAlert = 10
	CGMAlertSensorFailedCGMAlert            CGMAlert = 11
	CGMAlertSensorExpiringCGMAlert          CGMAlert = 12
	CGMAlertSensorExpiredCGMAlert           CGMAlert = 13
	CGMAlertOutOfRangeCGMAlert              CGMAlert = 14
	CGMAlertDefaultCGMAlert15               CGMAlert = 15
	CGMAlertFirstStartCalibrationCGMAlert   CGMAlert = 16
	CGMAlertSecondStartCalibrationCGMAlert  CGMAlert = 17
	CGMAlertCalibrationRequiredCGMAlert     CGMAlert = 18
	CGMAlertLowTransmitterCGMAlert          CGMAlert = 19
	CGMAlertTransmitterCGMAlert             CGMAlert = 20
	CGMAlertDefaultCGMAlert21               CGMAlert = 21
	CGMAlertSensorExpiringCGMAlert2         CGMAlert = 22
	CGMAlertDefaultCGMAlert23               CGMAlert = 23
	CGMAlertUrgentLowSoonCGMAlert           CGMAlert = 24
	CGMAlertSensorReuse                     CGMAlert = 25
	CGMAlertTemperatureCGMAlert             CGMAlert = 26
	CGMAlertFailedConnectionCGMAlert        CGMAlert = 27
	CGMAlertCheckSensorCGMAlert             CGMAlert = 28
	CGMAlertEgvUnusableCGMAlert             CGMAlert = 29
	CGMAlertDefaultCGMAlert30               CGMAlert = 30
	CGMAlertGracePeriodStartCGMAlert        CGMAlert = 31
	CGMAlertGracePeriodExpiringSoonCGMAlert CGMAlert = 32
	CGMAlertDefaultCGMAlert33               CGMAlert = 33
	CGMAlertDefaultCGMAlert34               CGMAlert = 34
	CGMAlertDefaultCGMAlert35               CGMAlert = 35
	CGMAlertDefaultCGMAlert36               CGMAlert = 36
	CGMAlertDefaultCGMAlert37               CGMAlert = 37
	CGMAlertDefaultCGMAlert38               CGMAlert = 38
	CGMAlertTransmitterExpiredCGMAlert      CGMAlert = 39
	CGMAlertPumpBluetoothErrorCGMAlert      CGMAlert = 40
	CGMAlertDefaultCGMAlert41               CGMAlert = 41
	CGMAlertDefaultCGMAlert42               CGMAlert = 42
	CGMAlertDefaultCGMAlert43               CGMAlert = 43
	CGMAlertDefaultCGMAlert44               CGMAlert = 44
	CGMAlertTransmitterExpiringCGMAlert     CGMAlert = 45
	CGMAlertTransmitterExpiringCGMAlert2    CGMAlert = 46
	CGMAlertDefaultCGMAlert47               CGMAlert = 47
	CGMAlertUnavailableCGMAlert             CGMAlert = 48
	CGMAlertDefaultCGMAlert49               CGMAlert = 49
	CGMAlertDefaultCGMAlert50               CGMAlert = 50
	CGMAlertDefaultCGMAlert51               CGMAlert = 51
	CGMAlertDefaultCGMAlert52               CGMAlert = 52
	CGMAlertDefaultCGMAlert53               CGMAlert = 53
	CGMAlertDefaultCGMAlert55               CGMAlert = 55
	CGMAlertDefaultCGMAlert56               CGMAlert = 56
	CGMAlertDefaultCGMAlert57               CGMAlert = 57
	CGMAlertDefaultCGMAlert58               CGMAlert = 58
	CGMAlertDefaultCGMAlert59               CGMAlert = 59
	CGMAlertDefaultCGMAlert60               CGMAlert = 60
	CGMAlertDefaultCGMAlert61               CGMAlert = 61
	CGMAlertDefaultCGMAlert62               CGMAlert = 62
	CGMAlertDefaultCGMAlert63               CGMAlert = 63
)

// CGMAlerts are the CGM alerts in pumpX2's declaration order.
var CGMAlerts = []CGMAlert{
	CGMAlertDefaultCGMAlert0, CGMAlertFixedLowCGMAlert, CGMAlertHighCGMAlert, CGMAlertLowCGMAlert,
	CGMAlertCalibrationRequestCGMAlert, CGMAlertRiseCGMAlert, CGMAlertRapidRiseCGMAlert, CGMAlertFallCGMAlert,
	CGMAlertRapidFallCGMAlert, CGMAlertLowCalibrationErrorCGMAlert, CGMAlertHighCalibrationErrorCGMAlert,
	CGMAlertSensorFailedCGMAlert, CGMAlertSensorExpiringCGMAlert, CGMAlertSensorExpiredCGMAlert,
	CGMAlertOutOfRangeCGMAlert, CGMAlertDefaultCGMAlert15, CGMAlertFirstStartCalibrationCGMAlert,
	CGMAlertSecondStartCalibrationCGMAlert, CGMAlertCalibrationRequiredCGMAlert, CGMAlertLowTransmitterCGMAlert,
	CGMAlertTransmitterCGMAlert, CGMAlertDefaultCGMAlert21, CGMAlertSensorExpiringCGMAlert2, CGMAlertDefaultCGMAlert23,
	CGMAlertUrgentLowSoonCGMAlert, CGMAlertSensorReuse, CGMAlertTemperatureCGMAlert, CGMAlertFailedConnectionCGMAlert,
	CGMAlertCheckSensorCGMAlert, CGMAlertEgvUnusableCGMAlert, CGMAlertDefaultCGMAlert30, CGMAlertGracePeriodStartCGMAlert,
	CGMAlertGracePeriodExpiringSoonCGMAlert, CGMAlertDefaultCGMAlert33, CGMAlertDefaultCGMAlert34,
	CGMAlertDefaultCGMAlert35, CGMAlertDefaultCGMAlert36, CGMAlertDefaultCGMAlert37, CGMAlertDefaultCGMAlert38,
	CGMAlertTransmitterExpiredCGMAlert, CGMAlertPumpBluetoothErrorCGMAlert, CGMAlertDefaultCGMAlert41,
	CGMAlertDefaultCGMAlert42, CGMAlertDefaultCGMAlert43, CGMAlertDefaultCGMAlert44, CGMAlertTransmitterExpiringCGMAlert,
	CGMAlertTransmitterExpiringCGMAlert2, CGMAlertDefaultCGMAlert47, CGMAlertUnavailableCGMAlert,
	CGMAlertDefaultCGMAlert49, CGMAlertDefaultCGMAlert50, CGMAlertDefaultCGMAlert51, CGMAlertDefaultCGMAlert52,
	CGMAlertDefaultCGMAlert53, CGMAlertDefaultCGMAlert55, CGMAlertDefaultCGMAlert56, CGMAlertDefaultCGMAlert57,
	CGMAlertDefaultCGMAlert58, CGMAlertDefaultCGMAlert59, CGMAlertDefaultCGMAlert60, CGMAlertDefaultCGMAlert61,
	CGMAlertDefaultCGMAlert62, CGMAlertDefaultCGMAlert63,
}

var cgmAlertNames = [64]string{
	0:  "DEFAULT_CGM_ALERT_0",
	1:  "FIXED_LOW_CGM_ALERT",
	2:  "HIGH_CGM_ALERT",
	3:  "LOW_CGM_ALERT",
	4:  "CALIBRATION_REQUEST_CGM_ALERT",
	5:  "RISE_CGM_ALERT",
	6:  "RAPID_RISE_CGM_ALERT",
	7:  "FALL_CGM_ALERT",
	8:  "RAPID_FALL_CGM_ALERT",
	9:  "LOW_CALIBRATION_ERROR_CGM_ALERT",
	10: "HIGH_CALIBRATION_ERROR_CGM_ALERT",
	11: "SENSOR_FAILED_CGM_ALERT",
	12: "SENSOR_EXPIRING_CGM_ALERT",
	13: "SENSOR_EXPIRED_CGM_ALERT",
	14: "OUT_OF_RANGE_CGM_ALERT",
	15: "DEFAULT_CGM_ALERT_15",
	16: "FIRST_START_CALIBRATION_CGM_ALERT",
	17: "SECOND_START_CALIBRATION_CGM_ALERT",
	18: "CALIBRATION_REQUIRED_CGM_ALERT",
	19: "LOW_TRANSMITTER_CGM_ALERT",
	20: "TRANSMITTER_CGM_ALERT",
	21: "DEFAULT_CGM_ALERT_21",
	22: "SENSOR_EXPIRING_CGM_ALERT2",
	23: "DEFAULT_CGM_ALERT_23",
	24: "URGENT_LOW_SOON_CGM_ALERT",
	25: "SENSOR_REUSE",
	26: "TEMPERATURE_CGM_ALERT",
	27: "FAILED_CONNECTION_CGM_ALERT",
	28: "CHECK_SENSOR_CGM_ALERT",
	29: "EGV_UNUSABLE_CGM_ALERT",
	30: "DEFAULT_CGM_ALERT_30",
	31: "GRACE_PERIOD_START_CGM_ALERT",
	32: "GRACE_PERIOD_EXPIRING_SOON_CGM_ALERT",
	33: "DEFAULT_CGM_ALERT_33",
	34: "DEFAULT_CGM_ALERT_34",
	35: "DEFAULT_CGM_ALERT_35",
	36: "DEFAULT_CGM_ALERT_36",
	37: "DEFAULT_CGM_ALERT_37",
	38: "DEFAULT_CGM_ALERT_38",
	39: "TRANSMITTER_EXPIRED_CGM_ALERT",
	40: "PUMP_BLUETOOTH_ERROR_CGM_ALERT",
	41: "DEFAULT_CGM_ALERT_41",
	42: "DEFAULT_CGM_ALERT_42",
	43: "DEFAULT_CGM_ALERT_43",
	44: "DEFAULT_CGM_ALERT_44",
	45: "TRANSMITTER_EXPIRING_CGM_ALERT",
	46: "TRANSMITTER_EXPIRING_CGM_ALERT2",
	47: "DEFAULT_CGM_ALERT_47",
	48: "UNAVAILABLE_CGM_ALERT",
	49: "DEFAULT_CGM_ALERT_49",
	50: "DEFAULT_CGM_ALERT_50",
	51: "DEFAULT_CGM_ALERT_51",
	52: "DEFAULT_CGM_ALERT_52",
	53: "DEFAULT_CGM_ALERT_53",
	55: "DEFAULT_CGM_ALERT_55",
	56: "DEFAULT_CGM_ALERT_56",
	57: "DEFAULT_CGM_ALERT_57",
	58: "DEFAULT_CGM_ALERT_58",
	59: "DEFAULT_CGM_ALERT_59",
	60: "DEFAULT_CGM_ALERT_60",
	61: "DEFAULT_CGM_ALERT_61",
	62: "DEFAULT_CGM_ALERT_62",
	63: "DEFAULT_CGM_ALERT_63",
}

// CGMAlertFromID is the CGM alert at bit id, or nil.
func CGMAlertFromID(id int32) *CGMAlert {
	if id < 0 || int(id) >= len(cgmAlertNames) || cgmAlertNames[id] == "" {
		return nil
	}
	a := CGMAlert(id)
	return &a
}

// CGMAlertFromBitmask is the CGM alerts whose bits bitmask sets, in
// declaration order; never nil.
func CGMAlertFromBitmask(bitmask *big.Int) []CGMAlert {
	out := []CGMAlert{}
	for _, a := range CGMAlerts {
		if bitmask.Bit(int(a)) != 0 {
			out = append(out, a)
		}
	}
	return out
}

// IsKnown reports whether pumpX2 names the alert, rather than numbering it
// DEFAULT_CGM_ALERT_n.
func (a CGMAlert) IsKnown() bool { return !strings.HasPrefix(a.String(), "DEFAULT_CGM_ALERT") }

func (a CGMAlert) String() string {
	if a >= 0 && int(a) < len(cgmAlertNames) && cgmAlertNames[a] != "" {
		return cgmAlertNames[a]
	}
	return fmt.Sprintf("CGMAlert(%d)", int32(a))
}
