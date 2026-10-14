package messages

func csReqEmptyCargo(m *Message, _ args) { m.Cargo = []byte{} }

func csReqParseSized(m *Message, raw []byte) {
	check(len(raw) == m.size())
	m.Cargo = raw
}

func csReqParseAny(m *Message, raw []byte) { m.Cargo = raw }

// csReqDeviceType is CgmSupportPackageStatusRequest.DeviceType.fromId.
func csReqDeviceType(id int32) string {
	if id == 0 {
		return "DEXCOM_G7_ICGM"
	}
	return "UNKNOWN"
}

// csReqMCUType is CommonSoftwareInfoRequest.MCUType.fromId.
func csReqMCUType(id int32) string {
	switch id {
	case 0:
		return "ARM"
	case 1:
		return "BLE"
	case 2:
		return "MSP"
	}
	return "UNKNOWN"
}

// csReqStreamDataType is StreamDataReadinessResponse.StreamDataType.fromOrdinal.
func csReqStreamDataType(ordinal int32) string {
	if ordinal == 0 {
		return "FREESTYLE_LIBRE_2_SENSOR_CONTEXT"
	}
	return "UNKNOWN"
}

func init() {
	for _, name := range []string{
		"BasalIQAlertInfoRequest", "BasalIQSettingsRequest", "BasalIQStatusRequest", "BasalLimitSettingsRequest",
		"BolusCalcDataSnapshotRequest", "CGMAlertStatusRequest", "CGMGlucoseAlertSettingsRequest",
		"CGMOORAlertSettingsRequest", "CGMRateAlertSettingsRequest", "CGMStatusCRequest", "CGMStatusRequest",
		"CgmStatusV2Request", "ControlIQInfoV1Request", "ControlIQInfoV2Request", "ControlIQSleepScheduleRequest",
		"CurrentBatteryV1Request", "CurrentBatteryV2Request", "CurrentBolusStatusRequest", "CurrentEGVGuiDataRequest",
		"ExtendedBolusStatusRequest", "GetG6TransmitterHardwareInfoRequest", "GetSavedG7PairingCodeRequest",
		"GlobalMaxBolusSettingsRequest", "HighestAamRequest", "HistoryLogStatusRequest", "HomeScreenMirrorRequest",
		"InsulinStatusRequest", "LastBGRequest", "LastBolusStatusRequest", "LastBolusStatusV2Request",
		"LoadStatusRequest", "LocalizationRequest", "MalfunctionStatusRequest", "ProfileStatusRequest",
		"PumpFeaturesV1Request", "PumpGlobalsRequest", "PumpSettingsRequest", "PumpVersionRequest",
		"ReminderStatusRequest", "RemindersRequest", "TempRateRequest", "TempRateStatusRequest",
		"TimeSinceResetRequest", "UnknownMobiOpcode110Request",
	} {
		register("request.currentStatus."+name, classImpl{
			parse: csReqParseSized,
			ctors: map[string]func(*Message, args){"": csReqEmptyCargo},
		})
	}

	for _, name := range []string{
		"AlarmStatusRequest", "AlertStatusRequest", "BleSoftwareInfoRequest", "CGMHardwareInfoRequest",
		"ControlIQIOBRequest", "CurrentActiveIdpValuesRequest", "CurrentEgvGuiDataV2Request",
		"ExtendedBolusStatusV2Request", "LastBolusStatusV3Request", "NonControlIQIOBRequest",
		"PumpVersionBRequest", "SecretMenuRequest",
	} {
		register("request.currentStatus."+name, classImpl{
			parse: csReqParseAny,
			ctors: map[string]func(*Message, args){"": csReqEmptyCargo},
		})
	}

	register("request.currentStatus.ActiveAamBitsRequest", classImpl{
		parse: csReqParseSized,
		ctors: map[string]func(*Message, args){
			"": csReqEmptyCargo,
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	register("request.currentStatus.PumpFeaturesV2Request", classImpl{
		parse: csReqParseSized,
		ctors: map[string]func(*Message, args){
			"":    func(m *Message, _ args) { m.Cargo = firstByteLittleEndian(2) },
			"int": func(m *Message, a args) { m.Cargo = firstByteLittleEndian(a.Int(0)) },
		},
	})

	register("request.currentStatus.BolusPermissionChangeReasonRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("bolusId", readShort(raw, 0))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = combine(firstTwoBytesLittleEndian(a.Int(0)))
				m.setInt("bolusId", a.Int(0))
			},
		},
	})

	cgmSupportPackageStatus := func(m *Message, deviceTypeID int32) {
		m.Cargo = []byte{byte(deviceTypeID)}
		m.setInt("deviceTypeId", deviceTypeID)
		m.setObject("deviceType", csReqDeviceType(deviceTypeID))
	}
	register("request.currentStatus.CgmSupportPackageStatusRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("deviceTypeId", ub(raw[0]))
			m.setObject("deviceType", csReqDeviceType(ub(raw[0])))
		},
		ctors: map[string]func(*Message, args){
			"":    func(m *Message, _ args) { cgmSupportPackageStatus(m, 0) },
			"int": func(m *Message, a args) { cgmSupportPackageStatus(m, a.Int(0)) },
		},
	})

	commonSoftwareInfo := func(m *Message, mcuTypeID int32) {
		m.Cargo = []byte{byte(mcuTypeID)}
		m.setInt("mcuTypeId", mcuTypeID)
		m.setObject("mcuType", csReqMCUType(mcuTypeID))
	}
	register("request.currentStatus.CommonSoftwareInfoRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("mcuTypeId", ub(raw[0]))
			m.setObject("mcuType", csReqMCUType(ub(raw[0])))
		},
		ctors: map[string]func(*Message, args){
			"":       func(m *Message, _ args) { commonSoftwareInfo(m, 0) },
			"int":    func(m *Message, a args) { commonSoftwareInfo(m, a.Int(0)) },
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
		},
	})

	register("request.currentStatus.HistoryLogRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("startLog", readUint32(raw, 0))
			m.setInt("numberOfLogs", sb(raw[4]))
		},
		ctors: map[string]func(*Message, args){
			"long,int": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), []byte{byte(a.Int(1))})
				m.setLong("startLog", a.Long(0))
				m.setInt("numberOfLogs", a.Int(1))
			},
		},
	})

	register("request.currentStatus.IDPSegmentRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
			m.setInt("segmentIndex", sb(raw[1]))
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, []byte{byte(a.Int(1))})
				m.setInt("idpId", a.Int(0))
				m.setInt("segmentIndex", a.Int(1))
			},
		},
	})

	register("request.currentStatus.IDPSettingsRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("idpId", sb(raw[0]))
		},
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = combine(firstByteLittleEndian(a.Int(0)))
				m.setInt("idpId", a.Int(0))
			},
		},
	})

	streamDataReadiness := func(m *Message, streamDataTypeID int32) {
		m.Cargo = []byte{byte(streamDataTypeID)}
		m.setInt("streamDataTypeId", streamDataTypeID)
		m.setObject("streamDataType", csReqStreamDataType(streamDataTypeID))
	}
	register("request.currentStatus.StreamDataReadinessRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("streamDataTypeId", ub(raw[0]))
			m.setObject("streamDataType", csReqStreamDataType(ub(raw[0])))
		},
		ctors: map[string]func(*Message, args){
			"":    func(m *Message, _ args) { streamDataReadiness(m, 0) },
			"int": func(m *Message, a args) { streamDataReadiness(m, a.Int(0)) },
		},
	})
}
