package settings

import log "github.com/sirupsen/logrus"

// registerConstant is a helper to register a constant-mode default with error logging
func registerConstant(manager *Manager, name string, value map[string]interface{}) {
	if err := manager.RegisterDefault(name, &ResponseConfig{
		Mode:  ModeConstant,
		Value: value,
	}); err != nil {
		log.Warnf("Failed to register %s: %v", name, err)
	}
}

// RegisterDefaults registers default configurations for common message types
func RegisterDefaults(manager *Manager) {
	registerSettingsDefaults(manager)
	registerPollingDefaults(manager)
	registerQualifyingEventDefaults(manager)
	registerNotificationDefaults(manager)
	registerControlIQDefaults(manager)
	registerBolusAndBasalDefaults(manager)
	registerGlobalSettingsDefaults(manager)
	registerPumpInfoDefaults(manager)
	registerAdditionalStatusDefaults(manager)

	log.Info("Registered default settings configurations")
}

// registerSettingsDefaults registers defaults for settings-related messages
func registerSettingsDefaults(manager *Manager) {
	// BasalIQSettingsResponse(int hypoMinimization, int suspendAlert, int resumeAlert)
	registerConstant(manager, "BasalIQSettingsRequest", map[string]interface{}{
		"hypoMinimization": 1,
		"suspendAlert":     1,
		"resumeAlert":      1,
	})

	// ControlIQSettingsRequest, TherapySettingsGlobalsRequest, and
	// ControlIQGlobalsRequest were removed: none of these have a real pumpX2
	// counterpart (no matching Java class anywhere in the library), so a real
	// Tandem app can never send a message that decodes to these names.

	// PumpGlobalsResponse(int quickBolusEnabledRaw, int quickBolusIncrementUnits,
	// int quickBolusIncrementCarbs, int quickBolusEntryType, int quickBolusStatus,
	// int buttonAnnun, int quickBolusAnnun, int bolusAnnun, int reminderAnnun,
	// int alertAnnun, int alarmAnnun, int fillTubingAnnun)
	registerConstant(manager, "PumpGlobalsRequest", map[string]interface{}{
		"quickBolusEnabledRaw":     1,
		"quickBolusIncrementUnits": 100,
		"quickBolusIncrementCarbs": 1,
		"quickBolusEntryType":      0,
		"quickBolusStatus":         0,
		"buttonAnnun":              0,
		"quickBolusAnnun":          0,
		"bolusAnnun":               0,
		"reminderAnnun":            0,
		"alertAnnun":               0,
		"alarmAnnun":               0,
		"fillTubingAnnun":          0,
	})
}

// registerPollingDefaults registers defaults for messages controlX2 polls every 5 min
func registerPollingDefaults(manager *Manager) {
	// CurrentBatteryV2Response(int currentBatteryAbc, int currentBatteryIbc,
	// int chargingStatus, int unknown1, int unknown2, int unknown3, int unknown4)
	registerConstant(manager, "CurrentBatteryV2Request", map[string]interface{}{
		"currentBatteryAbc": 85,
		"currentBatteryIbc": 85,
		"chargingStatus":    0,
		"unknown1":          0,
		"unknown2":          0,
		"unknown3":          0,
		"unknown4":          0,
	})

	// ControlIQIOBResponse(long mudaliarIOB, long timeRemainingSeconds,
	// long mudaliarTotalIOB, long swan6hrIOB, int iobType)
	registerConstant(manager, "ControlIQIOBRequest", map[string]interface{}{
		"mudaliarIOB":          250, // 2.5 units * 100
		"timeRemainingSeconds": 0,
		"mudaliarTotalIOB":     250,
		"swan6hrIOB":           0,
		"iobType":              0,
	})

	// InsulinStatusResponse(int currentInsulinAmount, int isEstimate, int insulinLowAmount)
	registerConstant(manager, "InsulinStatusRequest", map[string]interface{}{
		"currentInsulinAmount": 20000, // 200.0 units * 100
		"isEstimate":           0,
		"insulinLowAmount":     0,
	})
}

// registerQualifyingEventDefaults registers defaults for qualifying event handler messages
func registerQualifyingEventDefaults(manager *Manager) {
	// CurrentBasalStatusResponse(long profileBasalRate, long currentBasalRate, int basalModifiedBitmask)
	registerConstant(manager, "CurrentBasalStatusRequest", map[string]interface{}{
		"profileBasalRate":     85, // 0.85 U/hr * 100
		"currentBasalRate":     85,
		"basalModifiedBitmask": 0,
	})

	// CurrentBolusStatusResponse(int statusId, int bolusId, long timestamp,
	// long requestedVolume, int bolusSourceId, int bolusTypeBitmask)
	registerConstant(manager, "CurrentBolusStatusRequest", map[string]interface{}{
		"statusId":         0,
		"bolusId":          0,
		"timestamp":        0,
		"requestedVolume":  0,
		"bolusSourceId":    0,
		"bolusTypeBitmask": 0,
	})

	// CurrentEGVGuiDataResponse(long bgReadingTimestampSeconds, int cgmReading,
	// int egvStatusId, int trendRate)
	registerConstant(manager, "CurrentEGVGuiDataRequest", map[string]interface{}{
		"bgReadingTimestampSeconds": 0,
		"cgmReading":                120,
		"egvStatusId":               0,
		"trendRate":                 0,
	})

	// HomeScreenMirrorResponse(int cgmTrendIconId, int cgmAlertIconId, int statusIcon0Id,
	// int statusIcon1Id, int bolusStatusIconId, int basalStatusIconId,
	// int apControlStateIconId, boolean remainingInsulinPlusIcon, boolean cgmDisplayData)
	registerConstant(manager, "HomeScreenMirrorRequest", map[string]interface{}{
		"cgmTrendIconId":           0,
		"cgmAlertIconId":           0,
		"statusIcon0Id":            0,
		"statusIcon1Id":            0,
		"bolusStatusIconId":        0,
		"basalStatusIconId":        0,
		"apControlStateIconId":     0,
		"remainingInsulinPlusIcon": false,
		"cgmDisplayData":           true,
	})

	// CGMStatusResponse(int sessionStateId, long lastCalibrationTimestamp,
	// long sensorStartedTimestamp, int transmitterBatteryStatusId)
	registerConstant(manager, "CGMStatusRequest", map[string]interface{}{
		"sessionStateId":             1,
		"lastCalibrationTimestamp":   0,
		"sensorStartedTimestamp":     0,
		"transmitterBatteryStatusId": 1,
	})

	// AlertStatusResponse's real "data" constructor takes a BigInteger, but
	// cliparser's constructor lookup (by parameter count) resolves any 1-param
	// call to the class's OTHER 1-arg constructor first: the raw byte[] one.
	// So a plain int/BigInteger value can never reach the real constructor
	// through this tool -- pass raw cargo bytes instead (size=8).
	registerConstant(manager, "AlertStatusRequest", map[string]interface{}{
		"raw": "0000000000000000",
	})

	// AlarmStatusResponse's real "data" constructor takes an AlarmResponseType...
	// varargs enum array, which cliparser cannot construct from JSON at all (no
	// enum support) -- no 1-param JSON input can ever succeed. Fall back to the
	// no-arg constructor (empty/zeroed cargo).
	registerConstant(manager, "AlarmStatusRequest", map[string]interface{}{})

	// LoadStatusResponse has two real "data" constructors with the same 3-param
	// shape (one takes a LoadState enum cliparser can't build), plus one
	// deprecated 2-int constructor -- the deprecated one is the only one
	// reachable through this tool.
	registerConstant(manager, "LoadStatusRequest", map[string]interface{}{
		"status":  0,
		"unknown": 0,
	})

	// ProfileStatusResponse(int numberOfProfiles, int idpSlot0Id, int idpSlot1Id,
	// int idpSlot2Id, int idpSlot3Id, int idpSlot4Id, int idpSlot5Id, int activeSegmentIndex)
	registerConstant(manager, "ProfileStatusRequest", map[string]interface{}{
		"numberOfProfiles":   1,
		"idpSlot0Id":         0,
		"idpSlot1Id":         -1,
		"idpSlot2Id":         -1,
		"idpSlot3Id":         -1,
		"idpSlot4Id":         -1,
		"idpSlot5Id":         -1,
		"activeSegmentIndex": 0,
	})

	// LastBolusStatusV2Response(int status, int bolusId, long timestamp,
	// long deliveredVolume, int bolusStatusId, int bolusSourceId,
	// int bolusTypeBitmask, long extendedBolusDuration, long requestedVolume)
	registerConstant(manager, "LastBolusStatusV2Request", map[string]interface{}{
		"status":                0,
		"bolusId":               0,
		"timestamp":             0,
		"deliveredVolume":       0,
		"bolusStatusId":         0,
		"bolusSourceId":         0,
		"bolusTypeBitmask":      0,
		"extendedBolusDuration": 0,
		"requestedVolume":       0,
	})
}

// registerPumpInfoDefaults registers defaults for pump info messages
func registerPumpInfoDefaults(manager *Manager) {
	// PumpFeaturesV2Response(int status, int supportedFeatureIndex, long pumpFeaturesBitmask)
	registerConstant(manager, "PumpFeaturesV2Request", map[string]interface{}{
		"status":                0,
		"supportedFeatureIndex": 0,
		"pumpFeaturesBitmask":   0,
	})

	// PumpVersionResponse's real constructor takes 10 fields (long armSwVer,
	// long mspSwVer, long configABits, long configBBits, long serialNum, long
	// partNum, String pumpRev, long pcbaSN, String pcbaRev, long modelNum) --
	// values below are a real captured Tandem Mobi PumpVersionResponse from
	// pumpX2's own test fixtures (PumpVersionResponseTest.testPumpVersionResponse_Mobi).
	registerConstant(manager, "PumpVersionRequest", map[string]interface{}{
		"armSwVer":    3628697757,
		"mspSwVer":    0,
		"configABits": 0,
		"configBBits": 0,
		"serialNum":   1226976,
		"partNum":     1013045,
		"pumpRev":     "0",
		"pcbaSN":      232700077,
		"pcbaRev":     "0",
		"modelNum":    1004000,
	})

	// BleSoftwareInfoResponse(int softDeviceId, int softDeviceMajorVersion,
	// int softDeviceMinorVersion, int softDeviceBugfixVersion, long softDeviceVersion,
	// int softDeviceSubVersion)
	registerConstant(manager, "BleSoftwareInfoRequest", map[string]interface{}{
		"softDeviceId":            0,
		"softDeviceMajorVersion":  1,
		"softDeviceMinorVersion":  0,
		"softDeviceBugfixVersion": 0,
		"softDeviceVersion":       0,
		"softDeviceSubVersion":    0,
	})

	// CommonSoftwareInfoResponse(String appSoftwareVersion, long appSoftwarePartNumber,
	// long appSoftwarePartDashNumber, long appSoftwarePartRevisionNumber,
	// String bootloaderVersion, long bootloaderPartNumber, long bootloaderPartDashNumber,
	// long bootloaderPartRevisionNumber)
	registerConstant(manager, "CommonSoftwareInfoRequest", map[string]interface{}{
		"appSoftwareVersion":            "1.0.0.0",
		"appSoftwarePartNumber":         0,
		"appSoftwarePartDashNumber":     0,
		"appSoftwarePartRevisionNumber": 0,
		"bootloaderVersion":             "1.0.0.0",
		"bootloaderPartNumber":          0,
		"bootloaderPartDashNumber":      0,
		"bootloaderPartRevisionNumber":  0,
	})
}

// registerNotificationDefaults registers defaults for notification bundle / alarm / malfunction handlers
func registerNotificationDefaults(manager *Manager) {
	// HighestAamRequest (opcode 120) — from ALARM qualifying event
	// HighestAamResponse(long aamId, long faultId, byte[] remaining)
	registerConstant(manager, "HighestAamRequest", map[string]interface{}{
		"aamId":     0,
		"faultId":   0,
		"remaining": "000000",
	})

	// ActiveAamBitsRequest — from ALARM qualifying event. The real "data"
	// constructor takes 2 BigIntegers plus an AamType enum, which cliparser
	// can't construct from JSON -- pass raw cargo bytes instead (17 bytes: two
	// zeroed 8-byte bitmasks + aamType=NONE(5)).
	registerConstant(manager, "ActiveAamBitsRequest", map[string]interface{}{
		"raw": "0000000000000000000000000000000005",
	})

	// MalfunctionStatusRequest (opcode 118) — from MALFUNCTION qualifying event.
	// Its real response class is MalfunctionBitmaskStatusResponse, not
	// "MalfunctionStatusResponse" (see genericSettingsResponseTypeOverrides in
	// pkg/handler/generic_settings.go). That class has a BigInteger ctor and an
	// unambiguous 2-arg (long codeA, long codeB) ctor; use the latter since
	// cliparser can't build a BigInteger param.
	registerConstant(manager, "MalfunctionStatusRequest", map[string]interface{}{
		"codeA": 0,
		"codeB": 0,
	})

	// ReminderStatusResponse's only real constructor besides no-arg takes a
	// BigInteger, which cliparser cannot construct from a plain JSON number
	// (org.json only produces BigInteger for values exceeding Long.MAX_VALUE).
	// Fall back to the no-arg constructor (empty/zeroed cargo).
	registerConstant(manager, "ReminderStatusRequest", map[string]interface{}{})

	// CGMAlertStatusResponse(long cgmAlertBitmask)
	registerConstant(manager, "CGMAlertStatusRequest", map[string]interface{}{
		"cgmAlertBitmask": 0,
	})
}

// registerControlIQDefaults registers defaults for ControlIQ info and sleep schedule
func registerControlIQDefaults(manager *Manager) {
	// ControlIQInfoV1Response(boolean closedLoopEnabled, int weight, int weightUnit,
	// int totalDailyInsulin, int currentUserModeType, int byte6, int byte7,
	// int byte8, int controlStateType)
	registerConstant(manager, "ControlIQInfoV1Request", map[string]interface{}{
		"closedLoopEnabled":   true,
		"weight":              70,
		"weightUnit":          0,
		"totalDailyInsulin":   40,
		"currentUserModeType": 0,
		"byte6":               0,
		"byte7":               0,
		"byte8":               0,
		"controlStateType":    0,
	})

	// ControlIQInfoV2Response — same 9 fields as V1 plus exercise fields
	registerConstant(manager, "ControlIQInfoV2Request", map[string]interface{}{
		"closedLoopEnabled":     true,
		"weight":                70,
		"weightUnit":            0,
		"totalDailyInsulin":     40,
		"currentUserModeType":   0,
		"byte6":                 0,
		"byte7":                 0,
		"byte8":                 0,
		"controlStateType":      0,
		"exerciseChoice":        0,
		"exerciseDuration":      0,
		"exerciseTimeRemaining": 0,
	})

	// ControlIQSleepScheduleResponse's real constructor takes 4 nested
	// SleepSchedule objects, which cliparser cannot construct from JSON at all,
	// and unlike most Message subclasses this one has no raw byte[] constructor
	// either. The no-arg constructor is the only reachable path; it produces an
	// empty (not real 24-byte) cargo, so this response is known-incomplete.
	registerConstant(manager, "ControlIQSleepScheduleRequest", map[string]interface{}{})

	// BasalIQStatusResponse(int basalIQStatusState, boolean deliveringTherapy)
	registerConstant(manager, "BasalIQStatusRequest", map[string]interface{}{
		"basalIQStatusState": 0,
		"deliveringTherapy":  true,
	})

	// NonControlIQIOBResponse(long iob, long timeRemaining, long totalIOB)
	registerConstant(manager, "NonControlIQIOBRequest", map[string]interface{}{
		"iob":           250,
		"timeRemaining": 0,
		"totalIOB":      250,
	})
}

// registerBolusAndBasalDefaults registers defaults for extended bolus and temp rate handlers
func registerBolusAndBasalDefaults(manager *Manager) {
	// ExtendedBolusStatusResponse(int bolusStatus, int bolusId, long timestamp,
	// long requestedVolume, long duration, int bolusSource)
	registerConstant(manager, "ExtendedBolusStatusRequest", map[string]interface{}{
		"bolusStatus":     0, // no extended bolus active
		"bolusId":         0,
		"timestamp":       0,
		"requestedVolume": 0,
		"duration":        0,
		"bolusSource":     0,
	})

	// ExtendedBolusStatusV2Response — same 6 fields as V1 plus secondsSincePumpReset
	registerConstant(manager, "ExtendedBolusStatusV2Request", map[string]interface{}{
		"bolusStatus":           0,
		"bolusId":               0,
		"timestamp":             0,
		"requestedVolume":       0,
		"duration":              0,
		"bolusSource":           0,
		"secondsSincePumpReset": 0,
	})

	// LastBolusStatusV3Response — 22-field constructor covering both a standard
	// and an extended bolus record, including 2 unused/reserved byte[] fields.
	registerConstant(manager, "LastBolusStatusV3Request", map[string]interface{}{
		"lastBolusTypeBitmask":               0,
		"standardBolusStatusId":              0,
		"standardBolusId":                    0,
		"standardUnknown":                    "0000",
		"standardBolusTimestamp":             0,
		"standardBolusDeliveredVolume":       0,
		"standardBolusEndReasonId":           0,
		"standardBolusSourceId":              0,
		"standardBolusTypeBitmask":           0,
		"standardBolusRequestedVolume":       0,
		"standardBolusSecondsSincePumpReset": 0,
		"extendedBolusStatusId":              0,
		"extendedBolusId":                    0,
		"extendedUnknown":                    "0000",
		"extendedBolusTimestamp":             0,
		"extendedBolusDeliveredVolume":       0,
		"extendedBolusEndReasonId":           0,
		"extendedBolusSourceId":              0,
		"extendedBolusTypeBitmask":           0,
		"extendedBolusRequestedVolume":       0,
		"extendedBolusSecondsSincePumpReset": 0,
		"extendedBolusDuration":              0,
	})

	// TempRateResponse(boolean active, int percentage, long startTimeRaw, long duration)
	registerConstant(manager, "TempRateRequest", map[string]interface{}{
		"active":       false,
		"percentage":   0,
		"startTimeRaw": 0,
		"duration":     0,
	})

	// TempRateStatusResponse has no named-field constructor at all (only
	// no-arg and a raw byte[] constructor exist) -- pass raw cargo bytes
	// (size=16).
	registerConstant(manager, "TempRateStatusRequest", map[string]interface{}{
		"raw": "00000000000000000000000000000000",
	})

	// LastBGResponse: KNOWN BROKEN. Its real constructors are (long bgTimestamp,
	// int bgValue, int bgSourceId) and an ambiguous (long, int, BgSource enum)
	// overload; cliparser's reflection-based constructor lookup deterministically
	// picks the enum overload for any 3-param call and has no enum conversion
	// support, so this always throws regardless of field names/values -- an
	// upstream cliparser limitation, not fixable from here. Kept
	// semantically-correct (matching the real field names) for clarity even
	// though it's known to fail.
	registerConstant(manager, "LastBGRequest", map[string]interface{}{
		"bgTimestamp": 0,
		"bgValue":     0,
		"bgSourceId":  0,
	})

	// BolusPermissionChangeReasonResponse(int bolusId, boolean isAcked,
	// int lastChangeReasonId, boolean currentPermissionHolder)
	registerConstant(manager, "BolusPermissionChangeReasonRequest", map[string]interface{}{
		"bolusId":                 0,
		"isAcked":                 true,
		"lastChangeReasonId":      0,
		"currentPermissionHolder": true,
	})
}

// registerGlobalSettingsDefaults registers defaults for global pump settings messages
func registerGlobalSettingsDefaults(manager *Manager) {
	// GlobalMaxBolusSettingsResponse(int maxBolus, int maxBolusDefault)
	registerConstant(manager, "GlobalMaxBolusSettingsRequest", map[string]interface{}{
		"maxBolus":        2500, // 25.0 units * 100
		"maxBolusDefault": 2500,
	})

	// BasalLimitSettingsResponse(long basalLimit, long basalLimitDefault)
	registerConstant(manager, "BasalLimitSettingsRequest", map[string]interface{}{
		"basalLimit":        500, // 5.0 U/hr * 100
		"basalLimitDefault": 500,
	})

	// LocalizationResponse(int glucoseUOM, int languageSelected, int regionSetting,
	// long languagesAvailableBitmask)
	registerConstant(manager, "LocalizationRequest", map[string]interface{}{
		"glucoseUOM":                0, // mg/dL
		"languageSelected":          0, // English
		"regionSetting":             0,
		"languagesAvailableBitmask": 1,
	})

	// PumpSettingsResponse(int lowInsulinThreshold, int cannulaPrimeSize,
	// int autoShutdownEnabled, int autoShutdownDuration, int featureLock,
	// int oledTimeout, int status)
	registerConstant(manager, "PumpSettingsRequest", map[string]interface{}{
		"lowInsulinThreshold":  20,
		"cannulaPrimeSize":     0,
		"autoShutdownEnabled":  0,
		"autoShutdownDuration": 0,
		"featureLock":          0,
		"oledTimeout":          0,
		"status":               0,
	})

	// SendTipsControlGenericTestResponse(int status)
	registerConstant(manager, "SendTipsControlGenericTestRequest", map[string]interface{}{
		"status": 0,
	})
}

// registerAdditionalStatusDefaults registers defaults for additional status handlers
func registerAdditionalStatusDefaults(manager *Manager) {
	// IDPSegmentResponse(int idpId, int segmentIndex, int profileStartTime,
	// int profileBasalRate, long profileCarbRatio, int profileTargetBG,
	// int profileISF, int statusId)
	registerConstant(manager, "IDPSegmentRequest", map[string]interface{}{
		"idpId":            1,
		"segmentIndex":     0,
		"profileStartTime": 0,
		"profileBasalRate": 850,
		"profileCarbRatio": 10000,
		"profileTargetBG":  110,
		"profileISF":       50,
		"statusId":         0,
	})

	// IDPSettingsResponse(int idpId, String name, int numberOfProfileSegments,
	// int insulinDuration, int maxBolus, boolean carbEntry)
	registerConstant(manager, "IDPSettingsRequest", map[string]interface{}{
		"idpId":                   1,
		"name":                    "Profile 1",
		"numberOfProfileSegments": 1,
		"insulinDuration":         300,
		"maxBolus":                25000,
		"carbEntry":               true,
	})

	// GetSavedG7PairingCodeResponse(int pairingCode)
	registerConstant(manager, "GetSavedG7PairingCodeRequest", map[string]interface{}{
		"pairingCode": 0,
	})

	// CurrentActiveIdpValuesResponse(long currentCarbRatio, int currentTargetBg,
	// int currentInsulinDuration, int currentIsf)
	registerConstant(manager, "CurrentActiveIdpValuesRequest", map[string]interface{}{
		"currentCarbRatio":       10000,
		"currentTargetBg":        110,
		"currentInsulinDuration": 300,
		"currentIsf":             50,
	})

	// PumpFeaturesV1Response's real constructor takes a BigInteger, but it has
	// an ambiguous 1-arg raw byte[] constructor that cliparser's constructor
	// lookup resolves to first -- pass raw cargo bytes instead (size=8).
	registerConstant(manager, "PumpFeaturesV1Request", map[string]interface{}{
		"raw": "0504000000000000",
	})

	// PumpVersionBResponse(String softwareName, long configurationBitsA,
	// long configurationBitsB, long serialNumber, long modelNumber,
	// String pumpRevision, long pcbPartNumberA, long pcbSerialNumberA,
	// String pcbRevisionNumberA)
	registerConstant(manager, "PumpVersionBRequest", map[string]interface{}{
		"softwareName":       "1.1.1",
		"configurationBitsA": 0,
		"configurationBitsB": 0,
		"serialNumber":       123456,
		"modelNumber":        1000,
		"pumpRevision":       "A",
		"pcbPartNumberA":     0,
		"pcbSerialNumberA":   0,
		"pcbRevisionNumberA": "A",
	})

	// CgmStatusV2Response(int sessionStateId, long lastCalibrationTimestamp,
	// long sensorStartedTimestamp, int transmitterBatteryStatusId,
	// long sessionDurationSeconds, long sessionTimeRemainingSeconds,
	// int cgmSensorTypeId, boolean gracePeriod)
	registerConstant(manager, "CgmStatusV2Request", map[string]interface{}{
		"sessionStateId":              2,
		"lastCalibrationTimestamp":    0,
		"sensorStartedTimestamp":      0,
		"transmitterBatteryStatusId":  3,
		"sessionDurationSeconds":      0,
		"sessionTimeRemainingSeconds": 0,
		"cgmSensorTypeId":             1,
		"gracePeriod":                 false,
	})

	// CurrentEgvGuiDataV2Response(long bgReadingTimestampSeconds, int cgmReading,
	// int egvStatusId, int trendRate)
	registerConstant(manager, "CurrentEgvGuiDataV2Request", map[string]interface{}{
		"bgReadingTimestampSeconds": 0,
		"cgmReading":                120,
		"egvStatusId":               0,
		"trendRate":                 0,
	})

	// LastBolusStatusResponse(int status, int bolusId, long timestamp,
	// long deliveredVolume, int bolusStatusId, int bolusSourceId,
	// int bolusTypeBitmask, long extendedBolusDuration, byte[] unknown)
	registerConstant(manager, "LastBolusStatusRequest", map[string]interface{}{
		"status":                0,
		"bolusId":               0,
		"timestamp":             0,
		"deliveredVolume":       0,
		"bolusStatusId":         0,
		"bolusSourceId":         0,
		"bolusTypeBitmask":      0,
		"extendedBolusDuration": 0,
		"unknown":               "0000",
	})

	// CGMHardwareInfoResponse(String hardwareInfoString, int lastByte)
	registerConstant(manager, "CGMHardwareInfoRequest", map[string]interface{}{
		"hardwareInfoString": "80AB12",
		"lastByte":           0,
	})

	// CGMGlucoseAlertSettingsResponse(int highGlucoseAlertThreshold,
	// int highGlucoseAlertEnabled, int highGlucoseRepeatDuration,
	// int highGlucoseAlertDefaultBitmask, int lowGlucoseAlertThreshold,
	// int lowGlucoseAlertEnabled, int lowGlucoseRepeatDuration,
	// int lowGlucoseAlertDefaultBitmask)
	registerConstant(manager, "CGMGlucoseAlertSettingsRequest", map[string]interface{}{
		"highGlucoseAlertThreshold":      250,
		"highGlucoseAlertEnabled":        0,
		"highGlucoseRepeatDuration":      0,
		"highGlucoseAlertDefaultBitmask": 0,
		"lowGlucoseAlertThreshold":       70,
		"lowGlucoseAlertEnabled":         0,
		"lowGlucoseRepeatDuration":       0,
		"lowGlucoseAlertDefaultBitmask":  0,
	})

	// CGMOORAlertSettingsResponse(int sensorTimeoutAlertThreshold,
	// int sensorTimeoutAlertEnabled, int sensorTimeoutDefaultBitmask)
	registerConstant(manager, "CGMOORAlertSettingsRequest", map[string]interface{}{
		"sensorTimeoutAlertThreshold": 0,
		"sensorTimeoutAlertEnabled":   0,
		"sensorTimeoutDefaultBitmask": 0,
	})

	// CGMRateAlertSettingsResponse(int riseRateThreshold, int riseRateEnabled,
	// int riseRateDefaultBitmask, int fallRateThreshold, int fallRateEnabled,
	// int fallRateDefaultBitmask)
	registerConstant(manager, "CGMRateAlertSettingsRequest", map[string]interface{}{
		"riseRateThreshold":      0,
		"riseRateEnabled":        0,
		"riseRateDefaultBitmask": 0,
		"fallRateThreshold":      0,
		"fallRateEnabled":        0,
		"fallRateDefaultBitmask": 0,
	})

	// BasalIQAlertInfoResponse(long alertId)
	registerConstant(manager, "BasalIQAlertInfoRequest", map[string]interface{}{
		"alertId": 0,
	})

	// RemindersResponse's real constructor takes 13 fields, 9 of which are a
	// nested Reminder custom type cliparser cannot construct from JSON, and
	// there's no raw byte[] fallback constructor either. The no-arg
	// constructor is the only reachable path; it produces an empty (not real)
	// cargo, so this response is known-incomplete.
	registerConstant(manager, "RemindersRequest", map[string]interface{}{})

	// QuickBolusSettingsRequest was removed: no "QuickBolusSettingsResponse"
	// class exists anywhere in pumpX2 (confirmed via the jar's own class
	// table). The only quick-bolus message pumpX2 has is the write pair
	// SetQuickBolusSettingsRequest/Response (already handled elsewhere via
	// SettingsWriteHandler), not a "get".

	// CgmSupportPackageStatusResponse(int status, boolean validity)
	registerConstant(manager, "CgmSupportPackageStatusRequest", map[string]interface{}{
		"status":   0,
		"validity": true,
	})
}
