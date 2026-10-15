package state

// History log event type IDs.
//
// These are the `typeId` values that go in the first 12 bits of a history log
// record's 26-byte cargo. Every value below was taken from the corresponding
// class in TandemKit's Sources/TandemCore/Messages/HistoryLog (each declares
// `public static let typeId`), which in turn mirrors pumpX2's
// @HistoryLogProps(typeId=...).
//
// An earlier version of this file carried invented, sequentially-assigned IDs
// that happened to be right for only a handful of events (BolusCompleted 20,
// BolusDelivery 280, DateChange 14). Anything streamed with a wrong ID decodes
// on the driver side as a different event class, or as an unknown record that
// is silently dropped -- so these must not be "tidied" into a neat sequence
// again.
const (
	// Bolus events
	HistoryBolusRequestedMsg1 = 64
	HistoryBolusRequestedMsg2 = 65
	HistoryBolusRequestedMsg3 = 66
	HistoryBolusActivated     = 55
	HistoryBolusCompleted     = 20
	HistoryBolusDelivery      = 280
	HistoryBolexActivated     = 59
	HistoryBolexCompleted     = 21
	HistoryCorrectionDeclined = 93

	// Basal events
	HistoryBasalDelivery   = 279
	HistoryBasalRateChange = 3
	HistoryDailyBasal      = 81

	// Temp rate events
	HistoryTempRateActivated = 2
	HistoryTempRateCompleted = 15

	// Cartridge/tubing events
	HistoryCartridgeInserted = 32
	HistoryCartridgeRemoved  = 31
	HistoryCartridgeFilled   = 33
	HistoryTubingFilled      = 63
	HistoryCannulaFilled     = 61

	// Pump state events
	HistoryPumpingSuspended = 11
	HistoryPumpingResumed   = 12

	// Remote entry events
	HistoryBGEntry   = 16
	HistoryCarbEntry = 48

	// System events
	HistoryDateChange   = 14
	HistoryTimeChanged  = 13
	HistoryNewDay       = 90
	HistoryVersionInfo  = 191
	HistoryFactoryReset = 82
	HistoryLogErased    = 0

	// Alarm/Alert events
	HistoryAlarmActivated = 5
	HistoryAlarmAck       = 8
	HistoryAlarmCleared   = 28
	HistoryAlertActivated = 4
	HistoryAlertAck       = 27
	HistoryAlertCleared   = 26
	HistoryMalfunction    = 6

	// CGM events
	HistoryCGMData             = 256
	HistoryCGMCalibration      = 160
	HistoryCGMSensorTypeChange = 368
	HistoryCGMStartSession     = 212
	HistoryCGMStopSession      = 214
	HistoryCGMAlertActivated   = 171
	HistoryCGMAlertCleared     = 172

	// Settings events
	HistoryParamChangeGlobalSettings = 74
	HistoryParamChangePumpSettings   = 73
	HistoryControlIQUserModeChange   = 229
	HistoryBasalIQSettingsChange     = 142

	// Control-IQ state changes and Bluetooth temp rate sets
	HistoryControlIQPcmChange  = 230
	HistorySetTempRateResponse = 309

	// IDP events
	HistoryIDPAction  = 69
	HistoryIDPBolus   = 70
	HistoryIDPList    = 71
	HistoryIDPSegment = 68

	// Hypo protection
	HistoryHypoMinimizerSuspend = 198
	HistoryHypoMinimizerResume  = 199

	// Additional PumpX2 LOG_MESSAGE_TYPES. Their payload fields can be supplied
	// by name when implemented below, or as Fields["raw"] bytes for exact replay.
	HistoryBolusIobEntry                 = 195
	HistoryIobSnapshotPreBolus           = 206
	HistoryIobSnapshotPostBolus          = 207
	HistoryInitiateBolusRequestMsg1      = 292
	HistoryInitiateBolusRequestMsg2      = 293
	HistoryBolusPermissionChange         = 295
	HistoryBolusPermissionResponse       = 297
	HistoryInitiateBolusResponse         = 298
	HistoryBolusPermissionRequest        = 338
	HistoryInitiateBolusRequestMsg3      = 343
	HistoryReservoirEstimate             = 231
	HistoryCartridgeFillResponse         = 359
	HistoryPumpButton                    = 17
	HistoryDataLogCorruption             = 60
	HistoryIdpActionMsg2                 = 57
	HistoryParamChangeRemSettings        = 97
	HistoryParamChangeReminder           = 96
	HistoryControlRequest                = 290
	HistoryControlResponse               = 300
	HistoryUsbConnected                  = 36
	HistoryUsbDisconnected               = 37
	HistoryUsbEnumerated                 = 67
	HistoryChargingStarted               = 34
	HistoryChargingStopped               = 35
	HistoryBatteryHourly                 = 271
	HistoryBatteryEstimateUpdate         = 330
	HistoryCgmDataSample                 = 151
	HistoryCgmDataGx                     = 211
	HistoryCgmCalibrationGx              = 210
	HistoryCgmAlertAckDex                = 371
	HistoryCgmAlertActivatedDex          = 369
	HistoryCgmAlertActivatedFsl2         = 460
	HistoryCgmAlertClearedDex            = 370
	HistoryCgmAlertClearedFsl2           = 461
	HistoryCgmDataFsl2                   = 372
	HistoryCgmDataFsl3                   = 480
	HistoryBolusDeliverySummary          = 194
	HistoryPumpDriveEvent                = 277
	HistoryCgmJoinSessionFsl2            = 406
	HistoryCgmJoinSessionFsl3            = 477
	HistoryCgmJoinSessionG7              = 394
	HistoryCgmJoinSession                = 213
	HistoryCgmStartSessionFsl2           = 404
	HistoryCgmStopSessionFsl2            = 405
	HistoryCgmStopSessionFsl3            = 486
	HistoryCgmStopSessionG7              = 447
	HistorySoftwareID                    = 449
	HistoryControlIQCycle                = 227
	HistoryDexcomG7CGM                   = 399
	HistoryPlgsPeriodic                  = 140
	HistoryShelfMode                     = 53
	HistoryArmInit                       = 99
	HistoryMaxBasalLimitSet              = 274
	HistoryVersionsA                     = 307
	HistoryMaxBolusLimitSet              = 326
	HistoryControlIQGlucoseModel         = 327
	HistoryMalfunctionAck                = 7
	HistoryReminderSnoozed               = 30
	HistoryConfirmCartridgeFilled        = 41
	HistoryFillEstimateFinal             = 98
	HistoryCgmTransmitterID              = 156
	HistoryCgmAnnuSettings               = 157
	HistoryCgmStopSessionMsg1            = 162
	HistoryCgmStopSessionMsg2            = 163
	HistoryCgmHgaSettings                = 165
	HistoryCgmLgaSettings                = 166
	HistoryCgmRraSettings                = 167
	HistoryCgmFraSettings                = 168
	HistoryCgmOorSettings                = 169
	HistoryCgmAlertAck                   = 173
	HistoryCgmUnexpectedGeAlert          = 187
	HistoryCgmInactiveGx                 = 215
	HistoryCgmTransmitterIDGx            = 216
	HistoryCgmStartSessionReqGx          = 217
	HistoryCgmStopSessionReqGx           = 218
	HistoryCgmStateChangeGx              = 219
	HistoryCgmTransmitterVersionGx       = 220
	HistoryAaSleepScheduleChange         = 235
	HistoryControlIQAutomatedInsulin     = 237
	HistoryAaDeliveryStatusChange        = 238
	HistoryAaEnableSettingChange         = 244
	HistoryAaTdiSettingChange            = 245
	HistoryAaWeightSettingChange         = 246
	HistoryCgmSessionTypeChange          = 267
	HistoryWumpOcclusionDebug            = 283
	HistorySnoozeActivated               = 286
	HistoryAaAutoBolusRejected           = 288
	HistoryTipscReqPrimeCannula          = 291
	HistoryWumpCartridgeFilled           = 301
	HistoryWumpCartridgeRemoved          = 302
	HistoryAAExerciseTimeChange          = 318
	HistoryAAExerciseChoiceChange        = 319
	HistoryAATdiEstChange                = 332
	HistoryControlIQFeatures             = 328
	HistoryPrimeInprocess                = 348
	HistoryCgmRejoinSession              = 367
	HistoryCgmStartSensorReqG7           = 390
	HistoryCgmCalibrationG7              = 438
	HistoryCgmBleCalibrationEvtG7        = 439
	HistoryCgmInactiveG7                 = 441
	HistoryCgmStopSessionReqG7           = 443
	HistoryBleSoftDeviceVersion          = 450
	HistoryDismissNotificationResponse   = 334
	HistoryAamAnnunciation               = 434
	HistoryReminderActivated             = 25
	HistoryReminderDismissed             = 29
	HistoryHistoryRequestCounters        = 490
	HistoryHistoryRequestRange           = 491
	HistoryControlRequestCounters        = 492
	HistoryControlStreamCounters         = 493
	HistoryAuthorizationCounters         = 494
	HistoryCgmPairingCodeG7              = 395
	HistoryBleLinkStats                  = 473
	HistoryBleDailyConnections           = 474
	HistoryTipsError                     = 419
	HistorySyncDoseStart                 = 558
	HistorySyncDoseComplete              = 559
	HistoryAntDailyStatus1               = 562
	HistoryAntDailyStatus2               = 563
	HistoryTempAdjustmentCompleted       = 564
	HistoryTempAdjustmentActivated       = 565
	HistoryAntSleepEatSchedSettingChange = 566

	// Daily/status. There is no LoadStatus history log in pumpX2 or TandemKit
	// (LoadStatus is a CURRENT_STATUS message, not a log record), so the
	// constant that used to sit here has been removed rather than given a
	// guessed ID.
	HistoryDailyStatus  = 313
	HistoryUpdateStatus = 203
)

// historyTypeIDByName maps every class in PumpX2 HistoryLogParser.LOG_MESSAGE_TYPES
// to its wire type ID, so API callers can append any tracked record by name.
// Payload fields without a dedicated encoder can be supplied as Fields["raw"].
var historyTypeIDByName = map[string]int{
	"DateChange":                    HistoryDateChange,
	"BG":                            HistoryBGEntry,
	"DexcomG6CGM":                   HistoryCGMData,
	"BolusDelivery":                 HistoryBolusDelivery,
	"BolusCompleted":                HistoryBolusCompleted,
	"BolusIobEntry":                 HistoryBolusIobEntry,
	"IobSnapshotPreBolus":           HistoryIobSnapshotPreBolus,
	"IobSnapshotPostBolus":          HistoryIobSnapshotPostBolus,
	"BolusRequestedMsg1":            HistoryBolusRequestedMsg1,
	"BolusRequestedMsg2":            HistoryBolusRequestedMsg2,
	"BolusRequestedMsg3":            HistoryBolusRequestedMsg3,
	"InitiateBolusRequestMsg1":      HistoryInitiateBolusRequestMsg1,
	"InitiateBolusRequestMsg2":      HistoryInitiateBolusRequestMsg2,
	"BolusPermissionChange":         HistoryBolusPermissionChange,
	"BolusPermissionResponse":       HistoryBolusPermissionResponse,
	"InitiateBolusResponse":         HistoryInitiateBolusResponse,
	"BolusPermissionRequest":        HistoryBolusPermissionRequest,
	"InitiateBolusRequestMsg3":      HistoryInitiateBolusRequestMsg3,
	"BolexCompleted":                HistoryBolexCompleted,
	"BasalRateChange":               HistoryBasalRateChange,
	"BolexActivated":                HistoryBolexActivated,
	"BolusActivated":                HistoryBolusActivated,
	"CannulaFilled":                 HistoryCannulaFilled,
	"CarbEntered":                   HistoryCarbEntry,
	"CartridgeFilled":               HistoryCartridgeFilled,
	"ReservoirEstimate":             HistoryReservoirEstimate,
	"CartridgeFillResponse":         HistoryCartridgeFillResponse,
	"PumpButton":                    HistoryPumpButton,
	"CorrectionDeclined":            HistoryCorrectionDeclined,
	"DailyBasal":                    HistoryDailyBasal,
	"DataLogCorruption":             HistoryDataLogCorruption,
	"IdpAction":                     HistoryIDPAction,
	"IdpBolus":                      HistoryIDPBolus,
	"IdpList":                       HistoryIDPList,
	"FactoryReset":                  HistoryFactoryReset,
	"IdpActionMsg2":                 HistoryIdpActionMsg2,
	"IdpTimeDependentSegment":       HistoryIDPSegment,
	"LogErased":                     HistoryLogErased,
	"NewDay":                        HistoryNewDay,
	"ParamChangeGlobalSettings":     HistoryParamChangeGlobalSettings,
	"ParamChangePumpSettings":       HistoryParamChangePumpSettings,
	"ParamChangeRemSettings":        HistoryParamChangeRemSettings,
	"ParamChangeReminder":           HistoryParamChangeReminder,
	"PumpingResumed":                HistoryPumpingResumed,
	"PumpingSuspended":              HistoryPumpingSuspended,
	"TempRateActivated":             HistoryTempRateActivated,
	"TempRateCompleted":             HistoryTempRateCompleted,
	"ControlRequest":                HistoryControlRequest,
	"ControlResponse":               HistoryControlResponse,
	"SetTempRateResponse":           HistorySetTempRateResponse,
	"TimeChanged":                   HistoryTimeChanged,
	"TubingFilled":                  HistoryTubingFilled,
	"UsbConnected":                  HistoryUsbConnected,
	"UsbDisconnected":               HistoryUsbDisconnected,
	"UsbEnumerated":                 HistoryUsbEnumerated,
	"ChargingStarted":               HistoryChargingStarted,
	"ChargingStopped":               HistoryChargingStopped,
	"BatteryHourly":                 HistoryBatteryHourly,
	"BatteryEstimateUpdate":         HistoryBatteryEstimateUpdate,
	"AlarmActivated":                HistoryAlarmActivated,
	"AlertActivated":                HistoryAlertActivated,
	"AlarmCleared":                  HistoryAlarmCleared,
	"CgmDataSample":                 HistoryCgmDataSample,
	"CgmCalibration":                HistoryCGMCalibration,
	"CgmDataGx":                     HistoryCgmDataGx,
	"CgmCalibrationGx":              HistoryCgmCalibrationGx,
	"CgmAlertAckDex":                HistoryCgmAlertAckDex,
	"CgmAlertActivatedDex":          HistoryCgmAlertActivatedDex,
	"CgmAlertActivatedFsl2":         HistoryCgmAlertActivatedFsl2,
	"CgmAlertActivated":             HistoryCGMAlertActivated,
	"CgmAlertClearedDex":            HistoryCgmAlertClearedDex,
	"CgmAlertClearedFsl2":           HistoryCgmAlertClearedFsl2,
	"CgmAlertCleared":               HistoryCGMAlertCleared,
	"CgmDataFsl2":                   HistoryCgmDataFsl2,
	"CgmDataFsl3":                   HistoryCgmDataFsl3,
	"HypoMinimizerSuspend":          HistoryHypoMinimizerSuspend,
	"HypoMinimizerResume":           HistoryHypoMinimizerResume,
	"BolusDeliverySummary":          HistoryBolusDeliverySummary,
	"BasalDelivery":                 HistoryBasalDelivery,
	"PumpDriveEvent":                HistoryPumpDriveEvent,
	"CgmJoinSessionFsl2":            HistoryCgmJoinSessionFsl2,
	"CgmJoinSessionFsl3":            HistoryCgmJoinSessionFsl3,
	"CgmJoinSessionG7":              HistoryCgmJoinSessionG7,
	"CgmJoinSession":                HistoryCgmJoinSession,
	"CgmStartSession":               HistoryCGMStartSession,
	"CgmStartSessionFsl2":           HistoryCgmStartSessionFsl2,
	"CgmStopSession":                HistoryCGMStopSession,
	"CgmStopSessionFsl2":            HistoryCgmStopSessionFsl2,
	"CgmStopSessionFsl3":            HistoryCgmStopSessionFsl3,
	"CgmStopSessionG7":              HistoryCgmStopSessionG7,
	"SoftwareId":                    HistorySoftwareID,
	"DailyStatus":                   HistoryDailyStatus,
	"ControlIQPcmChange":            HistoryControlIQPcmChange,
	"ControlIQUserModeChange":       HistoryControlIQUserModeChange,
	"ControlIQCycle":                HistoryControlIQCycle,
	"DexcomG7CGM":                   HistoryDexcomG7CGM,
	"Malfunction":                   HistoryMalfunction,
	"PlgsPeriodic":                  HistoryPlgsPeriodic,
	"ShelfMode":                     HistoryShelfMode,
	"ArmInit":                       HistoryArmInit,
	"AlertCleared":                  HistoryAlertCleared,
	"VersionInfo":                   HistoryVersionInfo,
	"UpdateStatus":                  HistoryUpdateStatus,
	"MaxBasalLimitSet":              HistoryMaxBasalLimitSet,
	"VersionsA":                     HistoryVersionsA,
	"MaxBolusLimitSet":              HistoryMaxBolusLimitSet,
	"ControlIQGlucoseModel":         HistoryControlIQGlucoseModel,
	"MalfunctionAck":                HistoryMalfunctionAck,
	"ReminderSnoozed":               HistoryReminderSnoozed,
	"CartridgeRemoved":              HistoryCartridgeRemoved,
	"CartridgeInserted":             HistoryCartridgeInserted,
	"ConfirmCartridgeFilled":        HistoryConfirmCartridgeFilled,
	"FillEstimateFinal":             HistoryFillEstimateFinal,
	"BasalIqSettingsChange":         HistoryBasalIQSettingsChange,
	"CgmTransmitterId":              HistoryCgmTransmitterID,
	"CgmAnnuSettings":               HistoryCgmAnnuSettings,
	"CgmStopSessionMsg1":            HistoryCgmStopSessionMsg1,
	"CgmStopSessionMsg2":            HistoryCgmStopSessionMsg2,
	"CgmHgaSettings":                HistoryCgmHgaSettings,
	"CgmLgaSettings":                HistoryCgmLgaSettings,
	"CgmRraSettings":                HistoryCgmRraSettings,
	"CgmFraSettings":                HistoryCgmFraSettings,
	"CgmOorSettings":                HistoryCgmOorSettings,
	"CgmAlertAck":                   HistoryCgmAlertAck,
	"CgmUnexpectedGeAlert":          HistoryCgmUnexpectedGeAlert,
	"CgmInactiveGx":                 HistoryCgmInactiveGx,
	"CgmTransmitterIdGx":            HistoryCgmTransmitterIDGx,
	"CgmStartSessionReqGx":          HistoryCgmStartSessionReqGx,
	"CgmStopSessionReqGx":           HistoryCgmStopSessionReqGx,
	"CgmStateChangeGx":              HistoryCgmStateChangeGx,
	"CgmTransmitterVersionGx":       HistoryCgmTransmitterVersionGx,
	"AaSleepScheduleChange":         HistoryAaSleepScheduleChange,
	"ControlIQAutomatedInsulin":     HistoryControlIQAutomatedInsulin,
	"AaDeliveryStatusChange":        HistoryAaDeliveryStatusChange,
	"AaEnableSettingChange":         HistoryAaEnableSettingChange,
	"AaTdiSettingChange":            HistoryAaTdiSettingChange,
	"AaWeightSettingChange":         HistoryAaWeightSettingChange,
	"CgmSessionTypeChange":          HistoryCgmSessionTypeChange,
	"WumpOcclusionDebug":            HistoryWumpOcclusionDebug,
	"SnoozeActivated":               HistorySnoozeActivated,
	"AaAutoBolusRejected":           HistoryAaAutoBolusRejected,
	"TipscReqPrimeCannula":          HistoryTipscReqPrimeCannula,
	"WumpCartridgeFilled":           HistoryWumpCartridgeFilled,
	"WumpCartridgeRemoved":          HistoryWumpCartridgeRemoved,
	"AAExerciseTimeChange":          HistoryAAExerciseTimeChange,
	"AAExerciseChoiceChange":        HistoryAAExerciseChoiceChange,
	"AATdiEstChange":                HistoryAATdiEstChange,
	"ControlIQFeatures":             HistoryControlIQFeatures,
	"PrimeInprocess":                HistoryPrimeInprocess,
	"CgmRejoinSession":              HistoryCgmRejoinSession,
	"CgmSensorTypeChange":           HistoryCGMSensorTypeChange,
	"CgmStartSensorReqG7":           HistoryCgmStartSensorReqG7,
	"CgmCalibrationG7":              HistoryCgmCalibrationG7,
	"CgmBleCalibrationEvtG7":        HistoryCgmBleCalibrationEvtG7,
	"CgmInactiveG7":                 HistoryCgmInactiveG7,
	"CgmStopSessionReqG7":           HistoryCgmStopSessionReqG7,
	"BleSoftDeviceVersion":          HistoryBleSoftDeviceVersion,
	"AlarmAck":                      HistoryAlarmAck,
	"AlertAck":                      HistoryAlertAck,
	"DismissNotificationResponse":   HistoryDismissNotificationResponse,
	"AamAnnunciation":               HistoryAamAnnunciation,
	"ReminderActivated":             HistoryReminderActivated,
	"ReminderDismissed":             HistoryReminderDismissed,
	"HistoryRequestCounters":        HistoryHistoryRequestCounters,
	"HistoryRequestRange":           HistoryHistoryRequestRange,
	"ControlRequestCounters":        HistoryControlRequestCounters,
	"ControlStreamCounters":         HistoryControlStreamCounters,
	"AuthorizationCounters":         HistoryAuthorizationCounters,
	"CgmPairingCodeG7":              HistoryCgmPairingCodeG7,
	"BleLinkStats":                  HistoryBleLinkStats,
	"BleDailyConnections":           HistoryBleDailyConnections,
	"TipsError":                     HistoryTipsError,
	"SyncDoseStart":                 HistorySyncDoseStart,
	"SyncDoseComplete":              HistorySyncDoseComplete,
	"AntDailyStatus1":               HistoryAntDailyStatus1,
	"AntDailyStatus2":               HistoryAntDailyStatus2,
	"TempAdjustmentCompleted":       HistoryTempAdjustmentCompleted,
	"TempAdjustmentActivated":       HistoryTempAdjustmentActivated,
	"AntSleepEatSchedSettingChange": HistoryAntSleepEatSchedSettingChange,
}

// HistoryTypeIDByName returns the numeric history-log type ID for a record
// type name, and whether the name is known.
func HistoryTypeIDByName(name string) (int, bool) {
	id, ok := historyTypeIDByName[name]
	return id, ok
}
