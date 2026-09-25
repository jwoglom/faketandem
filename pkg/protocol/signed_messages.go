// Code generated from TandemKit's TandemCore message catalog. DO NOT EDIT.
//
// Regenerate with:
//
//	go test ./pkg/protocol -run TestSignedMessageTableMatchesTandemKit -update
//
// See signed_messages_test.go for the derivation and for the guard that fails
// when this table and the catalog disagree.

package protocol

import "sort"

// signedMessages lists every message whose MessageProps declares signed: true,
// i.e. every message that carries the 24-byte timeSinceReset+HMAC-SHA1 trailer
// described in BuildMessageBody.
//
// It covers requests as well as responses. Only the responses matter to the
// emulator (it never sends a request), but the catalog marks both halves of a
// control exchange signed and splitting them here would invite the table and
// the catalog to drift apart.
var signedMessages = map[string]bool{
	"ActivateShelfModeRequest":                              true,
	"ActivateShelfModeResponse":                             true,
	"AdditionalBolusRequest":                                true,
	"AdditionalBolusResponse":                               true,
	"BolusPermissionReleaseRequest":                         true,
	"BolusPermissionReleaseResponse":                        true,
	"BolusPermissionRequest":                                true,
	"BolusPermissionResponse":                               true,
	"CancelBolusRequest":                                    true,
	"CancelBolusResponse":                                   true,
	"CgmHighLowAlertRequest":                                true,
	"CgmHighLowAlertResponse":                               true,
	"CgmOutOfRangeAlertRequest":                             true,
	"CgmOutOfRangeAlertResponse":                            true,
	"CgmRiseFallAlertRequest":                               true,
	"CgmRiseFallAlertResponse":                              true,
	"ChangeControlIQSettingsRequest":                        true,
	"ChangeControlIQSettingsResponse":                       true,
	"ChangeTimeDateRequest":                                 true,
	"ChangeTimeDateResponse":                                true,
	"CreateIDPRequest":                                      true,
	"CreateIDPResponse":                                     true,
	"DeleteIDPRequest":                                      true,
	"DeleteIDPResponse":                                     true,
	"DetectingCartridgeStateStreamResponse":                 true,
	"DisconnectPumpRequest":                                 true,
	"DisconnectPumpResponse":                                true,
	"DismissNotificationRequest":                            true,
	"DismissNotificationResponse":                           true,
	"EnterChangeCartridgeModeRequest":                       true,
	"EnterChangeCartridgeModeResponse":                      true,
	"EnterChangeCartridgeModeStateStreamResponse":           true,
	"EnterFillTubingModeRequest":                            true,
	"EnterFillTubingModeResponse":                           true,
	"ErrorResponse":                                         true,
	"ExitChangeCartridgeModeRequest":                        true,
	"ExitChangeCartridgeModeResponse":                       true,
	"ExitFillTubingModeRequest":                             true,
	"ExitFillTubingModeResponse":                            true,
	"ExitFillTubingModeStateStreamResponse":                 true,
	"FactoryResetBRequest":                                  true,
	"FactoryResetBResponse":                                 true,
	"FactoryResetRequest":                                   true,
	"FactoryResetResponse":                                  true,
	"FillCannulaRequest":                                    true,
	"FillCannulaResponse":                                   true,
	"FillCannulaStateStreamResponse":                        true,
	"FillTubingStateStreamResponse":                         true,
	"InitiateBolusRequest":                                  true,
	"InitiateBolusResponse":                                 true,
	"LoadCartridgeStateStreamResponse":                      true,
	"NonexistentDetectingCartridgeStateStreamRequest":       true,
	"NonexistentEnterChangeCartridgeModeStateStreamRequest": true,
	"NonexistentExitFillTubingModeStateStreamRequest":       true,
	"NonexistentFillCannulaStateStreamRequest":              true,
	"NonexistentFillTubingStateStreamRequest":               true,
	"NonexistentLoadCartridgeStateStreamRequest":            true,
	"NonexistentPrimeNudgeStateStreamRequest":               true,
	"NonexistentPumpingStateStreamRequest":                  true,
	"PlaySoundRequest":                                      true,
	"PlaySoundResponse":                                     true,
	"PrimeNudgeStateStreamResponse":                         true,
	"PrimeTubingSuspendRequest":                             true,
	"PrimeTubingSuspendResponse":                            true,
	"PumpingStateStreamResponse":                            true,
	"RemoteBgEntryRequest":                                  true,
	"RemoteBgEntryResponse":                                 true,
	"RemoteCarbEntryRequest":                                true,
	"RemoteCarbEntryResponse":                               true,
	"RenameIDPRequest":                                      true,
	"RenameIDPResponse":                                     true,
	"ResumePumpingRequest":                                  true,
	"ResumePumpingResponse":                                 true,
	"SendTipsControlGenericTestRequest":                     true,
	"SendTipsControlGenericTestResponse":                    true,
	"SetActiveIDPRequest":                                   true,
	"SetActiveIDPResponse":                                  true,
	"SetAutoOffAlertRequest":                                true,
	"SetAutoOffAlertResponse":                               true,
	"SetBgReminderRequest":                                  true,
	"SetBgReminderResponse":                                 true,
	"SetDexcomG7PairingCodeRequest":                         true,
	"SetDexcomG7PairingCodeResponse":                        true,
	"SetG6TransmitterIdRequest":                             true,
	"SetG6TransmitterIdResponse":                            true,
	"SetIDPSegmentRequest":                                  true,
	"SetIDPSegmentResponse":                                 true,
	"SetIDPSettingsRequest":                                 true,
	"SetIDPSettingsResponse":                                true,
	"SetLowInsulinAlertRequest":                             true,
	"SetLowInsulinAlertResponse":                            true,
	"SetMaxBasalLimitRequest":                               true,
	"SetMaxBasalLimitResponse":                              true,
	"SetMaxBolusLimitRequest":                               true,
	"SetMaxBolusLimitResponse":                              true,
	"SetMissedMealBolusReminderRequest":                     true,
	"SetMissedMealBolusReminderResponse":                    true,
	"SetModesRequest":                                       true,
	"SetModesResponse":                                      true,
	"SetPumpAlertSnoozeRequest":                             true,
	"SetPumpAlertSnoozeResponse":                            true,
	"SetPumpSoundsRequest":                                  true,
	"SetPumpSoundsResponse":                                 true,
	"SetQuickBolusSettingsRequest":                          true,
	"SetQuickBolusSettingsResponse":                         true,
	"SetSensorTypeRequest":                                  true,
	"SetSensorTypeResponse":                                 true,
	"SetSiteChangeReminderRequest":                          true,
	"SetSiteChangeReminderResponse":                         true,
	"SetSleepScheduleRequest":                               true,
	"SetSleepScheduleResponse":                              true,
	"SetTempRateRequest":                                    true,
	"SetTempRateResponse":                                   true,
	"StartDexcomG6SensorSessionRequest":                     true,
	"StartDexcomG6SensorSessionResponse":                    true,
	"StopDexcomCGMSensorSessionRequest":                     true,
	"StopDexcomCGMSensorSessionResponse":                    true,
	"StopTempRateRequest":                                   true,
	"StopTempRateResponse":                                  true,
	"StreamDataPreflightRequest":                            true,
	"StreamDataPreflightResponse":                           true,
	"SuspendPumpingRequest":                                 true,
	"SuspendPumpingResponse":                                true,
	"UnknownMobiOpcodeNeg124Request":                        true,
	"UnknownMobiOpcodeNeg124Response":                       true,
	"UserInteractionRequest":                                true,
	"UserInteractionResponse":                               true,
}

// IsSignedMessage reports whether the named message carries the signed
// trailer. The name is a pumpX2/TandemKit message class name, exactly as
// cliparser prints it (e.g. "SuspendPumpingResponse").
//
// An unknown name reports false: a message nobody has a catalog entry for is
// sent as-is rather than given a trailer the driver would not expect.
func IsSignedMessage(name string) bool {
	return signedMessages[name]
}

// SignedMessageNames returns the table's contents, sorted, for tests and for
// anything that needs to enumerate the signed half of the catalog.
func SignedMessageNames() []string {
	names := make([]string, 0, len(signedMessages))
	for name := range signedMessages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
