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

	// IDP events
	HistoryIDPAction  = 69
	HistoryIDPBolus   = 70
	HistoryIDPList    = 71
	HistoryIDPSegment = 68

	// Hypo protection
	HistoryHypoMinimizerSuspend = 198
	HistoryHypoMinimizerResume  = 199

	// Daily/status. There is no LoadStatus history log in pumpX2 or TandemKit
	// (LoadStatus is a CURRENT_STATUS message, not a log record), so the
	// constant that used to sit here has been removed rather than given a
	// guessed ID.
	HistoryDailyStatus  = 313
	HistoryUpdateStatus = 203
)
