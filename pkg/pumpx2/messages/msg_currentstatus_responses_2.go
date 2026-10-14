package messages

import (
	"fmt"
	"strconv"
)

func init() {
	register("response.currentStatus.CommonSoftwareInfoResponse", classImpl{
		parse: csResp2CommonSoftwareInfoParse,
		ctors: map[string]func(*Message, args){
			"java.lang.String,long,long,long,java.lang.String,long": func(m *Message, a args) {
				m.Cargo = csResp2CommonSoftwareInfoCargo(a.String(0), a.Long(1), a.Long(2), a.Long(3), a.String(4), a.Long(5), nil, nil)
				m.setString("appSoftwareVersion", a.String(0))
				m.setLong("appSoftwarePartNumber", a.Long(1))
				m.setLong("appSoftwarePartDashNumber", a.Long(2))
				m.setLong("appSoftwarePartRevisionNumber", a.Long(3))
				m.setString("bootloaderVersion", a.String(4))
				m.setLong("bootloaderPartNumber", a.Long(5))
				m.setNull("bootloaderPartDashNumber")
				m.setNull("bootloaderPartRevisionNumber")
			},
			"int,java.lang.String,long,long,java.lang.String,long,long": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					writeString(a.String(1), 17),
					toUint32(a.Long(2)),
					toUint32(a.Long(3)),
					writeString(a.String(4), 17),
					toUint32(a.Long(5)),
					toUint32(a.Long(6)))
				csResp2CommonSoftwareInfoParse(m, m.Cargo)
			},
			"java.lang.String,long,long,long,java.lang.String,long,long,long": func(m *Message, a args) {
				dash, rev := a.Long(6), a.Long(7)
				m.Cargo = csResp2CommonSoftwareInfoCargo(a.String(0), a.Long(1), a.Long(2), a.Long(3), a.String(4), a.Long(5), &dash, &rev)
				m.setString("appSoftwareVersion", a.String(0))
				m.setLong("appSoftwarePartNumber", a.Long(1))
				m.setLong("appSoftwarePartDashNumber", a.Long(2))
				m.setLong("appSoftwarePartRevisionNumber", a.Long(3))
				m.setString("bootloaderVersion", a.String(4))
				m.setLong("bootloaderPartNumber", a.Long(5))
				m.setLong("bootloaderPartDashNumber", dash)
				m.setLong("bootloaderPartRevisionNumber", rev)
			},
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("response.currentStatus.ControlIQIOBResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.setLong("mudaliarIOB", readUint32(raw, 0))
			m.setLong("timeRemainingSeconds", readUint32(raw, 4))
			m.setLong("mudaliarTotalIOB", readUint32(raw, 8))
			m.setLong("swan6hrIOB", readUint32(raw, 12))
			m.setInt("iobType", sb(raw[16]))
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"long,long,long,long,int": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)), toUint32(a.Long(2)), toUint32(a.Long(3)), []byte{byte(a.Int(4))})
				m.setLong("mudaliarIOB", a.Long(0))
				m.setLong("timeRemainingSeconds", a.Long(1))
				m.setLong("mudaliarTotalIOB", a.Long(2))
				m.setLong("swan6hrIOB", a.Long(3))
				m.setInt("iobType", a.Int(4))
			},
		},
	})

	register("response.currentStatus.ControlIQInfoV1Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			csResp2ControlIQInfoParse(m, raw)
		},
		ctors: map[string]func(*Message, args){
			"boolean,int,int,int,int,int,int,int,int": func(m *Message, a args) {
				m.Cargo = csResp2ControlIQInfoCargo(a)
				csResp2ControlIQInfoSet(m, a)
			},
		},
	})

	register("response.currentStatus.ControlIQInfoV2Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			csResp2ControlIQInfoParse(m, raw)
			m.setInt("exerciseChoice", sb(raw[10]))
			m.setLong("exerciseDuration", readUint32(raw, 11))
			m.setLong("exerciseTimeRemaining", readUint32(raw, 15))
		},
		ctors: map[string]func(*Message, args){
			"boolean,int,int,int,int,int,int,int,int,int,long,long": func(m *Message, a args) {
				m.Cargo = combine(csResp2ControlIQInfoCargo(a), []byte{byte(a.Int(9))}, toUint32(a.Long(10)), toUint32(a.Long(11)))
				csResp2ControlIQInfoSet(m, a)
				m.setInt("exerciseChoice", a.Int(9))
				m.setLong("exerciseDuration", a.Long(10))
				m.setLong("exerciseTimeRemaining", a.Long(11))
			},
		},
	})

	register("response.currentStatus.ControlIQSleepScheduleResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setObject("schedule0", csResp2SleepSchedule(raw, 0))
			m.setObject("schedule1", csResp2SleepSchedule(raw, 6))
			m.setObject("schedule2", csResp2SleepSchedule(raw, 12))
			m.setObject("schedule3", csResp2SleepSchedule(raw, 18))
		},
	})

	register("response.currentStatus.CurrentActiveIdpValuesResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("currentCarbRatio", readUint32(raw, 0))
			m.setInt("currentTargetBg", readShort(raw, 5))
			m.setInt("currentInsulinDuration", readShort(raw, 6))
			m.setInt("currentIsf", readShort(raw, 8))
		},
		ctors: map[string]func(*Message, args){
			"long,int,int,int": func(m *Message, a args) {
				targetBg, duration := a.Int(1), a.Int(2)
				m.Cargo = combine(
					toUint32(a.Long(0)),
					[]byte{0},
					[]byte{byte(targetBg), byte(duration), byte(duration >> 8)},
					firstTwoBytesLittleEndian(a.Int(3)))
				m.setLong("currentCarbRatio", a.Long(0))
				m.setInt("currentTargetBg", targetBg)
				m.setInt("currentInsulinDuration", duration)
				m.setInt("currentIsf", a.Int(3))
			},
		},
	})

	register("response.currentStatus.CurrentBatteryV1Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("currentBatteryAbc", sb(raw[0]))
			m.setInt("currentBatteryIbc", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1))}
				m.setInt("currentBatteryAbc", a.Int(0))
				m.setInt("currentBatteryIbc", a.Int(1))
			},
		},
	})

	register("response.currentStatus.CurrentBatteryV2Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("currentBatteryAbc", sb(raw[0]))
			m.setInt("currentBatteryIbc", sb(raw[1]))
			m.setInt("chargingStatus", sb(raw[2]))
			m.setInt("unknown1", readShort(raw, 3))
			m.setInt("unknown2", readShort(raw, 5))
			m.setInt("unknown3", readShort(raw, 7))
			m.setInt("unknown4", readShort(raw, 9))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,int,int,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0)), byte(a.Int(1)), byte(a.Int(2))},
					firstTwoBytesLittleEndian(a.Int(3)),
					firstTwoBytesLittleEndian(a.Int(4)),
					firstTwoBytesLittleEndian(a.Int(5)),
					firstTwoBytesLittleEndian(a.Int(6)))
				m.setInt("currentBatteryAbc", a.Int(0))
				m.setInt("currentBatteryIbc", a.Int(1))
				m.setInt("chargingStatus", a.Int(2))
				m.setInt("unknown1", a.Int(3))
				m.setInt("unknown2", a.Int(4))
				m.setInt("unknown3", a.Int(5))
				m.setInt("unknown4", a.Int(6))
			},
		},
	})

	register("response.currentStatus.CurrentBolusStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("statusId", sb(raw[0]))
			m.setInt("bolusId", readShort(raw, 1))
			m.setLong("timestamp", readUint32(raw, 5))
			m.setLong("requestedVolume", readUint32(raw, 9))
			m.setInt("bolusSourceId", sb(raw[13]))
			m.setInt("bolusTypeBitmask", sb(raw[14]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,long,long,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					firstTwoBytesLittleEndian(a.Int(1)),
					[]byte{0, 0},
					toUint32(a.Long(2)),
					toUint32(a.Long(3)),
					[]byte{byte(a.Int(4))},
					[]byte{byte(a.Int(5))})
				m.setInt("statusId", a.Int(0))
				m.setInt("bolusId", a.Int(1))
				m.setLong("timestamp", a.Long(2))
				m.setLong("requestedVolume", a.Long(3))
				m.setInt("bolusSourceId", a.Int(4))
				m.setInt("bolusTypeBitmask", a.Int(5))
			},
		},
	})

	for _, name := range []string{"response.currentStatus.CurrentEGVGuiDataResponse", "response.currentStatus.CurrentEgvGuiDataV2Response"} {
		register(name, classImpl{
			parse: func(m *Message, raw []byte) {
				check(len(raw) == m.size())
				m.Cargo = raw
				m.setLong("bgReadingTimestampSeconds", readUint32(raw, 0))
				m.setInt("cgmReading", readShort(raw, 4))
				m.setInt("egvStatusId", sb(raw[6]))
				csResp2SetEnum(m, "egvStatus", csResp2EGVStatus, sb(raw[6]))
				m.setInt("trendRate", sb(raw[7]))
			},
			ctors: map[string]func(*Message, args){
				"long,int,int,int": func(m *Message, a args) {
					m.Cargo = combine(
						toUint32(a.Long(0)),
						firstTwoBytesLittleEndian(a.Int(1)),
						[]byte{byte(a.Int(2))},
						[]byte{byte(a.Int(3))})
					m.setLong("bgReadingTimestampSeconds", a.Long(0))
					m.setInt("cgmReading", a.Int(1))
					m.setInt("egvStatusId", a.Int(2))
					csResp2SetEnum(m, "egvStatus", csResp2EGVStatus, a.Int(2))
					m.setInt("trendRate", a.Int(3))
				},
			},
		})
	}

	register("response.currentStatus.ExtendedBolusStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("bolusStatus", sb(raw[0]))
			m.setInt("bolusId", readShort(raw, 1))
			m.setLong("timestamp", readUint32(raw, 5))
			m.setLong("requestedVolume", readUint32(raw, 9))
			m.setLong("duration", readUint32(raw, 13))
			m.setInt("bolusSource", sb(raw[17]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,long,long,long,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					firstTwoBytesLittleEndian(a.Int(1)),
					toUint32(a.Long(2)),
					toUint32(a.Long(3)),
					toUint32(a.Long(4)),
					[]byte{byte(a.Int(5))},
					[]byte{0, 0})
				m.setInt("bolusStatus", a.Int(0))
				m.setInt("bolusId", a.Int(1))
				m.setLong("timestamp", a.Long(2))
				m.setLong("requestedVolume", a.Long(3))
				m.setLong("duration", a.Long(4))
				m.setInt("bolusSource", a.Int(5))
			},
		},
	})

	register("response.currentStatus.ExtendedBolusStatusV2Response", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("bolusStatus", sb(raw[0]))
			m.setInt("bolusId", readShort(raw, 1))
			m.setLong("timestamp", readUint32(raw, 5))
			m.setLong("requestedVolume", readUint32(raw, 9))
			m.setLong("duration", readUint32(raw, 13))
			m.setInt("bolusSource", sb(raw[17]))
			m.setLong("secondsSincePumpReset", readUint32(raw, 18))
		},
		ctors: map[string]func(*Message, args){
			"int,int,long,long,long,int,long": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					firstTwoBytesLittleEndian(a.Int(1)),
					[]byte{0, 0},
					toUint32(a.Long(2)),
					toUint32(a.Long(3)),
					toUint32(a.Long(4)),
					[]byte{byte(a.Int(5))},
					toUint32(a.Long(6)))
				m.setInt("bolusStatus", a.Int(0))
				m.setInt("bolusId", a.Int(1))
				m.setLong("timestamp", a.Long(2))
				m.setLong("requestedVolume", a.Long(3))
				m.setLong("duration", a.Long(4))
				m.setInt("bolusSource", a.Int(5))
				m.setLong("secondsSincePumpReset", a.Long(6))
			},
		},
	})

	register("response.currentStatus.GetG6TransmitterHardwareInfoResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setString("transmitterFirmwareVersion", readString(raw, 0, 16))
			m.setString("transmitterHardwareRevision", readString(raw, 16, 16))
			m.setString("transmitterBleHardwareId", readString(raw, 32, 16))
			m.setString("transmitterSoftwareNumber", readString(raw, 48, 16))
			m.setString("transmitterPairingCode", readString(raw, 64, 16))
			m.setString("transmitterSerialNumber", readString(raw, 80, 16))
		},
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("response.currentStatus.GetSavedG7PairingCodeResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("pairingCode", readShort(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = combine(firstTwoBytesLittleEndian(a.Int(0)))
				m.setInt("pairingCode", a.Int(0))
			},
		},
	})

	register("response.currentStatus.GlobalMaxBolusSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("maxBolus", readShort(raw, 0))
			m.setInt("maxBolusDefault", readShort(raw, 2))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = combine(firstTwoBytesLittleEndian(a.Int(0)), firstTwoBytesLittleEndian(a.Int(1)))
				m.setInt("maxBolus", a.Int(0))
				m.setInt("maxBolusDefault", a.Int(1))
			},
		},
	})

	register("response.currentStatus.HighestAamResponse", classImpl{
		init: func(m *Message) {
			m.setObject("IGNORABLE_CODES", jLongArray2D(csResp2IgnorableAamCodes))
		},
		parse: csResp2HighestAamParse,
		ctors: map[string]func(*Message, args){
			"long,long,byte[]": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)), a.Bytes(2))
				csResp2HighestAamParse(m, m.Cargo)
			},
		},
	})

	register("response.currentStatus.HistoryLogResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("streamId", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1))}
				m.setInt("status", a.Int(0))
				m.setInt("streamId", a.Int(1))
			},
		},
	})

	register("response.currentStatus.HistoryLogStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("numEntries", readUint32(raw, 0))
			m.setLong("firstSequenceNum", readUint32(raw, 4))
			m.setLong("lastSequenceNum", readUint32(raw, 8))
		},
		ctors: map[string]func(*Message, args){
			"long,long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)), toUint32(a.Long(2)))
				m.setLong("numEntries", a.Long(0))
				m.setLong("firstSequenceNum", a.Long(1))
				m.setLong("lastSequenceNum", a.Long(2))
			},
		},
	})

	register("response.currentStatus.HomeScreenMirrorResponse", classImpl{
		parse: csResp2HomeScreenMirrorParse,
		ctors: map[string]func(*Message, args){
			"int,int,int,int,int,int,int,boolean,boolean": func(m *Message, a args) {
				m.Cargo = []byte{
					byte(a.Int(0)), byte(a.Int(1)), byte(a.Int(2)), byte(a.Int(3)),
					byte(a.Int(4)), byte(a.Int(5)), byte(a.Int(6)),
					csResp2BoolByte(a.Bool(7)), csResp2BoolByte(a.Bool(8)),
				}
				csResp2HomeScreenMirrorParse(m, m.Cargo)
			},
		},
	})

	register("response.currentStatus.IDPSegmentResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setInt("segmentIndex", sb(raw[1]))
			m.setInt("profileStartTime", readShort(raw, 2))
			m.setObject("profileProcessedStartTime", csResp2MinsTime(readShort(raw, 2)))
			m.setInt("profileBasalRate", readShort(raw, 4))
			m.setLong("profileCarbRatio", readUint32(raw, 6))
			m.setInt("profileTargetBG", readShort(raw, 10))
			m.setInt("profileISF", readShort(raw, 12))
			m.setInt("idpStatusId", sb(raw[14]))
			m.setEnumSet("idpStatus", csResp2IDPSegmentStatus(sb(raw[14])))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,int,long,int,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					[]byte{byte(a.Int(1))},
					firstTwoBytesLittleEndian(a.Int(2)),
					firstTwoBytesLittleEndian(a.Int(3)),
					toUint32(a.Long(4)),
					firstTwoBytesLittleEndian(a.Int(5)),
					firstTwoBytesLittleEndian(a.Int(6)),
					[]byte{byte(a.Int(7))})
				m.setInt("idpId", a.Int(0))
				m.setInt("segmentIndex", a.Int(1))
				m.setInt("profileStartTime", a.Int(2))
				m.setObject("profileProcessedStartTime", csResp2MinsTime(a.Int(2)))
				m.setInt("profileBasalRate", a.Int(3))
				m.setLong("profileCarbRatio", a.Long(4))
				m.setInt("profileTargetBG", a.Int(5))
				m.setInt("profileISF", a.Int(6))
				m.setInt("idpStatusId", a.Int(7))
				m.setEnumSet("idpStatus", csResp2IDPSegmentStatus(a.Int(7)))
			},
		},
	})

	register("response.currentStatus.IDPSettingsResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setString("name", readString(raw, 1, 16))
			m.setInt("numberOfProfileSegments", sb(raw[17]))
			m.setInt("insulinDuration", readShort(raw, 18))
			m.setInt("maxBolus", readShort(raw, 20))
			m.setBool("carbEntry", raw[22] != 0)
		},
		ctors: map[string]func(*Message, args){
			"int,java.lang.String,int,int,int,boolean": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					writeString(a.String(1), 16),
					[]byte{byte(a.Int(2))},
					firstTwoBytesLittleEndian(a.Int(3)),
					firstTwoBytesLittleEndian(a.Int(4)),
					[]byte{csResp2BoolByte(a.Bool(5))})
				m.setInt("idpId", a.Int(0))
				m.setString("name", a.String(1))
				m.setInt("numberOfProfileSegments", a.Int(2))
				m.setInt("insulinDuration", a.Int(3))
				m.setInt("maxBolus", a.Int(4))
				m.setBool("carbEntry", a.Bool(5))
			},
		},
	})
}

func csResp2BoolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// csResp2Enum is an id-keyed enum's constants in declaration order; fromId returns the first match.
type csResp2Enum []struct {
	id   int32
	name string
}

func (e csResp2Enum) fromID(id int32) (string, bool) {
	for _, c := range e {
		if c.id == id {
			return c.name, true
		}
	}
	return "", false
}

func csResp2SetEnum(m *Message, field string, e csResp2Enum, id int32) {
	if name, ok := e.fromID(id); ok {
		m.setObject(field, name)
	} else {
		m.setNull(field)
	}
}

func csResp2CommonSoftwareInfoCargo(appVersion string, appPart, appDash, appRev int64, bootVersion string, bootPart int64, bootDash, bootRev *int64) []byte {
	tail := []byte{}
	if bootDash != nil || bootRev != nil {
		tail = combine(toUint32(*bootDash), toUint32(*bootRev))
	}
	return combine(
		writeString(appVersion, 18),
		toUint32(appPart),
		toUint32(appDash),
		toUint32(appRev),
		writeString(bootVersion, 17),
		toUint32(bootPart),
		tail)
}

func csResp2CommonSoftwareInfoParse(m *Message, raw []byte) {
	const mobiSize = 51
	m.Cargo = raw
	if len(raw) == mobiSize {
		m.setInt("mcuTypeId", ub(raw[0]))
		m.setString("appSoftwareVersion", readString(raw, 1, 17))
		m.setLong("appSoftwarePartNumber", readUint32(raw, 18))
		m.setLong("appSoftwarePartDashNumber", 0)
		m.setLong("appSoftwarePartRevisionNumber", readUint32(raw, 22))
		m.setString("bootloaderVersion", readString(raw, 26, 17))
		m.setLong("bootloaderPartNumber", readUint32(raw, 43))
		m.setNull("bootloaderPartDashNumber")
		m.setLong("bootloaderPartRevisionNumber", readUint32(raw, 47))
		return
	}
	m.setNull("mcuTypeId")
	m.setString("appSoftwareVersion", readString(raw, 0, 18))
	m.setLong("appSoftwarePartNumber", readUint32(raw, 18))
	m.setLong("appSoftwarePartDashNumber", readUint32(raw, 22))
	m.setLong("appSoftwarePartRevisionNumber", readUint32(raw, 26))
	m.setString("bootloaderVersion", readString(raw, 30, 17))
	m.setLong("bootloaderPartNumber", readUint32(raw, 47))
	if len(raw) >= 59 {
		m.setLong("bootloaderPartDashNumber", readUint32(raw, 51))
		m.setLong("bootloaderPartRevisionNumber", readUint32(raw, 55))
	}
}

func csResp2ControlIQInfoParse(m *Message, raw []byte) {
	m.setBool("closedLoopEnabled", raw[0] != 0)
	m.setInt("weight", readShort(raw, 1))
	m.setInt("weightUnit", sb(raw[3]))
	m.setInt("totalDailyInsulin", sb(raw[4]))
	m.setInt("currentUserModeType", sb(raw[5]))
	m.setInt("byte6", sb(raw[6]))
	m.setInt("byte7", sb(raw[7]))
	m.setInt("byte8", sb(raw[8]))
	m.setInt("controlStateType", sb(raw[9]))
}

func csResp2ControlIQInfoCargo(a args) []byte {
	return combine(
		[]byte{csResp2BoolByte(a.Bool(0))},
		firstTwoBytesLittleEndian(a.Int(1)),
		[]byte{byte(a.Int(2)), byte(a.Int(3)), byte(a.Int(4)), byte(a.Int(5)), byte(a.Int(6)), byte(a.Int(7)), byte(a.Int(8))})
}

func csResp2ControlIQInfoSet(m *Message, a args) {
	m.setBool("closedLoopEnabled", a.Bool(0))
	m.setInt("weight", a.Int(1))
	m.setInt("weightUnit", a.Int(2))
	m.setInt("totalDailyInsulin", a.Int(3))
	m.setInt("currentUserModeType", a.Int(4))
	m.setInt("byte6", a.Int(5))
	m.setInt("byte7", a.Int(6))
	m.setInt("byte8", a.Int(7))
	m.setInt("controlStateType", a.Int(8))
}

func csResp2SleepSchedule(cargo []byte, i int) string {
	enabled := sb(cargo[i])
	activeDays := sb(cargo[i+1])
	startTime := readShort(cargo, i+2)
	endTime := readShort(cargo, i+4)
	return jAutoToString("ControlIQSleepScheduleResponse.SleepSchedule", [][2]string{
		{"activeDays", jInt(activeDays)},
		{"enabled", jInt(enabled)},
		{"endTime", jInt(endTime)},
		{"startTime", jInt(startTime)},
	})
}

var csResp2EGVStatus = csResp2Enum{
	{0, "INVALID"}, {1, "VALID"}, {2, "LOW"}, {3, "HIGH"}, {4, "UNAVAILABLE"},
}

var csResp2IgnorableAamCodes = [][]int64{{0, 0}, {3, 8230}, {18, 8311}, {26, 8322}}

func csResp2HighestAamParse(m *Message, raw []byte) {
	checkf(len(raw) == m.size() || len(raw) == m.size()-1, "Unexpected HighestAamResponse payload length: %d", len(raw))
	m.Cargo = raw
	aamID := readUint32(raw, 0)
	faultID := readUint32(raw, 4)
	m.setLong("aamId", aamID)
	m.setLong("faultId", faultID)
	m.setBytes("remaining", dropFirstN(raw, 8))
	m.setString("errorString", csResp2AamErrorString(aamID, faultID))
}

func csResp2AamErrorString(aamID, faultID int64) string {
	for _, code := range csResp2IgnorableAamCodes {
		if code[0] == aamID && code[1] == faultID {
			return ""
		}
	}
	return fmt.Sprintf("%d-0x%s", aamID, strconv.FormatInt(faultID, 16))
}

var (
	csResp2CGMTrendIcon = csResp2Enum{
		{0, "NO_ARROW"}, {1, "ARROW_DOUBLE_UP"}, {2, "ARROW_UP"}, {3, "ARROW_UP_RIGHT"},
		{4, "ARROW_FLAT"}, {5, "ARROW_DOWN_RIGHT"}, {6, "ARROW_DOWN"}, {7, "ARROW_DOUBLE_DOWN"},
	}
	csResp2CGMAlertIcon = csResp2Enum{
		{0, "STARTUP_1"}, {1, "STARTUP_2"}, {2, "STARTUP_3"}, {3, "STARTUP_4"}, {4, "CALIBRATE"},
		{5, "STARTUP_CALIBRATE"}, {6, "CHECKMARK_BLOOD_DROP"}, {7, "ERROR_HIGH_WEDGE"}, {8, "ERROR_LOW_WEDGE"},
		{11, "REPLACE_SENSOR"}, {12, "REPLACE_TRANSMITTER"}, {13, "OUT_OF_RANGE"}, {14, "FAILED_SENSOR"},
		{15, "TRIPLE_DASHES"}, {16, "LOW"}, {17, "HIGH"}, {255, "NO_ERROR"},
	}
	csResp2StatusIcon0 = csResp2Enum{
		{0, "AAM"}, {1, "CGM_TRANSMITTER"}, {4, "CGM_TRANSMITTER_OOR"}, {200, "HIDE_ICON"},
	}
	csResp2StatusIcon1 = csResp2Enum{
		{0, "AAM"}, {1, "BLOOD_DROP"}, {200, "HIDE_ICON"},
	}
	csResp2BolusStatusIcon = csResp2Enum{
		{0, "BOLUS"}, {1, "AUTO_BOLUS"}, {2, "BLACK_LOCK_SECURITY_PIN"},
		{3, "GREY_LOCK_SECURITY_PIN_CGM_GRAPH_ACTIVE"}, {200, "HIDE_ICON"},
	}
	csResp2BasalStatusIcon = csResp2Enum{
		{0, "BASAL"}, {1, "ZERO_BASAL"}, {2, "TEMP_RATE"}, {3, "ZERO_TEMP_RATE"}, {4, "SUSPEND"},
		{5, "HYPO_SUSPEND_BASAL_IQ"}, {6, "INCREASE_BASAL"}, {7, "ATTENUATED_BASAL"}, {200, "HIDE"},
	}
	csResp2ApControlStateIcon = csResp2Enum{
		{0, "STATE_GRAY"}, {1, "STATE_GRAY_RED_BIQ_CIQ_BASAL_SUSPENDED"}, {2, "STATE_GRAY_BLUE_CIQ_INCREASE_BASAL"},
		{3, "STATE_GRAY_ORANGE_CIQ_ATTENUATION_BASAL"}, {200, "HIDE_ICON"},
	}
)

func csResp2HomeScreenMirrorParse(m *Message, raw []byte) {
	check(len(raw) == m.size())
	m.Cargo = raw
	m.setInt("cgmTrendIconId", ub(raw[0]))
	m.setInt("cgmAlertIconId", ub(raw[1]))
	m.setInt("statusIcon0Id", ub(raw[2]))
	m.setInt("statusIcon1Id", ub(raw[3]))
	m.setInt("bolusStatusIconId", ub(raw[4]))
	m.setInt("basalStatusIconId", ub(raw[5]))
	m.setInt("apControlStateIconId", ub(raw[6]))
	m.setBool("remainingInsulinPlusIcon", raw[7] != 0)
	m.setBool("cgmDisplayData", raw[8] != 0)
	csResp2SetEnum(m, "cgmTrendIcon", csResp2CGMTrendIcon, ub(raw[0]))
	csResp2SetEnum(m, "cgmAlertIcon", csResp2CGMAlertIcon, ub(raw[1]))
	csResp2SetEnum(m, "statusIcon0", csResp2StatusIcon0, ub(raw[2]))
	csResp2SetEnum(m, "statusIcon1", csResp2StatusIcon1, ub(raw[3]))
	csResp2SetEnum(m, "bolusStatusIcon", csResp2BolusStatusIcon, ub(raw[4]))
	csResp2SetEnum(m, "basalStatusIcon", csResp2BasalStatusIcon, ub(raw[5]))
	csResp2SetEnum(m, "apControlStateIcon", csResp2ApControlStateIcon, ub(raw[6]))
}

// csResp2MinsTime is new MinsTime(totalMins).toString().
func csResp2MinsTime(totalMins int32) string {
	return fmt.Sprintf("%02d:%02d", totalMins/60, totalMins%60)
}

var csResp2IDPSegmentStatusValues = csResp2Enum{
	{1, "BASAL_RATE"}, {2, "CARB_RATIO"}, {4, "TARGET_BG"}, {8, "CORRECTION_FACTOR"}, {16, "START_TIME"},
}

func csResp2IDPSegmentStatus(mask int32) []string {
	items := []string{}
	for _, s := range csResp2IDPSegmentStatusValues {
		if mask&s.id != 0 {
			items = append(items, s.name)
		}
	}
	return items
}
