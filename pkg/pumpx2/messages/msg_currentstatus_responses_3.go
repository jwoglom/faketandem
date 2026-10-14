package messages

import (
	"math/big"
	"time"
)

type csResp3Flag struct {
	name    string
	bitmask int64
}

var csResp3PumpFeatureTypes = []csResp3Flag{
	{"DEXCOM_G5_SUPPORTED", 1},
	{"DEXCOM_G6_SUPPORTED", 2},
	{"BASAL_IQ_SUPPORTED", 4},
	{"CONTROL_IQ_SUPPORTED", 1024},
	{"WOMBAT_SUPPORTED", 65536},
	{"BASAL_LIMIT_SUPPORTED", 262144},
	{"CONTROL_IQ_PRO_SUPPORTED", 8388608},
	{"AUTO_POP_SUPPORTED", 33554432},
	{"BLE_PUMP_CONTROL_SUPPORTED", 268435456},
	{"PUMP_SETTINGS_IN_IDP_GUI_SUPPORTED", 536870912},
}

var csResp3ControlIqProFeatureTypes = []csResp3Flag{
	{"BLE_CONTROL", 1},
	{"GUI_CHANGES", 2},
	{"RANGE_CHANGES", 4},
	{"NVM_PARAM", 8},
	{"EXTENDED_BOLUS_DURATION", 16},
	{"AUTO_BOLUS", 32},
}

var csResp3ControlFeatureTypes = []csResp3Flag{
	{"CHANGE_CARTRIDGE_CONTROL", 1},
	{"BASAL_DELIVERY_CONTROL", 2},
	{"STANDARD_BOLUS_CONTROL", 4},
	{"EXTENDED_BOLUS_CONTROL", 8},
	{"TEMP_RATE_CONTROL", 16},
	{"CGM_CONTROL", 32},
	{"AAM_CONTROL", 64},
	{"SHELF_MODE_CONTROL", 128},
	{"IDP_CONTROL", 256},
	{"LED_PATTERN_CONTROL", 512},
	{"UNREGISTER_CONTROL", 1024},
	{"PUMP_TIME_CONTROL", 2048},
	{"PLGS_CONTROL", 4096},
	{"TZ_CONTROL", 8192},
	{"REMINDER_SETTINGS_CONTROL", 16384},
	{"ALERT_SETTINGS_CONTROL", 32768},
	{"ANNUM_CONTROL", 65536},
	{"AAM_NOTIFY_ME_ON_CONTROL", 131072},
	{"PUMP_GLOBAL_SETTINGS_CONTROL", 262144},
	{"FACTORY_CONFIG_CONTROL", 524288},
	{"SNOOZE_SETTINGS_CONTROL", 1048576},
	{"UPDATED_SETTINGS_QE_CONTROL", 2097152},
	{"FIND_MY_PUMP_CONTROL", 4194304},
	{"BOLUS_OVERFLOW_CONTROL", 8388608},
	{"PREFLIGHT_CONTROL", 16777216},
	{"USER_INTERACTION_CONTROL", 33554432},
	{"BLOOD_GLUCOSE_ENTRY_CONTROL", 67108864},
	{"HLOG_CREATOR_CONTROL", 134217728},
	{"FEATURE_MANAGEMENT", 268435456},
	{"MOBI_RESEARCH_PUMP", 536870912},
	{"TIPS_CONTROL_GENERIC_TEST", 1073741824},
}

var csResp3ControlIqFeatureTypes = []csResp3Flag{
	{"TIMED_EXERCISE", 1},
	{"CONTROL_IQ_VERSION_ONE_FIVE", 256},
	{"TEMP_RATE_ENABLED", 512},
	{"EXTENDED_BOLUS_ENABLED", 2048},
}

var csResp3DexcomFeatureTypes = []csResp3Flag{
	{"G5_CGM", 1},
	{"G6_CGM", 2},
	{"G7_CGM", 4},
	{"AUTO_CAL", 8},
	{"DISABLE_GRAPH_SMOOTHING", 16},
}

var csResp3AbbottFeatureTypes = []csResp3Flag{
	{"IS_FSL2", 1},
}

var csResp3SecondaryFeatureTypes = []csResp3Flag{
	{"BASELINE_PROFILES", 1},
	{"MULTI_CGM", 2},
	{"BLE_TDU", 4},
}

var csResp3MalfunctionTypes = []string{
	"SOFTWARE", "CPU_CORE", "ARM_MSP_COM", "SENSOR", "LIPO", "TOUCHSCREEN", "NVM", "DISPLAY", "MSP", "MOTOR",
	"EXTERNAL_BINS", "SW_INIT", "VIBE", "PERIPH_POWER", "P2", "DATALOG", "SPEAKER", "MSP_SUPPLY", "CAL_DATA", "BTLE",
	"AP", "RTC", "ARM_BLE_COM", "OVERTRAVEL_STALL", "UNDERTRAVEL_STALL", "PUSHOFF_STALL", "HW_CRYPTO_FAIL",
}

var (
	csResp3LoadStates         = []string{"CHANGE_CARTRIDGE", "LOAD_CARTRIDGE", "PRIME_TUBING", "PRIME_CANNULA", "PRIME_NUDGE", "INVALID", "UNKNOWN"}
	csResp3PrimeTubingStatus  = []string{"STOP", "START", "ENTERED_CAN_EXIT", "ENTERED_CANNOT_EXIT", "SUSPENDED", "UNKNOWN"}
	csResp3PrimeNudgeStatuses = []string{"START", "COMPLETE", "UNKNOWN"}
)

// csResp3FromSequentialID is an enum fromId over ids 0..n-1 that falls back to its last constant, UNKNOWN.
func csResp3FromSequentialID(names []string, id int32) string {
	if id >= 0 && int(id) < len(names) {
		return names[id]
	}
	return names[len(names)-1]
}

func csResp3FromLongBitmask(flags []csResp3Flag, intMap int64) []string {
	out := []string{}
	for _, f := range flags {
		if intMap&f.bitmask != 0 {
			out = append(out, f.name)
		}
	}
	return out
}

func csResp3PumpFeatureTypesFromBitmask(intMap *big.Int) []string {
	out := []string{}
	for _, f := range csResp3PumpFeatureTypes {
		if new(big.Int).And(intMap, big.NewInt(f.bitmask)).Sign() != 0 {
			out = append(out, f.name)
		}
	}
	return out
}

func csResp3MalfunctionsFromBitmask(intMap *big.Int) []string {
	out := []string{}
	for bit, name := range csResp3MalfunctionTypes {
		if intMap.Bit(bit) != 0 {
			out = append(out, name)
		}
	}
	return out
}

// csResp3Jan12008Instant is Dates.fromJan12008EpochSecondsToDate, with Instant.ofEpochSecond's range check.
func csResp3Jan12008Instant(seconds int64) time.Time {
	const minSecond, maxSecond = -31557014167219200, 31556889864403199
	s := seconds + 1199145600
	if s < minSecond || s > maxSecond {
		throw("java.time.DateTimeException", "Instant exceeds minimum or maximum instant")
	}
	return time.Unix(s, 0).UTC()
}

func csResp3Reminder(cargo []byte, i int) string {
	frequency := readUint32(cargo, i)
	startTime := readShort(cargo, i+4)
	endTime := readShort(cargo, i+6)
	activeDays := sb(cargo[i+8])
	enabled := sb(cargo[i+9])
	validityStatus := sb(cargo[i+10])
	return jAutoToString("RemindersResponse.Reminder", [][2]string{
		{"activeDays", jInt(activeDays)},
		{"enabled", jInt(enabled)},
		{"endTime", jInt(endTime)},
		{"frequency", jLong(frequency)},
		{"startTime", jInt(startTime)},
		{"validityStatus", jInt(validityStatus)},
	})
}

func csResp3SetLastBolusStatus(m *Message, status, bolusID int32, timestamp, deliveredVolume int64, bolusStatusID, bolusSourceID, bolusTypeBitmask int32, extendedBolusDuration int64) {
	m.setInt("status", status)
	m.setInt("bolusId", bolusID)
	m.setLong("timestamp", timestamp)
	m.setLong("deliveredVolume", deliveredVolume)
	m.setInt("bolusStatusId", bolusStatusID)
	m.setInt("bolusSourceId", bolusSourceID)
	m.setInt("bolusTypeBitmask", bolusTypeBitmask)
	m.setLong("extendedBolusDuration", extendedBolusDuration)
}

func csResp3ParseLastBolusStatus(m *Message, raw []byte) {
	check(len(raw) == m.size())
	m.Cargo = raw
	csResp3SetLastBolusStatus(m, sb(raw[0]), readShort(raw, 1), readUint32(raw, 5), readUint32(raw, 9),
		sb(raw[13]), sb(raw[14]), sb(raw[15]), readUint32(raw, 16))
}

func csResp3LastBolusStatusCargo(status, bolusID int32, unknown []byte, timestamp, deliveredVolume int64, bolusStatusID, bolusSourceID, bolusTypeBitmask int32, extendedBolusDuration int64) []byte {
	return combine(
		[]byte{byte(status)},
		firstTwoBytesLittleEndian(bolusID),
		unknown,
		toUint32(timestamp),
		toUint32(deliveredVolume),
		[]byte{byte(bolusStatusID)},
		[]byte{byte(bolusSourceID)},
		[]byte{byte(bolusTypeBitmask)},
		toUint32(extendedBolusDuration),
	)
}

func csResp3SetPumpFeaturesV2(m *Message, status, supportedFeatureIndexID int32, pumpFeaturesBitmask int64) {
	m.setInt("status", status)
	m.setInt("supportedFeatureIndexId", supportedFeatureIndexID)
	m.setLong("pumpFeaturesBitmask", pumpFeaturesBitmask)
	sets := []struct {
		field string
		index int32
		flags []csResp3Flag
	}{
		{"primaryFeatures", 0, csResp3PumpFeatureTypes},
		{"controlIqProFeatures", 1, csResp3ControlIqProFeatureTypes},
		{"controlFeatures", 2, csResp3ControlFeatureTypes},
		{"controlIqFeatures", 3, csResp3ControlIqFeatureTypes},
		{"dexcomFeatures", 4, csResp3DexcomFeatureTypes},
		{"abbottFeatures", 5, csResp3AbbottFeatureTypes},
		{"secondaryFeatures", 6, csResp3SecondaryFeatureTypes},
	}
	for _, s := range sets {
		if supportedFeatureIndexID == s.index {
			m.setEnumSet(s.field, csResp3FromLongBitmask(s.flags, pumpFeaturesBitmask))
		} else {
			m.setNull(s.field)
		}
	}
}

func csResp3ParseLoadStatus(m *Message, raw []byte) {
	check(len(raw) == m.size())
	m.Cargo = raw
	isLoadingActiveID := ub(raw[0])
	m.setInt("isLoadingActiveId", isLoadingActiveID)
	m.setBool("isLoadingActive", isLoadingActiveID != 0)
	loadStateID := ub(raw[1])
	m.setInt("loadStateId", loadStateID)
	loadState := csResp3FromSequentialID(csResp3LoadStates, loadStateID)
	m.setObject("loadState", loadState)
	primeStatusID := ub(raw[2])
	m.setInt("primeStatusId", primeStatusID)
	if loadState == "PRIME_TUBING" {
		m.setObject("primeTubingStatus", csResp3FromSequentialID(csResp3PrimeTubingStatus, primeStatusID))
	} else {
		m.setNull("primeTubingStatus")
	}
	if loadState == "PRIME_NUDGE" {
		m.setObject("primeNudgeStatus", csResp3FromSequentialID(csResp3PrimeNudgeStatuses, primeStatusID))
	} else {
		m.setNull("primeNudgeStatus")
	}
}

func csResp3ParseMalfunctionBitmask(m *Message, raw []byte) {
	check(len(raw) == m.size())
	m.Cargo = raw
	bitmask := readUint64(raw, 0)
	m.setBigInt("bitmask", bitmask)
	m.setEnumSet("malfunctions", csResp3MalfunctionsFromBitmask(bitmask))
}

func csResp3SetLastBG(m *Message, bgTimestamp int64, bgValue, bgSourceID int32) {
	m.setLong("bgTimestamp", bgTimestamp)
	m.setInt("bgValue", bgValue)
	m.setInt("bgSourceId", bgSourceID)
	switch bgSourceID {
	case 0:
		m.setObject("bgSource", "MANUAL")
	case 1:
		m.setObject("bgSource", "CGM")
	default:
		m.setNull("bgSource")
	}
}

func init() {
	register("response.currentStatus.InsulinStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("currentInsulinAmount", readShort(raw, 0))
			m.setInt("isEstimate", sb(raw[2]))
			m.setInt("insulinLowAmount", sb(raw[3]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = combine(firstTwoBytesLittleEndian(a.Int(0)), []byte{byte(a.Int(1))}, []byte{byte(a.Int(2))})
				m.setInt("currentInsulinAmount", a.Int(0))
				m.setInt("isEstimate", a.Int(1))
				m.setInt("insulinLowAmount", a.Int(2))
			},
		},
	})

	register("response.currentStatus.LastBGResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			csResp3SetLastBG(m, readUint32(raw, 0), readShort(raw, 4), sb(raw[6]))
		},
		ctors: map[string]func(*Message, args){
			"long,int,int": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), firstTwoBytesLittleEndian(a.Int(1)), []byte{byte(a.Int(2))})
				csResp3SetLastBG(m, a.Long(0), a.Int(1), a.Int(2))
			},
		},
	})

	register("response.currentStatus.LastBolusStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			csResp3ParseLastBolusStatus(m, raw)
			m.setBytes("unknown", []byte{raw[3], raw[4]})
		},
		ctors: map[string]func(*Message, args){
			"int,int,long,long,int,int,int,long,byte[]": func(m *Message, a args) {
				unknown := a.Bytes(8)
				m.Cargo = csResp3LastBolusStatusCargo(a.Int(0), a.Int(1), unknown, a.Long(2), a.Long(3), a.Int(4), a.Int(5), a.Int(6), a.Long(7))
				csResp3SetLastBolusStatus(m, a.Int(0), a.Int(1), a.Long(2), a.Long(3), a.Int(4), a.Int(5), a.Int(6), a.Long(7))
				m.setBytes("unknown", unknown)
			},
		},
	})

	register("response.currentStatus.LastBolusStatusV2Response", classImpl{
		parse: func(m *Message, raw []byte) {
			csResp3ParseLastBolusStatus(m, raw)
			m.setLong("requestedVolume", readUint32(raw, 20))
		},
		ctors: map[string]func(*Message, args){
			"int,int,long,long,int,int,int,long,long": func(m *Message, a args) {
				m.Cargo = combine(
					csResp3LastBolusStatusCargo(a.Int(0), a.Int(1), []byte{0, 0}, a.Long(2), a.Long(3), a.Int(4), a.Int(5), a.Int(6), a.Long(7)),
					toUint32(a.Long(8)),
				)
				csResp3SetLastBolusStatus(m, a.Int(0), a.Int(1), a.Long(2), a.Long(3), a.Int(4), a.Int(5), a.Int(6), a.Long(7))
				m.setLong("requestedVolume", a.Long(8))
			},
		},
	})

	register("response.currentStatus.LastBolusStatusV3Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("lastBolusTypeBitmask", ub(raw[0]))
			m.setInt("standardBolusStatusId", ub(raw[1]))
			m.setInt("standardBolusId", readShort(raw, 2))
			m.setBytes("standardUnknown", []byte{raw[4], raw[5]})
			m.setLong("standardBolusTimestamp", readUint32(raw, 6))
			m.setLong("standardBolusDeliveredVolume", readUint32(raw, 10))
			m.setInt("standardBolusEndReasonId", ub(raw[14]))
			m.setInt("standardBolusSourceId", ub(raw[15]))
			m.setInt("standardBolusTypeBitmask", ub(raw[16]))
			m.setLong("standardBolusRequestedVolume", readUint32(raw, 17))
			m.setLong("standardBolusSecondsSincePumpReset", readUint32(raw, 21))
			m.setInt("extendedBolusStatusId", ub(raw[25]))
			m.setInt("extendedBolusId", readShort(raw, 26))
			m.setBytes("extendedUnknown", []byte{raw[28], raw[29]})
			m.setLong("extendedBolusTimestamp", readUint32(raw, 30))
			m.setLong("extendedBolusDeliveredVolume", readUint32(raw, 34))
			m.setInt("extendedBolusEndReasonId", ub(raw[38]))
			m.setInt("extendedBolusSourceId", ub(raw[39]))
			m.setInt("extendedBolusTypeBitmask", ub(raw[40]))
			m.setLong("extendedBolusRequestedVolume", readUint32(raw, 41))
			m.setLong("extendedBolusSecondsSincePumpReset", readUint32(raw, 45))
			m.setLong("extendedBolusDuration", readUint32(raw, 49))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,byte[],long,long,int,int,int,long,long,int,int,byte[],long,long,int,int,int,long,long,long": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					[]byte{byte(a.Int(1))},
					firstTwoBytesLittleEndian(a.Int(2)),
					a.Bytes(3),
					toUint32(a.Long(4)),
					toUint32(a.Long(5)),
					[]byte{byte(a.Int(6))},
					[]byte{byte(a.Int(7))},
					[]byte{byte(a.Int(8))},
					toUint32(a.Long(9)),
					toUint32(a.Long(10)),
					[]byte{byte(a.Int(11))},
					firstTwoBytesLittleEndian(a.Int(12)),
					a.Bytes(13),
					toUint32(a.Long(14)),
					toUint32(a.Long(15)),
					[]byte{byte(a.Int(16))},
					[]byte{byte(a.Int(17))},
					[]byte{byte(a.Int(18))},
					toUint32(a.Long(19)),
					toUint32(a.Long(20)),
					toUint32(a.Long(21)),
				)
				m.setInt("lastBolusTypeBitmask", a.Int(0))
				m.setInt("standardBolusStatusId", a.Int(1))
				m.setInt("standardBolusId", a.Int(2))
				m.setBytes("standardUnknown", a.Bytes(3))
				m.setLong("standardBolusTimestamp", a.Long(4))
				m.setLong("standardBolusDeliveredVolume", a.Long(5))
				m.setInt("standardBolusEndReasonId", a.Int(6))
				m.setInt("standardBolusSourceId", a.Int(7))
				m.setInt("standardBolusTypeBitmask", a.Int(8))
				m.setLong("standardBolusRequestedVolume", a.Long(9))
				m.setLong("standardBolusSecondsSincePumpReset", a.Long(10))
				m.setInt("extendedBolusStatusId", a.Int(11))
				m.setInt("extendedBolusId", a.Int(12))
				m.setBytes("extendedUnknown", a.Bytes(13))
				m.setLong("extendedBolusTimestamp", a.Long(14))
				m.setLong("extendedBolusDeliveredVolume", a.Long(15))
				m.setInt("extendedBolusEndReasonId", a.Int(16))
				m.setInt("extendedBolusSourceId", a.Int(17))
				m.setInt("extendedBolusTypeBitmask", a.Int(18))
				m.setLong("extendedBolusRequestedVolume", a.Long(19))
				m.setLong("extendedBolusSecondsSincePumpReset", a.Long(20))
				m.setLong("extendedBolusDuration", a.Long(21))
			},
		},
	})

	register("response.currentStatus.LoadStatusResponse", classImpl{
		parse: csResp3ParseLoadStatus,
		ctors: map[string]func(*Message, args){
			"":       func(m *Message, _ args) { m.Cargo = []byte{} },
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"boolean,int,int": func(m *Message, a args) {
				active := byte(0)
				if a.Bool(0) {
					active = 1
				}
				m.Cargo = []byte{active, byte(a.Int(1) & 0xFF), byte(a.Int(2) & 0xFF)}
				m.parseSelf(m.Cargo)
			},
			"int,int": func(m *Message, a args) {
				status, unknown := a.Int(0), a.Int(1)
				m.Cargo = []byte{byte(status & 0xFF), byte(unknown & 0xFF), byte((unknown >> 8) & 0xFF)}
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("response.currentStatus.LocalizationResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("glucoseUOM", sb(raw[0]))
			m.setInt("languageSelected", sb(raw[1]))
			m.setInt("regionSetting", sb(raw[2]))
			m.setLong("languagesAvailableBitmask", readUint32(raw, 3))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,long": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, []byte{byte(a.Int(1))}, []byte{byte(a.Int(2))}, toUint32(a.Long(3)))
				m.setInt("glucoseUOM", a.Int(0))
				m.setInt("languageSelected", a.Int(1))
				m.setInt("regionSetting", a.Int(2))
				m.setLong("languagesAvailableBitmask", a.Long(3))
			},
		},
	})

	register("response.currentStatus.MalfunctionBitmaskStatusResponse", classImpl{
		parse: csResp3ParseMalfunctionBitmask,
		ctors: map[string]func(*Message, args){
			"java.math.BigInteger": func(m *Message, a args) {
				m.Cargo = toUint64(lowBits(a.BigInt(0), 64))
				m.parseSelf(m.Cargo)
			},
			"long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)))
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("response.currentStatus.NonControlIQIOBResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.setLong("iob", readUint32(raw, 0))
			m.setLong("timeRemaining", readUint32(raw, 4))
			m.setLong("totalIOB", readUint32(raw, 8))
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"long,long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)), toUint32(a.Long(2)))
				m.setLong("iob", a.Long(0))
				m.setLong("timeRemaining", a.Long(1))
				m.setLong("totalIOB", a.Long(2))
			},
		},
	})

	profileStatusFields := []string{"numberOfProfiles", "idpSlot0Id", "idpSlot1Id", "idpSlot2Id", "idpSlot3Id", "idpSlot4Id", "idpSlot5Id", "activeSegmentIndex"}
	register("response.currentStatus.ProfileStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			for i, f := range profileStatusFields {
				m.setInt(f, sb(raw[i]))
			}
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,int,int,int,int,int": func(m *Message, a args) {
				cargo := make([]byte, len(profileStatusFields))
				for i := range profileStatusFields {
					cargo[i] = byte(a.Int(i))
				}
				m.Cargo = cargo
				for i, f := range profileStatusFields {
					m.setInt(f, a.Int(i))
				}
			},
		},
	})

	register("response.currentStatus.PumpFeaturesV1Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setEnumSet("features", csResp3PumpFeatureTypesFromBitmask(readUint64(raw, 0)))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"java.math.BigInteger": func(m *Message, a args) {
				intMap := a.BigInt(0)
				m.Cargo = toUint64(lowBits(intMap, 64))
				m.setEnumSet("features", csResp3PumpFeatureTypesFromBitmask(intMap))
			},
		},
	})

	register("response.currentStatus.PumpFeaturesV2Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			csResp3SetPumpFeaturesV2(m, sb(raw[0]), sb(raw[1]), readUint32(raw, 2))
		},
		ctors: map[string]func(*Message, args){
			"int,int,long": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, []byte{byte(a.Int(1))}, toUint32(a.Long(2)))
				csResp3SetPumpFeaturesV2(m, a.Int(0), a.Int(1), a.Long(2))
			},
		},
	})

	register("response.currentStatus.PumpGlobalsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("quickBolusEnabledRaw", sb(raw[0]))
			m.setInt("quickBolusIncrementUnits", readShort(raw, 1))
			m.setInt("quickBolusIncrementCarbs", readShort(raw, 3))
			m.setInt("quickBolusEntryType", sb(raw[5]))
			m.setInt("quickBolusStatus", sb(raw[6]))
			m.setInt("buttonAnnun", sb(raw[7]))
			m.setInt("quickBolusAnnun", sb(raw[8]))
			m.setInt("bolusAnnun", sb(raw[9]))
			m.setInt("reminderAnnun", sb(raw[10]))
			m.setInt("alertAnnun", sb(raw[11]))
			m.setInt("alarmAnnun", sb(raw[12]))
			m.setInt("fillTubingAnnun", sb(raw[13]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,int,int,int,int,int,int,int,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					firstTwoBytesLittleEndian(a.Int(1)),
					firstTwoBytesLittleEndian(a.Int(2)),
					[]byte{byte(a.Int(3)), byte(a.Int(4)), byte(a.Int(5)), byte(a.Int(6)), byte(a.Int(7)), byte(a.Int(8)), byte(a.Int(9)), byte(a.Int(10)), byte(a.Int(11))},
				)
				for i, f := range []string{"quickBolusEnabledRaw", "quickBolusIncrementUnits", "quickBolusIncrementCarbs", "quickBolusEntryType",
					"quickBolusStatus", "buttonAnnun", "quickBolusAnnun", "bolusAnnun", "reminderAnnun", "alertAnnun", "alarmAnnun", "fillTubingAnnun"} {
					m.setInt(f, a.Int(i))
				}
			},
		},
	})

	register("response.currentStatus.PumpSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("lowInsulinThreshold", sb(raw[0]))
			m.setInt("cannulaPrimeSize", sb(raw[1]))
			m.setInt("autoShutdownEnabled", sb(raw[2]))
			m.setInt("autoShutdownDuration", readShort(raw, 3))
			m.setInt("featureLock", sb(raw[5]))
			m.setInt("oledTimeout", sb(raw[6]))
			m.setInt("status", readShort(raw, 7))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int,int,int,int,int,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0)), byte(a.Int(1)), byte(a.Int(2))},
					firstTwoBytesLittleEndian(a.Int(3)),
					[]byte{byte(a.Int(4)), byte(a.Int(5))},
					firstTwoBytesLittleEndian(a.Int(6)),
				)
				m.setInt("lowInsulinThreshold", a.Int(0))
				m.setInt("cannulaPrimeSize", a.Int(1))
				m.setInt("autoShutdownEnabled", a.Int(2))
				m.setInt("autoShutdownDuration", a.Int(3))
				m.setInt("featureLock", a.Int(4))
				m.setInt("oledTimeout", a.Int(5))
				m.setInt("status", a.Int(6))
			},
		},
	})

	register("response.currentStatus.PumpVersionBResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setString("softwareName", readString(raw, 0, 20))
			m.setLong("configurationBitsA", readUint32(raw, 20))
			m.setLong("configurationBitsB", readUint32(raw, 24))
			m.setLong("serialNumber", readUint32(raw, 28))
			m.setLong("modelNumber", readUint32(raw, 32))
			m.setString("pumpRevision", readString(raw, 36, 8))
			m.setLong("pcbPartNumberA", readUint32(raw, 44))
			m.setLong("pcbSerialNumberA", readUint32(raw, 48))
			m.setString("pcbRevisionNumberA", readString(raw, 52, 8))
		},
		ctors: map[string]func(*Message, args){
			"java.lang.String,long,long,long,long,java.lang.String,long,long,java.lang.String": func(m *Message, a args) {
				m.Cargo = combine(
					writeString(a.String(0), 20),
					toUint32(a.Long(1)),
					toUint32(a.Long(2)),
					toUint32(a.Long(3)),
					toUint32(a.Long(4)),
					writeString(a.String(5), 8),
					toUint32(a.Long(6)),
					toUint32(a.Long(7)),
					writeString(a.String(8), 8),
				)
				m.setString("softwareName", a.String(0))
				m.setLong("configurationBitsA", a.Long(1))
				m.setLong("configurationBitsB", a.Long(2))
				m.setLong("serialNumber", a.Long(3))
				m.setLong("modelNumber", a.Long(4))
				m.setString("pumpRevision", a.String(5))
				m.setLong("pcbPartNumberA", a.Long(6))
				m.setLong("pcbSerialNumberA", a.Long(7))
				m.setString("pcbRevisionNumberA", a.String(8))
			},
		},
	})

	register("response.currentStatus.PumpVersionResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("armSwVer", readUint32(raw, 0))
			m.setLong("mspSwVer", readUint32(raw, 4))
			m.setLong("configABits", readUint32(raw, 8))
			m.setLong("configBBits", readUint32(raw, 12))
			m.setLong("serialNum", readUint32(raw, 16))
			m.setLong("partNum", readUint32(raw, 20))
			m.setString("pumpRev", readString(raw, 24, 8))
			m.setLong("pcbaSN", readUint32(raw, 32))
			m.setString("pcbaRev", readString(raw, 36, 8))
			m.setLong("modelNum", readUint32(raw, 44))
		},
		ctors: map[string]func(*Message, args){
			"long,long,long,long,long,long,java.lang.String,long,java.lang.String,long": func(m *Message, a args) {
				m.Cargo = combine(
					toUint32(a.Long(0)),
					toUint32(a.Long(1)),
					toUint32(a.Long(2)),
					toUint32(a.Long(3)),
					toUint32(a.Long(4)),
					toUint32(a.Long(5)),
					writeString(a.String(6), 8),
					toUint32(a.Long(7)),
					writeString(a.String(8), 8),
					toUint32(a.Long(9)),
				)
				m.setLong("armSwVer", a.Long(0))
				m.setLong("mspSwVer", a.Long(1))
				m.setLong("configABits", a.Long(2))
				m.setLong("configBBits", a.Long(3))
				m.setLong("serialNum", a.Long(4))
				m.setLong("partNum", a.Long(5))
				m.setString("pumpRev", a.String(6))
				m.setLong("pcbaSN", a.Long(7))
				m.setString("pcbaRev", a.String(8))
				m.setLong("modelNum", a.Long(9))
			},
		},
	})

	register("response.currentStatus.ReminderStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"java.math.BigInteger": func(m *Message, a args) {
				m.Cargo = toUint64(lowBits(a.BigInt(0), 64))
			},
		},
	})

	register("response.currentStatus.RemindersResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			for i, f := range []string{"lowBGReminder", "highBGReminder", "siteChangeReminder", "missedBolusReminder0", "missedBolusReminder1",
				"missedBolusReminder2", "missedBolusReminder3", "afterBolusReminder", "additionalBolusReminder"} {
				m.setObject(f, csResp3Reminder(raw, 11*i))
			}
			m.setInt("lowBGThreshold", readShort(raw, 99))
			m.setInt("highBGThreshold", readShort(raw, 101))
			m.setInt("siteChangeDays", sb(raw[103]))
			m.setInt("status", sb(raw[104]))
		},
	})

	register("response.currentStatus.SecretMenuResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("timeOfLastConnectionTimestampSeconds", readUint32(raw, 0))
			m.setLong("reservedValue", readUint32(raw, 4))
		},
		ctors: map[string]func(*Message, args){
			"long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)))
				m.setLong("timeOfLastConnectionTimestampSeconds", a.Long(0))
				m.setLong("reservedValue", a.Long(1))
			},
		},
	})

	register("response.currentStatus.StreamDataReadinessResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("freestyleLibre2ReadinessId", ub(raw[0]))
			m.setInt("streamDataTypeOrdinal", ub(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1))}
				m.setInt("freestyleLibre2ReadinessId", a.Int(0))
				m.setInt("streamDataTypeOrdinal", a.Int(1))
			},
		},
	})

	register("response.currentStatus.TempRateResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("active", raw[0] != 0)
			m.setInt("percentage", ub(raw[1]))
			m.setLong("startTimeRaw", readUint32(raw, 2))
			m.setLong("duration", readUint32(raw, 6))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"boolean,int,long,long": func(m *Message, a args) {
				active := byte(0)
				if a.Bool(0) {
					active = 1
				}
				m.Cargo = combine([]byte{active}, []byte{byte(a.Int(1))}, toUint32(a.Long(2)), toUint32(a.Long(3)))
				m.setBool("active", a.Bool(0))
				m.setInt("percentage", a.Int(1))
				m.setLong("startTimeRaw", a.Long(2))
				m.setLong("duration", a.Long(3))
			},
		},
	})

	register("response.currentStatus.TempRateStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("active", raw[0] != 0)
			m.setInt("tempRateId", readShort(raw, 1))
			m.setInt("percentage", ub(raw[3]))
			m.setLong("startTimeRaw", readUint32(raw, 4))
			m.setLong("secondsSincePumpReset", readUint32(raw, 8))
			m.setLong("duration", readUint32(raw, 12))
		},
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("response.currentStatus.TimeSinceResetResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			currentTime := readUint32(raw, 0)
			m.setLong("currentTime", currentTime)
			m.setInstant("currentTimeInstant", csResp3Jan12008Instant(currentTime))
			m.setLong("pumpTimeSinceReset", readUint32(raw, 4))
		},
		ctors: map[string]func(*Message, args){
			"long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)))
				m.setLong("currentTime", a.Long(0))
				m.setInstant("currentTimeInstant", csResp3Jan12008Instant(a.Long(0)))
				m.setLong("pumpTimeSinceReset", a.Long(1))
			},
		},
	})

	register("response.currentStatus.UnknownMobiOpcode110Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
		},
	})
}
