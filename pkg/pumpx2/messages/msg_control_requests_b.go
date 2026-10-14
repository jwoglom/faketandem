package messages

func init() {
	register("request.control.ResumePumpingRequest", classImpl{
		parse: ctlReqBParseEmpty,
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
		},
	})

	register("request.control.SendTipsControlGenericTestRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("param1", readUint32(raw, 0))
			m.setLong("param2", readUint32(raw, 4))
			m.setLong("param3", readUint32(raw, 8))
			m.setLong("param4", readUint32(raw, 12))
			m.setLong("param5", readUint32(raw, 16))
			m.setLong("param6", readUint32(raw, 20))
		},
		ctors: map[string]func(*Message, args){
			"long,long,long,long,long,long": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)), toUint32(a.Long(2)), toUint32(a.Long(3)), toUint32(a.Long(4)), toUint32(a.Long(5)))
				m.setLong("param1", a.Long(0))
				m.setLong("param2", a.Long(1))
				m.setLong("param3", a.Long(2))
				m.setLong("param4", a.Long(3))
				m.setLong("param5", a.Long(4))
				m.setLong("param6", a.Long(5))
			},
		},
	})

	register("request.control.SetActiveIDPRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setInt("profileIndex", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"byte[]":  func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int":     func(m *Message, a args) { ctlReqBSetActiveIDP(m, a.Int(0), 0) },
			"int,int": func(m *Message, a args) { ctlReqBSetActiveIDP(m, a.Int(0), a.Int(1)) },
		},
	})

	register("request.control.SetAutoOffAlertRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("enableAutoOff", ub(raw[0]) != 0)
			m.setInt("autoOffDuration", readShort(raw, 1))
			m.setInt("bitmask", ub(raw[3]))
		},
		ctors: map[string]func(*Message, args){
			"boolean,int,int": func(m *Message, a args) {
				m.setBool("enableAutoOff", a.Bool(0))
				m.setInt("autoOffDuration", a.Int(1))
				m.setInt("bitmask", a.Int(2))
				m.Cargo = combine([]byte{ctlReqBBool(a.Bool(0))}, firstTwoBytesLittleEndian(a.Int(1)), []byte{byte(a.Int(2))})
			},
		},
	})

	register("request.control.SetBgReminderRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("reminderType", ub(raw[0]))
			m.setBool("enabledBGReminder", ub(raw[1]) != 0)
			m.setInt("reminderThreshold", readShort(raw, 2))
			m.setLong("reminderMinutes", readUint32(raw, 4))
			m.setInt("bitmask", ub(raw[8]))
		},
		ctors: map[string]func(*Message, args){
			"int,boolean,int,long,int": func(m *Message, a args) {
				m.setInt("reminderType", a.Int(0))
				m.setBool("enabledBGReminder", a.Bool(1))
				m.setInt("reminderThreshold", a.Int(2))
				m.setLong("reminderMinutes", a.Long(3))
				m.setInt("bitmask", a.Int(4))
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					[]byte{ctlReqBBool(a.Bool(1))},
					firstTwoBytesLittleEndian(a.Int(2)),
					toUint32(a.Long(3)),
					[]byte{byte(a.Int(4))},
				)
			},
		},
	})

	register("request.control.SetConfigurableFeatureRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("groupIndex", ub(raw[0]))
			m.setInt("featureIndex", ub(raw[1]))
			m.setBool("featureState", ub(raw[2]) != 0)
		},
		ctors: map[string]func(*Message, args){
			"int,int,boolean": func(m *Message, a args) {
				m.setInt("groupIndex", a.Int(0))
				m.setInt("featureIndex", a.Int(1))
				m.setBool("featureState", a.Bool(2))
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1)), ctlReqBBool(a.Bool(2))}
			},
		},
	})

	register("request.control.SetDexcomG7PairingCodeRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("pairingCode", readShort(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = combine(firstTwoBytesLittleEndian(a.Int(0)), make([]byte, 6))
				m.setInt("pairingCode", a.Int(0))
			},
		},
	})

	register("request.control.SetG6TransmitterIdRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setString("txId", readString(raw, 0, 6))
		},
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
			"java.lang.String": func(m *Message, a args) {
				m.Cargo = combine(writeString(a.String(0), 6), make([]byte, 10))
				m.setString("txId", a.String(0))
			},
		},
	})

	register("request.control.SetIDPSegmentRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setInt("profileIndex", sb(raw[1]))
			m.setInt("segmentIndex", sb(raw[2]))
			operationID := sb(raw[3])
			m.setInt("operationId", operationID)
			ctlReqBSetEnum(m, "operation", ctlReqBFromID(ctlReqBIDPSegmentOperations, operationID))
			m.setInt("profileStartTime", readShort(raw, 4))
			m.setInt("profileBasalRate", readShort(raw, 6))
			m.setLong("profileCarbRatio", readUint32(raw, 8))
			m.setInt("profileTargetBG", readShort(raw, 12))
			m.setInt("profileISF", readShort(raw, 14))
			idpStatusID := sb(raw[16])
			m.setInt("idpStatusId", idpStatusID)
			m.setEnumSet("idpStatus", ctlReqBFromBitmask(ctlReqBIDPSegmentStatuses, idpStatusID))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
		},
	})

	register("request.control.SetIDPSettingsRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setInt("profileIndex", sb(raw[1]))
			m.setInt("profileInsulinDuration", readShort(raw, 2))
			m.setInt("profileCarbEntry", sb(raw[4]))
			changeTypeID := sb(raw[5])
			m.setInt("changeTypeId", changeTypeID)
			ctlReqBSetEnum(m, "changeType", ctlReqBFromID(ctlReqBIDPChangeTypes, changeTypeID))
		},
	})

	register("request.control.SetLowInsulinAlertRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("insulinThreshold", ub(raw[0]))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.setInt("insulinThreshold", a.Int(0))
				m.Cargo = []byte{byte(a.Int(0))}
			},
		},
	})

	register("request.control.SetMaxBasalLimitRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("maxHourlyBasalMilliunits", readUint32(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"long": func(m *Message, a args) {
				limit := a.Long(0)
				checkf(limit >= 1000, "hourly basal limit must be greater than 1 unit")
				checkf(limit <= 15000, "basal limit must be less than or equal to 15 units")
				m.Cargo = toUint32(limit)
				m.setLong("maxHourlyBasalMilliunits", limit)
			},
		},
	})

	register("request.control.SetMaxBolusLimitRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("maxBolusMilliunits", readShort(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				limit := a.Int(0)
				checkf(limit >= 1000, "bolus limit must be greater than 1 unit")
				checkf(limit <= 25000, "bolus limit must be less than or equal to 25 units")
				m.Cargo = firstTwoBytesLittleEndian(limit)
				m.setInt("maxBolusMilliunits", limit)
			},
		},
	})

	register("request.control.SetMissedMealBolusReminderRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("reminderIndex", ub(raw[0]))
			m.setBool("enabledReminder", ub(raw[1]) != 0)
			m.setInt("startTime", readShort(raw, 2))
			m.setInt("endTime", readShort(raw, 4))
			m.setInt("selectedDays", ub(raw[6]))
			m.setInt("bitmask", ub(raw[7]))
		},
		ctors: map[string]func(*Message, args){
			"int,boolean,int,int,int,int": func(m *Message, a args) {
				m.setInt("reminderIndex", a.Int(0))
				m.setBool("enabledReminder", a.Bool(1))
				m.setInt("startTime", a.Int(2))
				m.setInt("endTime", a.Int(3))
				m.setInt("selectedDays", a.Int(4))
				m.setInt("bitmask", a.Int(5))
				m.Cargo = combine(
					[]byte{byte(a.Int(0))},
					[]byte{ctlReqBBool(a.Bool(1))},
					firstTwoBytesLittleEndian(a.Int(2)),
					firstTwoBytesLittleEndian(a.Int(3)),
					[]byte{byte(a.Int(4))},
					[]byte{byte(a.Int(5))},
				)
			},
		},
	})

	register("request.control.SetModesRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			checkf(len(raw) == m.size(), "got %d", len(raw))
			m.Cargo = raw
			bitmap := sb(raw[0])
			m.setInt("bitmap", bitmap)
			ctlReqBSetEnum(m, "command", ctlReqBFromID(ctlReqBModeCommands, bitmap))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) { m.parseSelf([]byte{byte(a.Int(0))}) },
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("request.control.SetPumpAlertSnoozeRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("snoozeEnabled", sb(raw[0]) == 1)
			m.setInt("snoozeDurationMins", sb(raw[1])%255)
		},
		ctors: map[string]func(*Message, args){
			"boolean,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{ctlReqBBool(a.Bool(0))}, []byte{byte(a.Int(1))})
				m.setBool("snoozeEnabled", a.Bool(0))
				m.setInt("snoozeDurationMins", a.Int(1))
			},
		},
	})

	register("request.control.SetPumpSoundsRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			quickBolus, general, reminder, alert, alarm := sb(raw[1]), sb(raw[2]), sb(raw[3]), sb(raw[4]), sb(raw[5])
			cgmA, cgmB, changeBitmask := sb(raw[6]), sb(raw[7]), sb(raw[8])
			m.setInt("quickBolusAnnunRaw", quickBolus)
			m.setInt("generalAnnunRaw", general)
			m.setInt("reminderAnnunRaw", reminder)
			m.setInt("alertAnnunRaw", alert)
			m.setInt("alarmAnnunRaw", alarm)
			m.setInt("cgmAlertAnnunA", cgmA)
			m.setInt("cgmAlertAnnunB", cgmB)
			m.setInt("changeBitmaskRaw", changeBitmask)
			ctlReqBSetEnum(m, "quickBolusAnnun", ctlReqBFromID(ctlReqBAnnunciations, quickBolus))
			ctlReqBSetEnum(m, "generalAnnun", ctlReqBFromID(ctlReqBAnnunciations, general))
			ctlReqBSetEnum(m, "reminderAnnun", ctlReqBFromID(ctlReqBAnnunciations, reminder))
			ctlReqBSetEnum(m, "alertAnnun", ctlReqBFromID(ctlReqBAnnunciations, alert))
			ctlReqBSetEnum(m, "alarmAnnun", ctlReqBFromID(ctlReqBAnnunciations, alarm))
			cgmAlert := ""
			for _, e := range ctlReqBCgmAlertAnnunciations {
				if e.idA == cgmA && e.idB == cgmB {
					cgmAlert = e.name
					break
				}
			}
			ctlReqBSetEnum(m, "cgmAlertAnnun", cgmAlert)
			m.setEnumSet("changeBitmask", ctlReqBFromBitmask(ctlReqBPumpSoundsChangeBitmasks, changeBitmask))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int,int,int,int,int,int,int,int,int": func(m *Message, a args) {
				ctlReqBSetPumpSounds(m, a.Int(0), a.Int(1), a.Int(2), a.Int(3), a.Int(4), a.Int(5), a.Int(6), a.Int(7), a.Int(8))
			},
			"int,int,int,int,int,int,int,int": func(m *Message, a args) {
				ctlReqBSetPumpSounds(m, 0, a.Int(0), a.Int(1), a.Int(2), a.Int(3), a.Int(4), a.Int(5), a.Int(6), a.Int(7))
			},
		},
	})

	register("request.control.SetQuickBolusSettingsRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("enabled", sb(raw[0]) == 1)
			m.setInt("modeRaw", sb(raw[1]))
			m.setInt("incrementUnits", readShort(raw, 2))
			m.setInt("incrementCarbs", readShort(raw, 4))
			m.setInt("changedFieldsRaw", ub(raw[6]))
		},
		ctors: map[string]func(*Message, args){
			"":       func(m *Message, _ args) { m.Cargo = []byte{} },
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"boolean,int,int,int,int": func(m *Message, a args) {
				m.Cargo = combine(
					[]byte{ctlReqBBool(a.Bool(0))},
					[]byte{byte(a.Int(1))},
					firstTwoBytesLittleEndian(a.Int(2)),
					firstTwoBytesLittleEndian(a.Int(3)),
					[]byte{byte(a.Int(4))},
				)
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("request.control.SetRuntimeConfigRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("config0", ub(raw[0]))
			m.setInt("config1", ub(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"":        func(m *Message, _ args) { ctlReqBSetRuntimeConfig(m, 0, 0) },
			"int,int": func(m *Message, a args) { ctlReqBSetRuntimeConfig(m, a.Int(0), a.Int(1)) },
		},
	})

	register("request.control.SetSensorTypeRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("cgmSensorType", ub(raw[0]))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.setInt("cgmSensorType", a.Int(0))
				m.Cargo = []byte{byte(a.Int(0))}
			},
		},
	})

	register("request.control.SetSiteChangeReminderRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setBool("enable", ub(raw[0]) != 0)
			m.setInt("dayCount", ub(raw[1]))
			m.setLong("timeOfDayMinutes", readUint32(raw, 2))
			m.setInt("bitmask", ub(raw[6]))
		},
		ctors: map[string]func(*Message, args){
			"boolean,int,long,int": func(m *Message, a args) {
				m.setBool("enable", a.Bool(0))
				m.setInt("dayCount", a.Int(1))
				m.setLong("timeOfDayMinutes", a.Long(2))
				m.setInt("bitmask", a.Int(3))
				m.Cargo = combine(
					[]byte{ctlReqBBool(a.Bool(0))},
					[]byte{byte(a.Int(1))},
					toUint32(a.Long(2)),
					[]byte{byte(a.Int(3))},
				)
			},
		},
	})

	register("request.control.SetSleepScheduleRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("slot", sb(raw[0]))
			m.setBytes("rawSchedule", dropLastN(dropFirstN(raw, 1), 1))
			m.setInt("flag", sb(raw[7]))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int,byte[],int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, a.Bytes(1), []byte{byte(a.Int(2))})
			},
		},
	})

	register("request.control.SetTempRateRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("minutes", int32(readUint32(raw, 0)/1000/60))
			m.setInt("percent", readShort(raw, 4))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int,int": func(m *Message, a args) {
				minutes, percent := a.Int(0), a.Int(1)
				checkf(minutes >= 15 && minutes <= 72*60, "duration of temp rate must be between 15 and 4,320 minutes (72 hours)")
				checkf(percent >= 0 && percent <= 250, "percent temp rate must be between 0-250%%")
				m.Cargo = combine(toUint32(int64(minutes)*1000*60), firstTwoBytesLittleEndian(percent))
				m.setInt("minutes", minutes)
				m.setInt("percent", percent)
			},
		},
	})

	register("request.control.StartDexcomG6SensorSessionRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("sensorCode", readShort(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) {
				m.Cargo = firstTwoBytesLittleEndian(0)
				m.setInt("sensorCode", 0)
			},
			"int": func(m *Message, a args) {
				m.Cargo = firstTwoBytesLittleEndian(a.Int(0))
				m.setInt("sensorCode", a.Int(0))
			},
		},
	})

	register("request.control.StopDexcomCGMSensorSessionRequest", classImpl{
		parse: ctlReqBParseEmpty,
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
		},
	})

	register("request.control.StopTempRateRequest", classImpl{
		parse: ctlReqBParseEmpty,
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
		},
	})

	register("request.control.StreamDataPreflightRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			m.Cargo = raw
			m.setInt("streamType", ub(raw[0]))
			m.setInt("length", readShort(raw, 1))
			m.setBytes("hmac", copyOfRange(raw, 3, len(raw)))
		},
		ctors: map[string]func(*Message, args){
			"int,int,byte[]": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, firstTwoBytesLittleEndian(a.Int(1)), a.Bytes(2))
				m.setInt("streamType", a.Int(0))
				m.setInt("length", a.Int(1))
				m.setBytes("hmac", a.Bytes(2))
			},
		},
	})

	register("request.control.SuspendPumpingRequest", classImpl{
		parse: ctlReqBParseEmpty,
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
		},
	})

	register("request.control.UserInteractionRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			m.Cargo = m.removeSignedRequestHmacBytes(raw)
		},
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
		},
	})
}

func ctlReqBParseEmpty(m *Message, raw []byte) {
	raw = m.removeSignedRequestHmacBytes(raw)
	check(len(raw) == m.size())
	m.Cargo = raw
}

func ctlReqBBool(v bool) byte {
	if v {
		return 1
	}
	return 0
}

type ctlReqBEnum struct {
	name string
	id   int32
}

// ctlReqBFromID is an enum's fromId: the first constant with that id, or "" for null.
func ctlReqBFromID(values []ctlReqBEnum, id int32) string {
	for _, e := range values {
		if e.id == id {
			return e.name
		}
	}
	return ""
}

func ctlReqBFromBitmask(values []ctlReqBEnum, mask int32) []string {
	out := []string{}
	for _, e := range values {
		if mask&e.id != 0 {
			out = append(out, e.name)
		}
	}
	return out
}

func ctlReqBSetEnum(m *Message, field, name string) {
	if name == "" {
		m.setNull(field)
		return
	}
	m.setObject(field, name)
}

var (
	ctlReqBIDPSegmentOperations = []ctlReqBEnum{{"MODIFY_SEGMENT_ID", 0}, {"CREATE_SEGMENT", 1}, {"DELETE_SEGMENT_ID", 2}}
	ctlReqBIDPSegmentStatuses   = []ctlReqBEnum{{"BASAL_RATE", 1}, {"CARB_RATIO", 2}, {"TARGET_BG", 4}, {"CORRECTION_FACTOR", 8}, {"START_TIME", 16}}
	ctlReqBIDPChangeTypes       = []ctlReqBEnum{{"CHANGE_INSULIN_DURATION", 1}, {"CHANGE_CARB_ENTRY", 4}}
	ctlReqBModeCommands         = []ctlReqBEnum{
		{"SLEEP_MODE_ON", 1}, {"SLEEP_MODE_OFF", 2}, {"EXERCISE_MODE_ON", 3}, {"EXERCISE_MODE_OFF", 4},
		{"NOT_SUPPORTED_IN_CURRENT_FIRMWARE__STOP_ALL", 5},
		{"NOT_SUPPORTED_IN_CURRENT_FIRMWARE__START_EATING_SOON", 6},
		{"NOT_SUPPORTED_IN_CURRENT_FIRMWARE__STOP_EATING_SOON", 7},
	}
	ctlReqBAnnunciations            = []ctlReqBEnum{{"AUDIO_HIGH", 0}, {"AUDIO_MEDIUM", 1}, {"AUDIO_LOW", 2}, {"VIBRATE", 3}}
	ctlReqBPumpSoundsChangeBitmasks = []ctlReqBEnum{{"QUICK_BOLUS", 2}, {"GENERAL", 4}, {"REMINDER", 8}, {"ALERT", 16}, {"ALARM", 32}, {"CGM_ALERT", 64}}
	ctlReqBCgmAlertAnnunciations    = []struct {
		name     string
		idA, idB int32
	}{{"VIBRATE", 0, 0}, {"BEEP", 0, 2}, {"HYPO_REPEAT", 0, 3}}
)

func ctlReqBSetActiveIDP(m *Message, idpID, profileIndex int32) {
	m.Cargo = []byte{byte(idpID), byte(profileIndex)}
	m.setInt("idpId", idpID)
	m.setInt("profileIndex", profileIndex)
}

func ctlReqBSetPumpSounds(m *Message, firstByteUnknown, quickBolus, general, reminder, alert, alarm, cgmA, cgmB, changeBitmask int32) {
	m.Cargo = []byte{byte(firstByteUnknown), byte(quickBolus), byte(general), byte(reminder), byte(alert), byte(alarm), byte(cgmA), byte(cgmB), byte(changeBitmask)}
	m.parseSelf(m.Cargo)
}

func ctlReqBSetRuntimeConfig(m *Message, config0, config1 int32) {
	m.setInt("config0", config0)
	m.setInt("config1", config1)
	m.Cargo = []byte{byte(config0), byte(config1)}
}
