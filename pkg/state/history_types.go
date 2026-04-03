package state

// History log event type IDs matching pumpX2's HistoryLog type IDs.
// These are the most commonly generated event types.
const (
	// Bolus events
	HistoryBolusRequestedMsg1 = 16
	HistoryBolusRequestedMsg2 = 17
	HistoryBolusRequestedMsg3 = 18
	HistoryBolusActivated     = 19
	HistoryBolusCompleted     = 20
	HistoryBolusDelivery      = 280
	HistoryBolexActivated     = 21
	HistoryBolexCompleted     = 22
	HistoryCorrectionDeclined = 23

	// Basal events
	HistoryBasalDelivery   = 32
	HistoryBasalRateChange = 33
	HistoryDailyBasal      = 34

	// Temp rate events
	HistoryTempRateActivated = 35
	HistoryTempRateCompleted = 36

	// Cartridge/tubing events
	HistoryCartridgeInserted = 40
	HistoryCartridgeRemoved  = 41
	HistoryCartridgeFilled   = 42
	HistoryTubingFilled      = 43
	HistoryCannulaFilled     = 44

	// Pump state events
	HistoryPumpingSuspended = 50
	HistoryPumpingResumed   = 51

	// Remote entry events
	HistoryBGEntry   = 55
	HistoryCarbEntry = 56

	// System events
	HistoryDateChange   = 14
	HistoryTimeChanged  = 15
	HistoryNewDay       = 60
	HistoryVersionInfo  = 61
	HistoryFactoryReset = 62
	HistoryLogErased    = 63

	// Alarm/Alert events
	HistoryAlarmActivated = 70
	HistoryAlarmAck       = 71
	HistoryAlarmCleared   = 72
	HistoryAlertActivated = 73
	HistoryAlertAck       = 74
	HistoryAlertCleared   = 75
	HistoryMalfunction    = 76

	// CGM events
	HistoryCGMData              = 100
	HistoryCGMCalibration       = 101
	HistoryCGMSensorTypeChange  = 102
	HistoryCGMStartSession      = 103
	HistoryCGMStopSession       = 104
	HistoryCGMAlertActivated    = 105
	HistoryCGMAlertCleared      = 106

	// Settings events
	HistoryParamChangeGlobalSettings = 110
	HistoryParamChangePumpSettings   = 111
	HistoryControlIQUserModeChange   = 112
	HistoryBasalIQSettingsChange     = 113

	// IDP events
	HistoryIDPAction  = 120
	HistoryIDPBolus   = 121
	HistoryIDPList    = 122
	HistoryIDPSegment = 123

	// Hypo protection
	HistoryHypoMinimizerSuspend = 130
	HistoryHypoMinimizerResume  = 131

	// Daily/status
	HistoryDailyStatus = 140
	HistoryLoadStatus  = 141
	HistoryUpdateStatus = 142
)
