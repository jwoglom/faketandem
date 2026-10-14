package currentstatus

import (
	"fmt"
	"math/big"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// AlarmStatusResponse is the pump's active alarms: a 64-bit mask with an
// AlarmResponseType at each bit.
type AlarmStatusResponse struct {
	messages.Base
	// IntMap is the mask, which pumpX2 leaves out of toString.
	IntMap *big.Int
	Alarms []AlarmResponseType `pumpx2:"alarms"`
}

var alarmStatusResponseProps = messages.Props{OpCode: 71, Size: 8, Type: messages.Response, Request: "AlarmStatusRequest"}

func init() {
	messages.Register("AlarmStatusResponse", &AlarmStatusResponse{}, alarmStatusResponseProps,
		messages.Constructor{Params: "AlarmStatusResponse.AlarmResponseType[] alarms", Func: NewAlarmStatusResponseFromAlarms},
		messages.Constructor{Params: "BigInteger intMap", Func: NewAlarmStatusResponse},
	)
}

// NewAlarmStatusResponse is the response with the low 64 bits of intMap.
func NewAlarmStatusResponse(intMap *big.Int) *AlarmStatusResponse {
	m := &AlarmStatusResponse{}
	m.decode(helpers.ToUint64(helpers.LongValue(intMap)))
	return m
}

// NewAlarmStatusResponseFromAlarms is the response with alarms active.
func NewAlarmStatusResponseFromAlarms(alarms ...AlarmResponseType) *AlarmStatusResponse {
	return NewAlarmStatusResponse(AlarmResponseTypeToBitmask(alarms...))
}

// Parse implements messages.Message.
func (m *AlarmStatusResponse) Parse(raw []byte) error {
	if len(raw) != alarmStatusResponseProps.Size {
		return messages.InvalidLength("AlarmStatusResponse", len(raw), alarmStatusResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *AlarmStatusResponse) decode(raw []byte) {
	m.IntMap = helpers.ReadUint64(raw, 0)
	m.SetCargo(raw)
	m.Alarms = AlarmResponseTypeFromBitmask(m.IntMap)
}

// Size is how many alarms are active.
func (m *AlarmStatusResponse) Size() int { return len(m.Alarms) }

// NotificationIDs are the active alarms pumpX2 has a description for.
func (m *AlarmStatusResponse) NotificationIDs() []int32 {
	ids := []int32{}
	for _, a := range m.Alarms {
		if a.IsKnown() {
			ids = append(ids, int32(a))
		}
	}
	return ids
}

// AlarmResponseType is a pump alarm, as its bit in AlarmStatusResponse's
// mask.
type AlarmResponseType int32

// The alarms, by their bits.
const (
	AlarmResponseTypeCartridgeAlarmDeliveryNotStopped   AlarmResponseType = 0
	AlarmResponseTypeCartridgeAlarmSpoolMove            AlarmResponseType = 1
	AlarmResponseTypeOcclusionAlarm                     AlarmResponseType = 2
	AlarmResponseTypePumpResetAlarm                     AlarmResponseType = 3
	AlarmResponseTypeObeAlarm                           AlarmResponseType = 4
	AlarmResponseTypeCartridgeAlarmP2                   AlarmResponseType = 5
	AlarmResponseTypeCartridgeAlarmVentNotWorking       AlarmResponseType = 6
	AlarmResponseTypeAutoOffAlarm                       AlarmResponseType = 7
	AlarmResponseTypeEmptyCartridgeAlarm                AlarmResponseType = 8
	AlarmResponseTypeCartridgeAlarmLiqEx                AlarmResponseType = 9
	AlarmResponseTypeTemperatureAlarm                   AlarmResponseType = 10
	AlarmResponseTypeTemperatureAlarm2                  AlarmResponseType = 11
	AlarmResponseTypeBatteryShutdownAlarm               AlarmResponseType = 12
	AlarmResponseTypeDefaultAlarm13                     AlarmResponseType = 13
	AlarmResponseTypeInvalidDateAlarm                   AlarmResponseType = 14
	AlarmResponseTypeTemperatureAlarm3                  AlarmResponseType = 15
	AlarmResponseTypeCartridgeAlarmDeliveryRequestFail  AlarmResponseType = 16
	AlarmResponseTypeDefaultAlarm17                     AlarmResponseType = 17
	AlarmResponseTypeResumePumpAlarm                    AlarmResponseType = 18
	AlarmResponseTypeDefaultAlarm19                     AlarmResponseType = 19
	AlarmResponseTypeCartridgeAlarmMotorPicasso         AlarmResponseType = 20
	AlarmResponseTypeAltitudeAlarm                      AlarmResponseType = 21
	AlarmResponseTypeStuckButtonAlarm                   AlarmResponseType = 22
	AlarmResponseTypeResumePumpAlarm2                   AlarmResponseType = 23
	AlarmResponseTypeAtmosphericPressureOutOfRangeAlarm AlarmResponseType = 24
	AlarmResponseTypeCartridgeRemovedAlarm              AlarmResponseType = 25
	AlarmResponseTypeOcclusionAlarm2                    AlarmResponseType = 26
	AlarmResponseTypeDefaultAlarm27                     AlarmResponseType = 27
	AlarmResponseTypeDefaultAlarm28                     AlarmResponseType = 28
	AlarmResponseTypeCartridgeAlarmMotorCartRangeLow    AlarmResponseType = 29
	AlarmResponseTypeCartridgeAlarmMotorStall           AlarmResponseType = 30
	AlarmResponseTypeCartridgeAlarmMotorVentStall       AlarmResponseType = 31
	AlarmResponseTypeNoCartridgeDetectedAlarm           AlarmResponseType = 32
	AlarmResponseTypeLipoInsuffPowerAlarm               AlarmResponseType = 33
	AlarmResponseTypeCartridgeAlarmMotorHallEffectFail  AlarmResponseType = 34
	AlarmResponseTypeMotorFailureModeAlarm              AlarmResponseType = 35
	AlarmResponseTypeMotorPrimeStallObeAlarm            AlarmResponseType = 36
	AlarmResponseTypeAuxcpuSystemResetAlarm             AlarmResponseType = 37
	AlarmResponseTypeDefaultAlarm38                     AlarmResponseType = 38
	AlarmResponseTypeDefaultAlarm39                     AlarmResponseType = 39
	AlarmResponseTypeDefaultAlarm40                     AlarmResponseType = 40
	AlarmResponseTypeDefaultAlarm41                     AlarmResponseType = 41
	AlarmResponseTypeDefaultAlarm42                     AlarmResponseType = 42
	AlarmResponseTypeDefaultAlarm43                     AlarmResponseType = 43
	AlarmResponseTypeDefaultAlarm44                     AlarmResponseType = 44
	AlarmResponseTypeDefaultAlarm45                     AlarmResponseType = 45
	AlarmResponseTypeDefaultAlarm46                     AlarmResponseType = 46
	AlarmResponseTypeDefaultAlarm47                     AlarmResponseType = 47
	AlarmResponseTypeDefaultAlarm48                     AlarmResponseType = 48
	AlarmResponseTypeDefaultAlarm49                     AlarmResponseType = 49
	AlarmResponseTypeDefaultAlarm50                     AlarmResponseType = 50
	AlarmResponseTypeDefaultAlarm51                     AlarmResponseType = 51
	AlarmResponseTypeDefaultAlarm52                     AlarmResponseType = 52
	AlarmResponseTypeDefaultAlarm53                     AlarmResponseType = 53
	AlarmResponseTypeDefaultAlarm54                     AlarmResponseType = 54
	AlarmResponseTypeDefaultAlarm55                     AlarmResponseType = 55
	AlarmResponseTypeDefaultAlarm56                     AlarmResponseType = 56
	AlarmResponseTypeDefaultAlarm57                     AlarmResponseType = 57
	AlarmResponseTypeDefaultAlarm58                     AlarmResponseType = 58
	AlarmResponseTypeDefaultAlarm59                     AlarmResponseType = 59
	AlarmResponseTypeDefaultAlarm60                     AlarmResponseType = 60
	AlarmResponseTypeDefaultAlarm61                     AlarmResponseType = 61
	AlarmResponseTypeDefaultAlarm62                     AlarmResponseType = 62
	AlarmResponseTypeDefaultAlarm63                     AlarmResponseType = 63
)

// AlarmResponseTypes are the alarms in pumpX2's declaration order.
var AlarmResponseTypes = []AlarmResponseType{
	AlarmResponseTypeCartridgeAlarmDeliveryNotStopped, AlarmResponseTypeCartridgeAlarmSpoolMove,
	AlarmResponseTypeOcclusionAlarm, AlarmResponseTypePumpResetAlarm, AlarmResponseTypeObeAlarm,
	AlarmResponseTypeCartridgeAlarmP2, AlarmResponseTypeCartridgeAlarmVentNotWorking, AlarmResponseTypeAutoOffAlarm,
	AlarmResponseTypeEmptyCartridgeAlarm, AlarmResponseTypeCartridgeAlarmLiqEx, AlarmResponseTypeTemperatureAlarm,
	AlarmResponseTypeTemperatureAlarm2, AlarmResponseTypeBatteryShutdownAlarm, AlarmResponseTypeDefaultAlarm13,
	AlarmResponseTypeInvalidDateAlarm, AlarmResponseTypeTemperatureAlarm3,
	AlarmResponseTypeCartridgeAlarmDeliveryRequestFail, AlarmResponseTypeDefaultAlarm17, AlarmResponseTypeResumePumpAlarm,
	AlarmResponseTypeDefaultAlarm19, AlarmResponseTypeCartridgeAlarmMotorPicasso, AlarmResponseTypeAltitudeAlarm,
	AlarmResponseTypeStuckButtonAlarm, AlarmResponseTypeResumePumpAlarm2,
	AlarmResponseTypeAtmosphericPressureOutOfRangeAlarm, AlarmResponseTypeCartridgeRemovedAlarm,
	AlarmResponseTypeOcclusionAlarm2, AlarmResponseTypeDefaultAlarm27, AlarmResponseTypeDefaultAlarm28,
	AlarmResponseTypeCartridgeAlarmMotorCartRangeLow, AlarmResponseTypeCartridgeAlarmMotorStall,
	AlarmResponseTypeCartridgeAlarmMotorVentStall, AlarmResponseTypeNoCartridgeDetectedAlarm,
	AlarmResponseTypeLipoInsuffPowerAlarm, AlarmResponseTypeCartridgeAlarmMotorHallEffectFail,
	AlarmResponseTypeMotorFailureModeAlarm, AlarmResponseTypeMotorPrimeStallObeAlarm,
	AlarmResponseTypeAuxcpuSystemResetAlarm, AlarmResponseTypeDefaultAlarm38, AlarmResponseTypeDefaultAlarm39,
	AlarmResponseTypeDefaultAlarm40, AlarmResponseTypeDefaultAlarm41, AlarmResponseTypeDefaultAlarm42,
	AlarmResponseTypeDefaultAlarm43, AlarmResponseTypeDefaultAlarm44, AlarmResponseTypeDefaultAlarm45,
	AlarmResponseTypeDefaultAlarm46, AlarmResponseTypeDefaultAlarm47, AlarmResponseTypeDefaultAlarm48,
	AlarmResponseTypeDefaultAlarm49, AlarmResponseTypeDefaultAlarm50, AlarmResponseTypeDefaultAlarm51,
	AlarmResponseTypeDefaultAlarm52, AlarmResponseTypeDefaultAlarm53, AlarmResponseTypeDefaultAlarm54,
	AlarmResponseTypeDefaultAlarm55, AlarmResponseTypeDefaultAlarm56, AlarmResponseTypeDefaultAlarm57,
	AlarmResponseTypeDefaultAlarm58, AlarmResponseTypeDefaultAlarm59, AlarmResponseTypeDefaultAlarm60,
	AlarmResponseTypeDefaultAlarm61, AlarmResponseTypeDefaultAlarm62, AlarmResponseTypeDefaultAlarm63,
}

var alarmResponseTypeNames = [64]string{
	0:  "CARTRIDGE_ALARM_DELIVERY_NOT_STOPPED",
	1:  "CARTRIDGE_ALARM_SPOOL_MOVE",
	2:  "OCCLUSION_ALARM",
	3:  "PUMP_RESET_ALARM",
	4:  "OBE_ALARM",
	5:  "CARTRIDGE_ALARM_P2",
	6:  "CARTRIDGE_ALARM_VENT_NOT_WORKING",
	7:  "AUTO_OFF_ALARM",
	8:  "EMPTY_CARTRIDGE_ALARM",
	9:  "CARTRIDGE_ALARM_LIQ_EX",
	10: "TEMPERATURE_ALARM",
	11: "TEMPERATURE_ALARM2",
	12: "BATTERY_SHUTDOWN_ALARM",
	13: "DEFAULT_ALARM_13",
	14: "INVALID_DATE_ALARM",
	15: "TEMPERATURE_ALARM3",
	16: "CARTRIDGE_ALARM_DELIVERY_REQUEST_FAIL",
	17: "DEFAULT_ALARM_17",
	18: "RESUME_PUMP_ALARM",
	19: "DEFAULT_ALARM_19",
	20: "CARTRIDGE_ALARM_MOTOR_PICASSO",
	21: "ALTITUDE_ALARM",
	22: "STUCK_BUTTON_ALARM",
	23: "RESUME_PUMP_ALARM2",
	24: "ATMOSPHERIC_PRESSURE_OUT_OF_RANGE_ALARM",
	25: "CARTRIDGE_REMOVED_ALARM",
	26: "OCCLUSION_ALARM2",
	27: "DEFAULT_ALARM_27",
	28: "DEFAULT_ALARM_28",
	29: "CARTRIDGE_ALARM_MOTOR_CART_RANGE_LOW",
	30: "CARTRIDGE_ALARM_MOTOR_STALL",
	31: "CARTRIDGE_ALARM_MOTOR_VENT_STALL",
	32: "NO_CARTRIDGE_DETECTED_ALARM",
	33: "LIPO_INSUFF_POWER_ALARM",
	34: "CARTRIDGE_ALARM_MOTOR_HALL_EFFECT_FAIL",
	35: "MOTOR_FAILURE_MODE_ALARM",
	36: "MOTOR_PRIME_STALL_OBE_ALARM",
	37: "AUXCPU_SYSTEM_RESET_ALARM",
	38: "DEFAULT_ALARM_38",
	39: "DEFAULT_ALARM_39",
	40: "DEFAULT_ALARM_40",
	41: "DEFAULT_ALARM_41",
	42: "DEFAULT_ALARM_42",
	43: "DEFAULT_ALARM_43",
	44: "DEFAULT_ALARM_44",
	45: "DEFAULT_ALARM_45",
	46: "DEFAULT_ALARM_46",
	47: "DEFAULT_ALARM_47",
	48: "DEFAULT_ALARM_48",
	49: "DEFAULT_ALARM_49",
	50: "DEFAULT_ALARM_50",
	51: "DEFAULT_ALARM_51",
	52: "DEFAULT_ALARM_52",
	53: "DEFAULT_ALARM_53",
	54: "DEFAULT_ALARM_54",
	55: "DEFAULT_ALARM_55",
	56: "DEFAULT_ALARM_56",
	57: "DEFAULT_ALARM_57",
	58: "DEFAULT_ALARM_58",
	59: "DEFAULT_ALARM_59",
	60: "DEFAULT_ALARM_60",
	61: "DEFAULT_ALARM_61",
	62: "DEFAULT_ALARM_62",
	63: "DEFAULT_ALARM_63",
}

var alarmResponseTypeDescriptions = [64]string{
	AlarmResponseTypeCartridgeAlarmDeliveryNotStopped:   "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeCartridgeAlarmSpoolMove:            "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeOcclusionAlarm:                     "An occlusion has occurred. Please check your pump site and tubing and restart insulin delivery.",
	AlarmResponseTypePumpResetAlarm:                     "The pump was reset. IOB has been reset to 0 and CGM may need to be re-activated.",
	AlarmResponseTypeCartridgeAlarmP2:                   "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeCartridgeAlarmVentNotWorking:       "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeAutoOffAlarm:                       "Pump will stop delivering insulin automatically soon because no user activity has occurred and the auto-off setting is enabled.",
	AlarmResponseTypeEmptyCartridgeAlarm:                "Cartridge is out of insulin and insulin delivery cannot occur. Please fill a new cartridge.",
	AlarmResponseTypeCartridgeAlarmLiqEx:                "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeTemperatureAlarm:                   "Pump temperature is out of range and insulin cannot be safely delivered.",
	AlarmResponseTypeTemperatureAlarm2:                  "Pump temperature is out of range and insulin cannot be safely delivered.",
	AlarmResponseTypeBatteryShutdownAlarm:               "Pump battery level is critically low and the device will shut down. Please charge pump immediately.",
	AlarmResponseTypeInvalidDateAlarm:                   "The pump's configured date is invalid.",
	AlarmResponseTypeTemperatureAlarm3:                  "Pump temperature is out of range and insulin cannot be safely delivered.",
	AlarmResponseTypeCartridgeAlarmDeliveryRequestFail:  "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeResumePumpAlarm:                    "Insulin delivery is currently off. Please restart insulin delivery soon.",
	AlarmResponseTypeCartridgeAlarmMotorPicasso:         "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeAltitudeAlarm:                      "Pump altitude is out of range and insulin cannot be safely delivered.",
	AlarmResponseTypeStuckButtonAlarm:                   "The pump button may be stuck or has been pressed for too long a period of time.",
	AlarmResponseTypeResumePumpAlarm2:                   "Insulin delivery is currently off. Please restart insulin delivery soon.",
	AlarmResponseTypeAtmosphericPressureOutOfRangeAlarm: "Pump atmospheric pressure is out of range and insulin cannot be safely delivered.",
	AlarmResponseTypeCartridgeRemovedAlarm:              "The cartridge was removed from the pump. Please fill a new cartridge.",
	AlarmResponseTypeOcclusionAlarm2:                    "An occlusion has occurred. Please check your pump site and tubing and restart insulin delivery.",
	AlarmResponseTypeCartridgeAlarmMotorCartRangeLow:    "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeCartridgeAlarmMotorStall:           "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeCartridgeAlarmMotorVentStall:       "There is an issue with the cartridge and it needs to be replaced.",
	AlarmResponseTypeCartridgeAlarmMotorHallEffectFail:  "There is an issue with the cartridge and it needs to be replaced.",
}

// AlarmResponseTypeFromBitmask is the alarms whose bits intMap sets, in
// declaration order; never nil.
func AlarmResponseTypeFromBitmask(intMap *big.Int) []AlarmResponseType {
	out := []AlarmResponseType{}
	for _, t := range AlarmResponseTypes {
		if intMap.Bit(int(t)) != 0 {
			out = append(out, t)
		}
	}
	return out
}

// AlarmResponseTypeToBitmask is the mask with the alarms' bits set.
func AlarmResponseTypeToBitmask(alarms ...AlarmResponseType) *big.Int {
	mask := new(big.Int)
	for _, t := range alarms {
		mask.SetBit(mask, int(t), 1)
	}
	return mask
}

// AlarmResponseTypeFromSingularID is the alarm at bit id, or nil.
func AlarmResponseTypeFromSingularID(id int64) *AlarmResponseType {
	if id < 0 || id >= int64(len(alarmResponseTypeNames)) {
		return nil
	}
	t := AlarmResponseType(id)
	return &t
}

// Description is what the alarm means, for the alarms pumpX2 describes, or
// "".
func (t AlarmResponseType) Description() string {
	if t >= 0 && int(t) < len(alarmResponseTypeDescriptions) {
		return alarmResponseTypeDescriptions[t]
	}
	return ""
}

// IsKnown reports whether pumpX2 describes the alarm.
func (t AlarmResponseType) IsKnown() bool { return t.Description() != "" }

func (t AlarmResponseType) String() string {
	if t >= 0 && int(t) < len(alarmResponseTypeNames) {
		return alarmResponseTypeNames[t]
	}
	return fmt.Sprintf("AlarmResponseType(%d)", int32(t))
}
