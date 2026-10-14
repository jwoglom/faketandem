package messages

import "math/big"

// csResp1Bit is one enum constant with the bit fromBitmask tests for it.
type csResp1Bit struct {
	name string
	bit  int
}

func csResp1Bits(names ...string) []csResp1Bit {
	out := make([]csResp1Bit, len(names))
	for i, n := range names {
		out[i] = csResp1Bit{n, i}
	}
	return out
}

// csResp1FromBitmask is an enum's fromBitmask into a TreeSet: the constants
// whose bit is set, in declaration order.
func csResp1FromBitmask(bitmask *big.Int, values []csResp1Bit) []string {
	out := []string{}
	for _, v := range values {
		if bitmask.Bit(v.bit) != 0 {
			out = append(out, v.name)
		}
	}
	return out
}

// csResp1FromID is an enum's fromId over constants whose ids are their
// ordinals, with fallback for an id none has ("" for Java's null).
func csResp1FromID(names []string, id int32, fallback string) string {
	if id >= 0 && int(id) < len(names) {
		return names[id]
	}
	return fallback
}

func csResp1SetEnum(m *Message, field, name string) {
	if name == "" {
		m.setNull(field)
		return
	}
	m.setObject(field, name)
}

func csResp1Bool(b bool) byte {
	if b {
		return 1
	}
	return 0
}

var csResp1MalfunctionTypes = csResp1Bits(
	"SOFTWARE", "CPU_CORE", "ARM_MSP_COM", "SENSOR", "LIPO", "TOUCHSCREEN", "NVM", "DISPLAY", "MSP", "MOTOR",
	"EXTERNAL_BINS", "SW_INIT", "VIBE", "PERIPH_POWER", "P2", "DATALOG", "SPEAKER", "MSP_SUPPLY", "CAL_DATA", "BTLE",
	"AP", "RTC", "ARM_BLE_COM", "OVERTRAVEL_STALL", "UNDERTRAVEL_STALL", "PUSHOFF_STALL", "HW_CRYPTO_FAIL",
)

var csResp1AamTypes = []string{"REMINDER", "ALERT", "ALARM", "CGM_ALERT", "MALFUNCTION", "NONE"}

var csResp1AlarmResponseTypes = csResp1Bits(
	"CARTRIDGE_ALARM_DELIVERY_NOT_STOPPED", "CARTRIDGE_ALARM_SPOOL_MOVE", "OCCLUSION_ALARM", "PUMP_RESET_ALARM",
	"OBE_ALARM", "CARTRIDGE_ALARM_P2", "CARTRIDGE_ALARM_VENT_NOT_WORKING", "AUTO_OFF_ALARM", "EMPTY_CARTRIDGE_ALARM",
	"CARTRIDGE_ALARM_LIQ_EX", "TEMPERATURE_ALARM", "TEMPERATURE_ALARM2", "BATTERY_SHUTDOWN_ALARM", "DEFAULT_ALARM_13",
	"INVALID_DATE_ALARM", "TEMPERATURE_ALARM3", "CARTRIDGE_ALARM_DELIVERY_REQUEST_FAIL", "DEFAULT_ALARM_17",
	"RESUME_PUMP_ALARM", "DEFAULT_ALARM_19", "CARTRIDGE_ALARM_MOTOR_PICASSO", "ALTITUDE_ALARM", "STUCK_BUTTON_ALARM",
	"RESUME_PUMP_ALARM2", "ATMOSPHERIC_PRESSURE_OUT_OF_RANGE_ALARM", "CARTRIDGE_REMOVED_ALARM", "OCCLUSION_ALARM2",
	"DEFAULT_ALARM_27", "DEFAULT_ALARM_28", "CARTRIDGE_ALARM_MOTOR_CART_RANGE_LOW", "CARTRIDGE_ALARM_MOTOR_STALL",
	"CARTRIDGE_ALARM_MOTOR_VENT_STALL", "NO_CARTRIDGE_DETECTED_ALARM", "LIPO_INSUFF_POWER_ALARM",
	"CARTRIDGE_ALARM_MOTOR_HALL_EFFECT_FAIL", "MOTOR_FAILURE_MODE_ALARM", "MOTOR_PRIME_STALL_OBE_ALARM",
	"AUXCPU_SYSTEM_RESET_ALARM", "DEFAULT_ALARM_38", "DEFAULT_ALARM_39", "DEFAULT_ALARM_40", "DEFAULT_ALARM_41",
	"DEFAULT_ALARM_42", "DEFAULT_ALARM_43", "DEFAULT_ALARM_44", "DEFAULT_ALARM_45", "DEFAULT_ALARM_46",
	"DEFAULT_ALARM_47", "DEFAULT_ALARM_48", "DEFAULT_ALARM_49", "DEFAULT_ALARM_50", "DEFAULT_ALARM_51",
	"DEFAULT_ALARM_52", "DEFAULT_ALARM_53", "DEFAULT_ALARM_54", "DEFAULT_ALARM_55", "DEFAULT_ALARM_56",
	"DEFAULT_ALARM_57", "DEFAULT_ALARM_58", "DEFAULT_ALARM_59", "DEFAULT_ALARM_60", "DEFAULT_ALARM_61",
	"DEFAULT_ALARM_62", "DEFAULT_ALARM_63",
)

var csResp1AlertResponseTypes = []csResp1Bit{
	{"LOW_INSULIN_ALERT", 0}, {"USB_CONNECTION_ALERT", 1}, {"LOW_POWER_ALERT", 2}, {"LOW_POWER_ALERT2", 3},
	{"DATA_ERROR_ALERT", 4}, {"AUTO_OFF_ALERT", 5}, {"MAX_BASAL_RATE_ALERT", 6}, {"POWER_SOURCE_ALERT", 7},
	{"MIN_BASAL_ALERT", 8}, {"CONNECTION_ERROR_ALERT", 9}, {"CONNECTION_ERROR_ALERT2", 10},
	{"INCOMPLETE_BOLUS_ALERT", 11}, {"INCOMPLETE_TEMP_RATE_ALERT", 12}, {"INCOMPLETE_CARTRIDGE_CHANGE_ALERT", 13},
	{"INCOMPLETE_FILL_TUBING_ALERT", 14}, {"INCOMPLETE_FILL_CANNULA_ALERT", 15}, {"INCOMPLETE_SETTING_ALERT", 16},
	{"LOW_INSULIN_ALERT2", 17}, {"MAX_BASAL_ALERT", 18}, {"LOW_TRANSMITTER_ALERT", 19}, {"TRANSMITTER_ALERT", 20},
	{"FSL2_SUPPORT_PACKAGE_ERROR_ALERT", 21}, {"SENSOR_EXPIRING_ALERT", 22}, {"PUMP_REBOOTING_ALERT", 23},
	{"DEVICE_CONNECTION_ERROR", 24}, {"CGM_GRAPH_REMOVED", 25}, {"MIN_BASAL_ALERT2", 26},
	{"INCOMPLETE_CALIBRATION", 27}, {"CALIBRATION_TIMEOUT", 28}, {"INVALID_TRANSMITTER_ID", 29},
	{"DEBUG_INFO_DECLARED_ALERT", 30}, {"OPEN_SESAME_ALERT", 32}, {"BUTTON_ALERT", 33}, {"QUICK_BOLUS_ALERT", 34},
	{"BASAL_IQ_ALERT", 35}, {"STK_EXCHANGE_REQ_ALERT", 36}, {"STK_ODYSSEY_REQ_ALERT", 37},
	{"STK_INVALID_ALERT", 38}, {"TRANSMITTER_END_OF_LIFE", 39}, {"CGM_ERROR", 40}, {"CGM_ERROR2", 41},
	{"CGM_ERROR3", 42}, {"CGM_INCOMPATIBLE_ALERT", 43}, {"TRANSMITTER_EXPIRING_ALERT", 44},
	{"TRANSMITTER_EXPIRING_ALERT2", 45}, {"TRANSMITTER_EXPIRING_ALERT3", 46}, {"TZ_INCOMPLETE_SETTING_ALERT", 47},
	{"CGM_UNAVAILABLE", 48}, {"FILL_TUBING_STILL_IN_PROGRESS", 49}, {"TZ_HYPER_LIGHT_ALERT", 50},
	{"CONTROL_IQ_LOW", 51}, {"TZ_IIDM_ALERT", 52}, {"BLE_CRASH_DUMP_ALERT", 53}, {"DEVICE_PAIRED", 54},
	{"BLE_HW_ERROR_ALERT", 55}, {"USER_SET_MAX_BASAL_ALERT", 56}, {"SENSOR_ALERT", 57},
	{"COM_LOG_ERROR_LOGGED_ALERT", 58}, {"FCL_AUTOMATION_OFF_ALERT", 59}, {"CONTROLLER_CONNECTION_LOST_ALERT", 60},
	{"LIPO_SUPER_LOW_ALERT", 61}, {"LIPO_INSUFF_POWER_ALERT", 62}, {"BTLE_PAIRING_TIMEOUT_ALERT", 63},
}

var (
	csResp1CGMSessionStates              = []string{"SESSION_STOPPED", "SESSION_START_PENDING", "SESSION_ACTIVE", "SESSION_STOP_PENDING"}
	csResp1CGMTransmitterBatteryStatus   = []string{"UNAVAILABLE", "ERROR", "EXPIRED", "OK", "OUT_OF_RANGE"}
	csResp1CgmV2SessionStates            = []string{"STATUS_STOPPED", "STATUS_START_PENDING", "SESSION_ACTIVE", "SESSION_STOP_PENDING", "UNKNOWN"}
	csResp1CgmV2TransmitterBatteryStatus = []string{"UNAVAILABLE", "ERROR", "EXPIRED", "OKAY", "OUT_OF_RANGE", "UNKNOWN"}
	csResp1CgmV2SensorTypes              = []string{"NOT_APPLICABLE", "DEXCOM_G6", "FSL2", "DEXCOM_G7", "UNKNOWN"}
)

// csResp1CgmStatusV2 is CgmStatusV2Response.parse, which CGMStatusCResponse's
// parse repeats before its warmup fields.
func csResp1CgmStatusV2(m *Message, raw []byte) {
	check(len(raw) == m.size())
	m.Cargo = raw
	sessionStateID := ub(raw[0])
	m.setInt("sessionStateId", sessionStateID)
	m.setObject("sessionState", csResp1FromID(csResp1CgmV2SessionStates, sessionStateID, "UNKNOWN"))
	m.setLong("lastCalibrationTimestamp", readUint32(raw, 1))
	m.setLong("sensorStartedTimestamp", readUint32(raw, 5))
	batteryID := ub(raw[9])
	m.setInt("transmitterBatteryStatusId", batteryID)
	m.setObject("transmitterBatteryStatus", csResp1FromID(csResp1CgmV2TransmitterBatteryStatus, batteryID, "UNKNOWN"))
	m.setLong("sessionDurationSeconds", readUint32(raw, 10))
	m.setLong("sessionTimeRemainingSeconds", readUint32(raw, 14))
	sensorTypeID := ub(raw[18])
	m.setInt("cgmSensorTypeId", sensorTypeID)
	m.setObject("cgmSensorType", csResp1FromID(csResp1CgmV2SensorTypes, sensorTypeID, "UNKNOWN"))
	m.setBool("gracePeriod", ub(raw[19]) != 0)
}

func csResp1CgmStatusV2Cargo(a args) []byte {
	return combine(
		[]byte{byte(a.Int(0))},
		toUint32(a.Long(1)),
		toUint32(a.Long(2)),
		[]byte{byte(a.Int(3))},
		toUint32(a.Long(4)),
		toUint32(a.Long(5)),
		[]byte{byte(a.Int(6))},
		[]byte{csResp1Bool(a.Bool(7))},
	)
}

func init() {
	activeAamBitsParse := func(m *Message, raw []byte) {
		check(len(raw) == m.size())
		m.Cargo = raw
		unacknowledged := readUint64(raw, 0)
		active := readUint64(raw, 8)
		m.setBigInt("unacknowledgedBitmask", unacknowledged)
		m.setBigInt("activeBitmask", active)
		aamType := csResp1FromID(csResp1AamTypes, ub(raw[16]), "UNKNOWN")
		m.setObject("aamType", aamType)
		if aamType == "MALFUNCTION" {
			m.setEnumSet("unacknowledgedMalfunctions", csResp1FromBitmask(unacknowledged, csResp1MalfunctionTypes))
			m.setEnumSet("activeMalfunctions", csResp1FromBitmask(active, csResp1MalfunctionTypes))
		} else {
			m.setEnumSet("unacknowledgedMalfunctions", []string{})
			m.setEnumSet("activeMalfunctions", []string{})
		}
	}
	register("response.currentStatus.ActiveAamBitsResponse", classImpl{
		parse: activeAamBitsParse,
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) {
				m.Cargo = []byte{}
				m.setBigInt("unacknowledgedBitmask", big.NewInt(0))
				m.setBigInt("activeBitmask", big.NewInt(0))
				m.setObject("aamType", "UNKNOWN")
				m.setEnumSet("unacknowledgedMalfunctions", []string{})
				m.setEnumSet("activeMalfunctions", []string{})
			},
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				activeAamBitsParse(m, a.Bytes(0))
			},
		},
	})

	alarmStatusParse := func(m *Message, raw []byte) {
		check(len(raw) == m.size())
		intMap := readUint64(raw, 0)
		m.Cargo = raw
		m.setEnumSet("alarms", csResp1FromBitmask(intMap, csResp1AlarmResponseTypes))
	}
	register("response.currentStatus.AlarmStatusResponse", classImpl{
		parse: alarmStatusParse,
		ctors: map[string]func(*Message, args){
			"java.math.BigInteger": func(m *Message, a args) {
				m.Cargo = toUint64(lowBits(a.BigInt(0), 64))
				alarmStatusParse(m, m.Cargo)
			},
		},
	})

	alertStatusParse := func(m *Message, raw []byte) {
		check(len(raw) == m.size())
		intMap := readUint64(raw, 0)
		m.Cargo = raw
		m.setEnumSet("alerts", csResp1FromBitmask(intMap, csResp1AlertResponseTypes))
	}
	register("response.currentStatus.AlertStatusResponse", classImpl{
		parse: alertStatusParse,
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { alertStatusParse(m, a.Bytes(0)) },
			"java.math.BigInteger": func(m *Message, a args) {
				m.Cargo = toUint64(lowBits(a.BigInt(0), 64))
				alertStatusParse(m, m.Cargo)
			},
		},
	})

	register("response.currentStatus.BasalIQAlertInfoResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("alertId", readUint32(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"long": func(m *Message, a args) {
				m.Cargo = toUint32(a.Long(0))
				m.setLong("alertId", a.Long(0))
			},
		},
	})

	register("response.currentStatus.BasalIQSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("hypoMinimization", sb(raw[0]))
			m.setInt("suspendAlert", sb(raw[1]))
			m.setInt("resumeAlert", sb(raw[2]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1)), byte(a.Int(2))}
				m.setInt("hypoMinimization", a.Int(0))
				m.setInt("suspendAlert", a.Int(1))
				m.setInt("resumeAlert", a.Int(2))
			},
		},
	})

	register("response.currentStatus.BasalIQStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("basalIQStatusState", sb(raw[0]))
			m.setBool("deliveringTherapy", raw[1] != 0)
		},
		ctors: map[string]func(*Message, args){
			"int,boolean": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), csResp1Bool(a.Bool(1))}
				m.setInt("basalIQStatusState", a.Int(0))
				m.setBool("deliveringTherapy", a.Bool(1))
			},
		},
	})

	register("response.currentStatus.BasalLimitSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("basalLimit", readUint32(raw, 0))
			m.setLong("basalLimitDefault", readUint32(raw, 4))
		},
		ctors: map[string]func(*Message, args){
			"long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)))
				m.setLong("basalLimit", a.Long(0))
				m.setLong("basalLimitDefault", a.Long(1))
			},
		},
	})

	bleSoftwareInfoParse := func(m *Message, raw []byte) {
		m.Cargo = raw
		m.setInt("softDeviceId", readShort(raw, 0))
		m.setInt("softDeviceMajorVersion", readShort(raw, 2))
		m.setInt("softDeviceMinorVersion", readShort(raw, 4))
		m.setInt("softDeviceBugfixVersion", readShort(raw, 6))
		m.setLong("softDeviceVersion", readUint32(raw, 8))
		m.setInt("softDeviceSubVersion", readShort(raw, 12))
	}
	register("response.currentStatus.BleSoftwareInfoResponse", classImpl{
		parse: bleSoftwareInfoParse,
		ctors: map[string]func(*Message, args){
			"int,int,int,int,long,int": func(m *Message, a args) {
				m.Cargo = combine(
					firstTwoBytesLittleEndian(a.Int(0)),
					firstTwoBytesLittleEndian(a.Int(1)),
					firstTwoBytesLittleEndian(a.Int(2)),
					firstTwoBytesLittleEndian(a.Int(3)),
					toUint32(a.Long(4)),
					firstTwoBytesLittleEndian(a.Int(5)),
				)
				m.setInt("softDeviceId", a.Int(0))
				m.setInt("softDeviceMajorVersion", a.Int(1))
				m.setInt("softDeviceMinorVersion", a.Int(2))
				m.setInt("softDeviceBugfixVersion", a.Int(3))
				m.setLong("softDeviceVersion", a.Long(4))
				m.setInt("softDeviceSubVersion", a.Int(5))
			},
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				bleSoftwareInfoParse(m, a.Bytes(0))
			},
		},
	})

	bolusCalcDataSnapshot := func(m *Message, a args, unknown11bytes, unknown8bytes []byte) {
		m.Cargo = combine(
			[]byte{csResp1Bool(a.Bool(0))},
			firstTwoBytesLittleEndian(a.Int(1)),
			toUint32(a.Long(2)),
			firstTwoBytesLittleEndian(a.Int(3)),
			firstTwoBytesLittleEndian(a.Int(4)),
			firstTwoBytesLittleEndian(a.Int(5)),
			[]byte{csResp1Bool(a.Bool(6))},
			toUint32(a.Long(7)),
			firstTwoBytesLittleEndian(a.Int(8)),
			toUint32(a.Long(9)),
			[]byte{csResp1Bool(a.Bool(10))},
			[]byte{csResp1Bool(a.Bool(11))},
			unknown11bytes,
			[]byte{csResp1Bool(a.Bool(12))},
			unknown8bytes,
		)
		m.setBool("isUnacked", a.Bool(0))
		m.setInt("correctionFactor", a.Int(1))
		m.setLong("iob", a.Long(2))
		m.setInt("cartridgeRemainingInsulin", a.Int(3))
		m.setInt("targetBg", a.Int(4))
		m.setInt("isf", a.Int(5))
		m.setBool("carbEntryEnabled", a.Bool(6))
		m.setLong("carbRatio", a.Long(7))
		m.setInt("maxBolusAmount", a.Int(8))
		m.setLong("maxBolusHourlyTotal", a.Long(9))
		m.setBool("maxBolusEventsExceeded", a.Bool(10))
		m.setBool("maxIobEventsExceeded", a.Bool(11))
		m.setBool("isAutopopAllowed", a.Bool(12))
	}
	register("response.currentStatus.BolusCalcDataSnapshotResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("isUnacked", raw[0] != 0)
			m.setInt("correctionFactor", readShort(raw, 1))
			m.setLong("iob", readUint32(raw, 3))
			m.setInt("cartridgeRemainingInsulin", readShort(raw, 7))
			m.setInt("targetBg", readShort(raw, 9))
			m.setInt("isf", readShort(raw, 11))
			m.setBool("carbEntryEnabled", raw[13] != 0)
			m.setLong("carbRatio", readUint32(raw, 14))
			m.setInt("maxBolusAmount", readShort(raw, 18))
			m.setLong("maxBolusHourlyTotal", readUint32(raw, 20))
			m.setBool("maxBolusEventsExceeded", raw[24] != 0)
			m.setBool("maxIobEventsExceeded", raw[25] != 0)
			m.setBool("isAutopopAllowed", raw[37] != 0)
		},
		ctors: map[string]func(*Message, args){
			"boolean,int,long,int,int,int,boolean,long,int,long,boolean,boolean,boolean,byte[],byte[]": func(m *Message, a args) {
				bolusCalcDataSnapshot(m, a, a.Bytes(13), a.Bytes(14))
			},
			"boolean,int,long,int,int,int,boolean,long,int,long,boolean,boolean,boolean": func(m *Message, a args) {
				bolusCalcDataSnapshot(m, a, make([]byte, 11), make([]byte, 8))
			},
		},
	})

	register("response.currentStatus.BolusPermissionChangeReasonResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("bolusId", readShort(raw, 0))
			m.setBool("isAcked", raw[2] != 0)
			m.setInt("lastChangeReasonId", sb(raw[3]))
			m.setBool("currentPermissionHolder", raw[4] != 0)
		},
		ctors: map[string]func(*Message, args){
			"int,boolean,int,boolean": func(m *Message, a args) {
				m.Cargo = combine(
					firstTwoBytesLittleEndian(a.Int(0)),
					[]byte{csResp1Bool(a.Bool(1))},
					[]byte{byte(a.Int(2))},
					[]byte{csResp1Bool(a.Bool(3))},
				)
				m.setInt("bolusId", a.Int(0))
				m.setBool("isAcked", a.Bool(1))
				m.setInt("lastChangeReasonId", a.Int(2))
				m.setBool("currentPermissionHolder", a.Bool(3))
			},
		},
	})

	register("response.currentStatus.CGMAlertStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			readUint64(raw, 0)
		},
		ctors: map[string]func(*Message, args){
			"long": func(m *Message, a args) { m.Cargo = toUint64(a.Long(0)) },
		},
	})

	register("response.currentStatus.CGMGlucoseAlertSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("highGlucoseAlertThreshold", readShort(raw, 0))
			m.setInt("highGlucoseAlertEnabled", sb(raw[2]))
			m.setInt("highGlucoseRepeatDuration", readShort(raw, 3))
			m.setInt("highGlucoseAlertDefaultBitmask", sb(raw[5]))
			m.setInt("lowGlucoseAlertThreshold", readShort(raw, 6))
			m.setInt("lowGlucoseAlertEnabled", sb(raw[8]))
			m.setInt("lowGlucoseRepeatDuration", readShort(raw, 9))
			m.setInt("lowGlucoseAlertDefaultBitmask", sb(raw[11]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,int,int,int,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					firstTwoBytesLittleEndian(a.Int(0)),
					[]byte{byte(a.Int(1))},
					firstTwoBytesLittleEndian(a.Int(2)),
					[]byte{byte(a.Int(3))},
					firstTwoBytesLittleEndian(a.Int(4)),
					[]byte{byte(a.Int(5))},
					firstTwoBytesLittleEndian(a.Int(6)),
					[]byte{byte(a.Int(7))},
				)
				m.setInt("highGlucoseAlertThreshold", a.Int(0))
				m.setInt("highGlucoseAlertEnabled", a.Int(1))
				m.setInt("highGlucoseRepeatDuration", a.Int(2))
				m.setInt("highGlucoseAlertDefaultBitmask", a.Int(3))
				m.setInt("lowGlucoseAlertThreshold", a.Int(4))
				m.setInt("lowGlucoseAlertEnabled", a.Int(5))
				m.setInt("lowGlucoseRepeatDuration", a.Int(6))
				m.setInt("lowGlucoseAlertDefaultBitmask", a.Int(7))
			},
		},
	})

	register("response.currentStatus.CGMHardwareInfoResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.setString("hardwareInfoString", readString(raw, 0, 16))
			m.setInt("lastByte", sb(raw[16]))
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"java.lang.String,int": func(m *Message, a args) {
				m.Cargo = combine(writeString(a.String(0), 16), []byte{byte(a.Int(1))})
				m.setString("hardwareInfoString", a.String(0))
				m.setInt("lastByte", a.Int(1))
			},
		},
	})

	register("response.currentStatus.CGMOORAlertSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("sensorTimeoutAlertThreshold", sb(raw[0]))
			m.setInt("sensorTimeoutAlertEnabled", sb(raw[1]))
			m.setInt("sensorTimeoutDefaultBitmask", sb(raw[2]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1)), byte(a.Int(2))}
				m.setInt("sensorTimeoutAlertThreshold", a.Int(0))
				m.setInt("sensorTimeoutAlertEnabled", a.Int(1))
				m.setInt("sensorTimeoutDefaultBitmask", a.Int(2))
			},
		},
	})

	register("response.currentStatus.CGMRateAlertSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("riseRateThreshold", sb(raw[0]))
			m.setInt("riseRateEnabled", sb(raw[1]))
			m.setInt("riseRateDefaultBitmask", sb(raw[2]))
			m.setInt("fallRateThreshold", sb(raw[3]))
			m.setInt("fallRateEnabled", sb(raw[4]))
			m.setInt("fallRateDefaultBitmask", sb(raw[5]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,int,int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1)), byte(a.Int(2)), byte(a.Int(3)), byte(a.Int(4)), byte(a.Int(5))}
				m.setInt("riseRateThreshold", a.Int(0))
				m.setInt("riseRateEnabled", a.Int(1))
				m.setInt("riseRateDefaultBitmask", a.Int(2))
				m.setInt("fallRateThreshold", a.Int(3))
				m.setInt("fallRateEnabled", a.Int(4))
				m.setInt("fallRateDefaultBitmask", a.Int(5))
			},
		},
	})

	cgmStatusCParse := func(m *Message, raw []byte) {
		csResp1CgmStatusV2(m, raw)
		m.setInt("sensorWarmupDurationSeconds", readShort(raw, 20))
		m.setInt("remainingWarmupDurationSeconds", readShort(raw, 22))
	}
	register("response.currentStatus.CGMStatusCResponse", classImpl{
		parse: cgmStatusCParse,
		ctors: map[string]func(*Message, args){
			"int,long,long,int,long,long,int,boolean,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					csResp1CgmStatusV2Cargo(a),
					firstTwoBytesLittleEndian(a.Int(8)),
					firstTwoBytesLittleEndian(a.Int(9)),
				)
				cgmStatusCParse(m, m.Cargo)
			},
		},
	})

	cgmStatusParse := func(m *Message, raw []byte) {
		check(len(raw) == m.size())
		m.Cargo = raw
		sessionStateID := sb(raw[0])
		m.setInt("sessionStateId", sessionStateID)
		csResp1SetEnum(m, "sessionState", csResp1FromID(csResp1CGMSessionStates, sessionStateID, ""))
		m.setLong("lastCalibrationTimestamp", readUint32(raw, 1))
		m.setLong("sensorStartedTimestamp", readUint32(raw, 5))
		batteryID := sb(raw[9])
		m.setInt("transmitterBatteryStatusId", batteryID)
		csResp1SetEnum(m, "transmitterBatteryStatus", csResp1FromID(csResp1CGMTransmitterBatteryStatus, batteryID, ""))
	}
	register("response.currentStatus.CGMStatusResponse", classImpl{
		parse: cgmStatusParse,
		ctors: map[string]func(*Message, args){
			"int,long,long,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					toUint32(a.Long(1)),
					toUint32(a.Long(2)),
					[]byte{byte(a.Int(3))},
				)
				cgmStatusParse(m, m.Cargo)
			},
		},
	})

	register("response.currentStatus.CgmStatusV2Response", classImpl{
		parse: csResp1CgmStatusV2,
		ctors: map[string]func(*Message, args){
			"int,long,long,int,long,long,int,boolean": func(m *Message, a args) {
				m.Cargo = csResp1CgmStatusV2Cargo(a)
				csResp1CgmStatusV2(m, m.Cargo)
			},
		},
	})

	register("response.currentStatus.CgmSupportPackageStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", ub(raw[0]))
			m.setBool("validity", ub(raw[1]) != 0)
		},
		ctors: map[string]func(*Message, args){
			"int,boolean": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0) & 0xFF), csResp1Bool(a.Bool(1))}
				m.setInt("status", a.Int(0))
				m.setBool("validity", a.Bool(1))
			},
		},
	})
}
