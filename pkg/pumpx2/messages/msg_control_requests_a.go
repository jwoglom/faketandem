package messages

import "time"

const ctlReqAJanuary12008UnixEpoch = 1199145600

func ctlReqAEmptyCargo(m *Message, _ args) { m.Cargo = []byte{} }

func ctlReqAEmptySizeChecked() classImpl {
	return classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){"": ctlReqAEmptyCargo},
	}
}

func ctlReqAEmptyAnySize() classImpl {
	return classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){"": ctlReqAEmptyCargo},
	}
}

func ctlReqABolusIDReserve() classImpl {
	build := func(m *Message, bolusID, reserve int32) {
		m.Cargo = combine(firstTwoBytesLittleEndian(bolusID), firstTwoBytesLittleEndian(reserve))
		m.setInt("bolusID", bolusID)
		m.setInt("reserve", reserve)
	}
	return classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("bolusID", readShort(raw, 0))
			m.setInt("reserve", readShort(raw, 2))
		},
		ctors: map[string]func(*Message, args){
			"int":     func(m *Message, a args) { build(m, a.Int(0), 0) },
			"int,int": func(m *Message, a args) { build(m, a.Int(0), a.Int(1)) },
		},
	}
}

func ctlReqABloodGlucoseReadingType(id int32) string {
	switch id {
	case 0:
		return "MANUAL"
	case 1:
		return "AUTO"
	}
	return "UNKNOWN"
}

func ctlReqABloodGlucoseReadingSource(id int32) string {
	switch id {
	case 0:
		return "PUMP"
	case 1:
		return "REMOTE"
	}
	return "UNKNOWN"
}

func ctlReqANotificationType(id int32) *string {
	names := []string{"REMINDER", "ALERT", "ALARM", "CGM_ALERT"}
	if id < 0 || int(id) >= len(names) {
		return nil
	}
	return &names[id]
}

func ctlReqABoolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func ctlReqACreateIDP(m *Message, profileName string, carbRatio, basalRate, targetBG, isf, insulinDuration, carbEntry, timeSegmentBitmask, bolusSettingsBitmask, sourceIdpID int32) {
	m.Cargo = combine(
		writeString(profileName, 17),
		toUint32(int64(carbRatio)),
		firstTwoBytesLittleEndian(0),
		firstTwoBytesLittleEndian(basalRate),
		firstTwoBytesLittleEndian(targetBG),
		firstTwoBytesLittleEndian(isf),
		firstTwoBytesLittleEndian(insulinDuration),
		[]byte{byte(timeSegmentBitmask), byte(bolusSettingsBitmask)},
		[]byte{byte(sourceIdpID)},
		[]byte{byte(carbEntry)},
	)
	m.setString("profileName", profileName)
	m.setInt("firstSegmentProfileCarbRatio", carbRatio)
	m.setInt("firstSegmentProfileStartTime", 0)
	m.setInt("firstSegmentProfileBasalRate", basalRate)
	m.setInt("firstSegmentProfileTargetBG", targetBG)
	m.setInt("firstSegmentProfileISF", isf)
	m.setInt("profileInsulinDuration", insulinDuration)
	m.setInt("timeSegmentBitmask", timeSegmentBitmask)
	m.setInt("bolusSettingsBitmask", bolusSettingsBitmask)
	m.setInt("profileCarbEntry", carbEntry)
	m.setInt("sourceIdpId", sourceIdpID)
}

func init() {
	register("request.control.ActivateShelfModeRequest", ctlReqAEmptyAnySize())
	register("request.control.PrimeTubingSuspendRequest", ctlReqAEmptyAnySize())

	register("request.control.BolusPermissionRequest", ctlReqAEmptySizeChecked())
	register("request.control.DisconnectPumpRequest", ctlReqAEmptySizeChecked())
	register("request.control.EnterChangeCartridgeModeRequest", ctlReqAEmptySizeChecked())
	register("request.control.EnterFillTubingModeRequest", ctlReqAEmptySizeChecked())
	register("request.control.ExitChangeCartridgeModeRequest", ctlReqAEmptySizeChecked())
	register("request.control.ExitFillTubingModeRequest", ctlReqAEmptySizeChecked())
	register("request.control.PlaySoundRequest", ctlReqAEmptySizeChecked())

	register("request.control.AdditionalBolusRequest", ctlReqABolusIDReserve())
	register("request.control.BolusPermissionReleaseRequest", ctlReqABolusIDReserve())

	register("request.control.CancelBolusRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("bolusId", readShort(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = combine(firstTwoBytesLittleEndian(a.Int(0)), []byte{0, 0})
				m.setInt("bolusId", a.Int(0))
			},
		},
	})

	register("request.control.CgmHighLowAlertRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("threshold", readShort(raw, 0))
			m.setInt("repeatDurationMinutes", readShort(raw, 2))
			m.setInt("alertType", ub(raw[4]))
			m.setBool("enableAlert", ub(raw[5]) != 0)
			m.setInt("bitmask", ub(raw[6]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int,boolean,int": func(m *Message, a args) {
				alertType, threshold, repeat, enable, bitmask := a.Int(0), a.Int(1), a.Int(2), a.Bool(3), a.Int(4)
				m.setInt("alertType", alertType)
				m.setInt("threshold", threshold)
				m.setInt("repeatDurationMinutes", repeat)
				m.setBool("enableAlert", enable)
				m.setInt("bitmask", bitmask)
				m.Cargo = combine(
					firstTwoBytesLittleEndian(threshold),
					firstTwoBytesLittleEndian(repeat),
					[]byte{byte(alertType)},
					[]byte{ctlReqABoolByte(enable)},
					[]byte{byte(bitmask)},
				)
			},
		},
	})

	register("request.control.CgmOutOfRangeAlertRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("enable", ub(raw[0]) != 0)
			m.setInt("alertDelay", ub(raw[1]))
			m.setInt("bitmask", ub(raw[2]))
		},
		ctors: map[string]func(*Message, args){
			"boolean,int,int": func(m *Message, a args) {
				enable, alertDelay, bitmask := a.Bool(0), a.Int(1), a.Int(2)
				m.setBool("enable", enable)
				m.setInt("alertDelay", alertDelay)
				m.setInt("bitmask", bitmask)
				m.Cargo = []byte{ctlReqABoolByte(enable), byte(alertDelay), byte(bitmask)}
			},
		},
	})

	register("request.control.CgmRiseFallAlertRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("alertType", ub(raw[0]))
			m.setBool("enable", ub(raw[1]) != 0)
			m.setInt("mgPerDl", ub(raw[2]))
			m.setInt("bitmask", ub(raw[3]))
		},
		ctors: map[string]func(*Message, args){
			"int,boolean,int,int": func(m *Message, a args) {
				alertType, enable, mgPerDl, bitmask := a.Int(0), a.Bool(1), a.Int(2), a.Int(3)
				m.setInt("alertType", alertType)
				m.setBool("enable", enable)
				m.setInt("mgPerDl", mgPerDl)
				m.setInt("bitmask", bitmask)
				m.Cargo = []byte{byte(alertType), ctlReqABoolByte(enable), byte(mgPerDl), byte(bitmask)}
			},
		},
	})

	register("request.control.ChangeControlIQSettingsRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("enabled", sb(raw[0]) == 1)
			m.setInt("weightLbs", readShort(raw, 1))
			m.setInt("totalDailyInsulinUnits", sb(raw[4]))
		},
		ctors: map[string]func(*Message, args){
			"":       ctlReqAEmptyCargo,
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"boolean,int,int": func(m *Message, a args) {
				m.parseSelf(combine(
					[]byte{ctlReqABoolByte(a.Bool(0))},
					firstTwoBytesLittleEndian(a.Int(1)),
					[]byte{1, byte(a.Int(2)), 1},
				))
			},
		},
	})

	register("request.control.ChangeTimeDateRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			tandemEpochTime := readUint32(raw, 0)
			m.setLong("tandemEpochTime", tandemEpochTime)
			m.setInstant("instant", time.Unix(tandemEpochTime+ctlReqAJanuary12008UnixEpoch, 0))
		},
		ctors: map[string]func(*Message, args){
			"":       ctlReqAEmptyCargo,
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"long": func(m *Message, a args) {
				m.Cargo = toUint32(a.Long(0))
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("request.control.CreateIDPRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setString("profileName", readString(raw, 0, 16))
			m.setInt("firstSegmentProfileCarbRatio", int32(readUint32(raw, 17)))
			m.setInt("firstSegmentProfileStartTime", readShort(raw, 21))
			m.setInt("firstSegmentProfileBasalRate", readShort(raw, 23))
			m.setInt("firstSegmentProfileTargetBG", readShort(raw, 25))
			m.setInt("firstSegmentProfileISF", readShort(raw, 27))
			m.setInt("profileInsulinDuration", readShort(raw, 29))
			m.setInt("timeSegmentBitmask", ub(raw[31]))
			m.setInt("bolusSettingsBitmask", ub(raw[32]))
			m.setInt("sourceIdpId", sb(raw[33]))
			m.setInt("profileCarbEntry", sb(raw[34]))
		},
		ctors: map[string]func(*Message, args){
			"java.lang.String,int": func(m *Message, a args) {
				ctlReqACreateIDP(m, a.String(0), 0, 0, 0, 0, 0, 0, 0, 0, a.Int(1))
			},
			"java.lang.String,int,int,int,int,int,int": func(m *Message, a args) {
				ctlReqACreateIDP(m, a.String(0), a.Int(1), a.Int(2), a.Int(3), a.Int(4), a.Int(5), a.Int(6), 31, 5, -1)
			},
			"java.lang.String,int,int,int,int,int,int,int,int": func(m *Message, a args) {
				ctlReqACreateIDP(m, a.String(0), a.Int(1), a.Int(2), a.Int(3), a.Int(4), a.Int(5), a.Int(6), a.Int(7), a.Int(8), -1)
			},
		},
	})

	register("request.control.DeleteIDPRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setInt("profileIndex", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), 0}
				m.setInt("idpId", a.Int(0))
				m.setInt("profileIndex", 0)
			},
			"int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1))}
				m.setInt("idpId", a.Int(0))
				m.setInt("profileIndex", a.Int(1))
			},
		},
	})

	register("request.control.DismissNotificationRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("notificationId", readUint32(raw, 0))
			typeID := ub(raw[4])
			m.setInt("notificationTypeId", typeID)
			m.setBool("executeExtraAction", ub(raw[5]) != 0)
			m.setObject("notificationType", jString(ctlReqANotificationType(typeID)))
		},
		ctors: map[string]func(*Message, args){
			"": ctlReqAEmptyCargo,
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("request.control.FactoryResetBRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			m.Cargo = raw
			m.setLong("key", readUint32(raw, 0))
			m.setLong("serialNumber", readUint32(raw, 4))
			m.setBool("enableShelfMode", raw[8] != 0)
		},
		ctors: map[string]func(*Message, args){
			"long,long,boolean": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)), []byte{ctlReqABoolByte(a.Bool(2))})
				m.setLong("key", a.Long(0))
				m.setLong("serialNumber", a.Long(1))
				m.setBool("enableShelfMode", a.Bool(2))
			},
		},
	})

	register("request.control.FactoryResetRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("key", readUint32(raw, 0))
			m.setLong("serialNumber", readUint32(raw, 4))
		},
		ctors: map[string]func(*Message, args){
			"long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)))
				m.setLong("key", a.Long(0))
				m.setLong("serialNumber", a.Long(1))
			},
		},
	})

	register("request.control.FillCannulaRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("primeSizeMilliUnits", readShort(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				primeSize := a.Int(0)
				checkf(primeSize > 0, "must have positive prime size")
				checkf(primeSize <= 3000, "cannot prime more than 3 units")
				m.Cargo = firstTwoBytesLittleEndian(primeSize)
				m.setInt("primeSizeMilliUnits", primeSize)
			},
		},
	})

	register("request.control.JoinFSL3Request", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBytes("bleAddress", copyOfRange(raw, 0, 6))
			m.setBytes("sensorCode", copyOfRange(raw, 6, 10))
			m.setBytes("activationTime", copyOfRange(raw, 10, 14))
			m.setBytes("firmwareVersion", copyOfRange(raw, 14, 18))
			m.setInt("warmupTime", ub(raw[18]))
			m.setBytes("compressedSerialNumber", copyOfRange(raw, 19, 28))
			m.setBytes("nfcUid", copyOfRange(raw, 28, 36))
		},
		ctors: map[string]func(*Message, args){
			"byte[],byte[],byte[],byte[],int,byte[],byte[]": func(m *Message, a args) {
				bleAddress, sensorCode, activationTime, firmwareVersion := a.Bytes(0), a.Bytes(1), a.Bytes(2), a.Bytes(3)
				compressedSerialNumber, nfcUID := a.Bytes(5), a.Bytes(6)
				check(len(bleAddress) == 6)
				check(len(sensorCode) == 4)
				check(len(activationTime) == 4)
				check(len(firmwareVersion) == 4)
				check(len(compressedSerialNumber) == 9)
				check(len(nfcUID) == 8)
				m.Cargo = combine(bleAddress, sensorCode, activationTime, firmwareVersion, []byte{byte(a.Int(4))}, compressedSerialNumber, nfcUID)
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("request.control.RemoteBgEntryRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("bg", readShort(raw, 0))
			m.setBool("useForCgmCalibration", sb(raw[2]) == 1)
			m.setObject("entryType", ctlReqABloodGlucoseReadingType(sb(raw[3])))
			m.setObject("source", ctlReqABloodGlucoseReadingSource(sb(raw[4])))
			m.setLong("pumpTime", readUint32(raw, 5))
			m.setInt("bolusId", readShort(raw, 9))
		},
		ctors: map[string]func(*Message, args){
			"int,boolean,boolean,long,int": func(m *Message, a args) {
				bg, useForCgmCalibration, pumpTime, bolusID := a.Int(0), a.Bool(1), a.Long(3), a.Int(4)
				var entryType, source int32 = 0, 0
				if a.Bool(2) {
					source = 1
				}
				m.Cargo = combine(
					firstTwoBytesLittleEndian(bg),
					[]byte{ctlReqABoolByte(useForCgmCalibration)},
					[]byte{byte(entryType)},
					[]byte{byte(source)},
					toUint32(pumpTime),
					firstTwoBytesLittleEndian(bolusID),
				)
				m.setInt("bg", bg)
				m.setBool("useForCgmCalibration", useForCgmCalibration)
				m.setObject("entryType", ctlReqABloodGlucoseReadingType(entryType))
				m.setObject("source", ctlReqABloodGlucoseReadingSource(source))
				m.setLong("pumpTime", pumpTime)
				m.setInt("bolusId", bolusID)
			},
		},
	})

	remoteCarbEntry := func(m *Message, carbs, unknown int32, pumpTime int64, bolusID int32) {
		checkf(pumpTime < ctlReqAJanuary12008UnixEpoch, "pumpTimeSecondsSinceBoot (%d) should be seconds since boot; a unix epoch was provided instead", pumpTime)
		m.Cargo = combine(
			firstTwoBytesLittleEndian(carbs),
			[]byte{byte(unknown)},
			toUint32(pumpTime),
			firstTwoBytesLittleEndian(bolusID),
		)
		m.setInt("carbs", carbs)
		m.setInt("unknown", unknown)
		m.setLong("pumpTime", pumpTime)
		m.setInt("bolusId", bolusID)
	}
	register("request.control.RemoteCarbEntryRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			checkf(len(raw) == m.size(), "size %d", len(raw))
			m.Cargo = raw
			m.setInt("carbs", readShort(raw, 0))
			m.setInt("unknown", sb(raw[2]))
			m.setLong("pumpTime", readUint32(raw, 3))
			m.setInt("bolusId", readShort(raw, 7))
		},
		ctors: map[string]func(*Message, args){
			"int,long,int": func(m *Message, a args) { remoteCarbEntry(m, a.Int(0), 1, a.Long(1), a.Int(2)) },
			"int,int,long,int": func(m *Message, a args) {
				remoteCarbEntry(m, a.Int(0), a.Int(1), a.Long(2), a.Int(3))
			},
		},
	})

	renameIDP := func(m *Message, idpID, profileIndex int32, profileName string) {
		m.Cargo = combine([]byte{byte(idpID)}, []byte{byte(profileIndex)}, writeString(profileName, 16), []byte{0})
		m.setInt("idpId", idpID)
		m.setInt("profileIndex", profileIndex)
		m.setString("profileName", profileName)
	}
	register("request.control.RenameIDPRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setInt("profileIndex", sb(raw[1]))
			m.setString("profileName", readString(raw, 2, 16))
		},
		ctors: map[string]func(*Message, args){
			"int,java.lang.String":     func(m *Message, a args) { renameIDP(m, a.Int(0), 0, a.String(1)) },
			"int,int,java.lang.String": func(m *Message, a args) { renameIDP(m, a.Int(0), a.Int(1), a.String(2)) },
		},
	})
}
