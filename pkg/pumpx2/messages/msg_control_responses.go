package messages

func init() {
	for _, name := range []string{"ActivateShelfModeResponse", "FactoryResetBResponse", "FactoryResetResponse"} {
		register("response.control."+name, classImpl{
			parse: func(m *Message, raw []byte) {
				raw = m.removeSignedRequestHmacBytes(raw)
				m.Cargo = raw
			},
		})
	}

	for _, name := range []string{
		"BolusPermissionReleaseResponse", "CgmHighLowAlertResponse", "CgmOutOfRangeAlertResponse",
		"CgmRiseFallAlertResponse", "ChangeTimeDateResponse", "DisconnectPumpResponse",
		"ExitChangeCartridgeModeResponse", "ExitFillTubingModeResponse", "FillCannulaResponse",
		"RemoteBgEntryResponse", "RemoteCarbEntryResponse", "SetActiveIDPResponse", "SetAutoOffAlertResponse",
		"SetBgReminderResponse", "SetLowInsulinAlertResponse", "SetMaxBasalLimitResponse",
		"SetMaxBolusLimitResponse", "SetMissedMealBolusReminderResponse", "SetPumpAlertSnoozeResponse",
		"SetPumpSoundsResponse", "SetRuntimeConfigResponse", "SetSiteChangeReminderResponse", "StartDexcomG6SensorSessionResponse",
		"SuspendPumpingResponse", "UserInteractionResponse",
	} {
		register("response.control."+name, ctlRespStatusImpl())
	}

	for _, name := range []string{"DismissNotificationResponse", "EnterFillTubingModeResponse", "SetG6TransmitterIdResponse", "StopDexcomCGMSensorSessionResponse"} {
		impl := ctlRespStatusImpl()
		impl.ctors[""] = ctlRespEmptyCargo
		register("response.control."+name, impl)
	}

	enterChangeCartridge := ctlRespStatusImpl()
	enterChangeCartridge.ctors["byte[]"] = func(m *Message, a args) {
		m.Cargo = a.Bytes(0)
		m.parseSelf(a.Bytes(0))
	}
	register("response.control.EnterChangeCartridgeModeResponse", enterChangeCartridge)

	for _, name := range []string{"ChangeControlIQSettingsResponse", "PlaySoundResponse"} {
		register("response.control."+name, classImpl{
			parse: ctlRespParseStatus,
			ctors: map[string]func(*Message, args){
				"":       ctlRespEmptyCargo,
				"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			},
		})
	}

	register("response.control.SetQuickBolusSettingsResponse", classImpl{
		parse: ctlRespParseStatus,
		ctors: map[string]func(*Message, args){
			"": ctlRespEmptyCargo,
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("response.control.SendTipsControlGenericTestResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", ub(raw[0]))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0))}
				m.setInt("status", a.Int(0))
			},
		},
	})

	register("response.control.SetDexcomG7PairingCodeResponse", classImpl{
		parse: ctlRespParseStatus,
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), 0}
				m.setInt("status", a.Int(0))
			},
		},
	})

	register("response.control.SetIDPSettingsResponse", classImpl{
		parse: ctlRespParseStatus,
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), 2}
				m.setInt("status", a.Int(0))
			},
		},
	})

	register("response.control.ResumePumpingResponse", classImpl{
		parse: ctlRespParseStatus,
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
			"int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0))}
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("response.control.SetModesResponse", classImpl{
		parse: ctlRespParseStatus,
		ctors: map[string]func(*Message, args){
			"int":    func(m *Message, a args) { m.Cargo = []byte{byte(a.Int(0))} },
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
		},
	})

	register("response.control.SetSleepScheduleResponse", classImpl{
		parse: ctlRespParseStatus,
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int":    func(m *Message, a args) { m.parseSelf([]byte{byte(a.Int(0))}) },
		},
	})

	register("response.control.AdditionalBolusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("bolusId", readShort(raw, 1))
			m.setInt("reserve", readShort(raw, 3))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, firstTwoBytesLittleEndian(a.Int(1)), firstTwoBytesLittleEndian(a.Int(2)))
				m.setInt("status", a.Int(0))
				m.setInt("bolusId", a.Int(1))
				m.setInt("reserve", a.Int(2))
			},
		},
	})

	register("response.control.BolusPermissionResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("bolusId", readShort(raw, 1))
			m.setInt("nackReasonId", sb(raw[5]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, firstTwoBytesLittleEndian(a.Int(1)), []byte{0, 0}, []byte{byte(a.Int(2))})
				m.setInt("status", a.Int(0))
				m.setInt("bolusId", a.Int(1))
				m.setInt("nackReasonId", a.Int(2))
			},
		},
	})

	register("response.control.CancelBolusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("statusId", sb(raw[0]))
			m.setInt("bolusId", readShort(raw, 1))
			m.setInt("reasonId", sb(raw[3]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, firstTwoBytesLittleEndian(a.Int(1)), []byte{byte(a.Int(2)), 0})
				m.setInt("statusId", a.Int(0))
				m.setInt("bolusId", a.Int(1))
				m.setInt("reasonId", a.Int(2))
			},
		},
	})

	register("response.control.CreateIDPResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("newIdpId", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
			"int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1))}
				m.setInt("status", a.Int(0))
				m.setInt("newIdpId", a.Int(1))
			},
		},
	})

	register("response.control.DeleteIDPResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("deletedIdpId", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), 0}
				m.setInt("status", a.Int(0))
			},
			"int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1))}
				m.setInt("status", a.Int(0))
				m.setInt("deletedIdpId", a.Int(1))
			},
		},
	})

	register("response.control.RenameIDPResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("numberOfProfiles", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, []byte{byte(a.Int(1))})
				m.setInt("status", a.Int(0))
				m.setInt("numberOfProfiles", a.Int(1))
			},
		},
	})

	register("response.control.SetIDPSegmentResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("unknown", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"": ctlRespEmptyCargo,
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("response.control.JoinFSL3Response", ctlRespStatusPair("statusType"))
	register("response.control.SetConfigurableFeatureResponse", ctlRespStatusPair("reason"))
	register("response.control.SetSensorTypeResponse", ctlRespStatusPair("statusAcknowledgement"))

	register("response.control.PrimeTubingSuspendResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("statusCode", sb(raw[0]))
			m.setInt("reserve", sb(raw[2]))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, []byte{0}, []byte{byte(a.Int(1))})
				m.setInt("statusCode", a.Int(0))
				m.setInt("reserve", a.Int(1))
			},
		},
	})

	register("response.control.SetTempRateResponse", classImpl{
		parse: ctlRespParseTempRate,
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int,int": func(m *Message, a args) {
				m.parseSelf(combine([]byte{byte(a.Int(0))}, firstTwoBytesLittleEndian(a.Int(1)), []byte{0}))
			},
		},
	})

	register("response.control.StopTempRateResponse", classImpl{
		parse: ctlRespParseTempRate,
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int,int": func(m *Message, a args) {
				m.parseSelf(combine([]byte{byte(a.Int(0))}, firstTwoBytesLittleEndian(a.Int(1))))
			},
		},
	})

	register("response.control.StreamDataPreflightResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("statusTypeId", ub(raw[1]))
			m.setInt("streamTypeId", ub(raw[2]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, []byte{byte(a.Int(1))}, []byte{byte(a.Int(2))})
				m.setInt("status", a.Int(0))
				m.setInt("statusTypeId", a.Int(1))
				m.setInt("streamTypeId", a.Int(2))
			},
		},
	})
}

func ctlRespEmptyCargo(m *Message, _ args) { m.Cargo = []byte{} }

func ctlRespParseStatus(m *Message, raw []byte) {
	raw = m.removeSignedRequestHmacBytes(raw)
	check(len(raw) == m.size())
	m.Cargo = raw
	m.setInt("status", sb(raw[0]))
}

func ctlRespStatusImpl() classImpl {
	return classImpl{
		parse: ctlRespParseStatus,
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0))}
				m.setInt("status", a.Int(0))
			},
		},
	}
}

// ctlRespStatusPair is a two-byte response of a signed status and an unsigned second field.
func ctlRespStatusPair(second string) classImpl {
	return classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt(second, ub(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = []byte{byte(a.Int(0)), byte(a.Int(1))}
				m.setInt("status", a.Int(0))
				m.setInt(second, a.Int(1))
			},
		},
	}
}

func ctlRespParseTempRate(m *Message, raw []byte) {
	raw = m.removeSignedRequestHmacBytes(raw)
	check(len(raw) == m.size())
	m.Cargo = raw
	m.setInt("status", sb(raw[0]))
	m.setInt("tempRateId", readShort(raw, 1))
}
