package currentstatus

import (
	"fmt"
	"math/big"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// AlertStatusResponse is the pump's active alerts: a 64-bit mask with an
// AlertResponseType at each bit.
type AlertStatusResponse struct {
	messages.Base
	// IntMap is the mask, which pumpX2 leaves out of toString.
	IntMap *big.Int
	Alerts []AlertResponseType `pumpx2:"alerts"`
}

// pumpX2 pairs it with AlarmStatusRequest, not AlertStatusRequest.
var alertStatusResponseProps = messages.Props{OpCode: 69, Size: 8, Type: messages.Response, Request: "AlarmStatusRequest"}

func init() {
	messages.Register("AlertStatusResponse", &AlertStatusResponse{}, alertStatusResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewAlertStatusResponseFromCargo},
		messages.Constructor{Params: "BigInteger intMap", Func: NewAlertStatusResponse},
	)
}

// NewAlertStatusResponse is the response with the low 64 bits of intMap.
func NewAlertStatusResponse(intMap *big.Int) *AlertStatusResponse {
	m := &AlertStatusResponse{}
	m.decode(helpers.ToUint64(helpers.LongValue(intMap)))
	return m
}

// NewAlertStatusResponseFromCargo parses a response's payload.
func NewAlertStatusResponseFromCargo(raw []byte) (*AlertStatusResponse, error) {
	m := &AlertStatusResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *AlertStatusResponse) Parse(raw []byte) error {
	if len(raw) != alertStatusResponseProps.Size {
		return messages.InvalidLength("AlertStatusResponse", len(raw), alertStatusResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *AlertStatusResponse) decode(raw []byte) {
	m.IntMap = helpers.ReadUint64(raw, 0)
	m.SetCargo(raw)
	m.Alerts = AlertResponseTypeFromBitmask(m.IntMap)
}

// BitMap is the mask as a signed 64-bit number.
func (m *AlertStatusResponse) BitMap() int64 { return helpers.LongValue(m.IntMap) }

// Size is how many alerts are active.
func (m *AlertStatusResponse) Size() int { return len(m.Alerts) }

// NotificationIDs are the active alerts pumpX2 has a description for.
func (m *AlertStatusResponse) NotificationIDs() []int32 {
	ids := []int32{}
	for _, a := range m.Alerts {
		if a.IsKnown() {
			ids = append(ids, int32(a))
		}
	}
	return ids
}

// AlertResponseType is a pump alert, as its bit in AlertStatusResponse's
// mask. No alert has bit 31.
type AlertResponseType int32

// The alerts, by their bits.
const (
	AlertResponseTypeLowInsulinAlert                AlertResponseType = 0
	AlertResponseTypeUSBConnectionAlert             AlertResponseType = 1
	AlertResponseTypeLowPowerAlert                  AlertResponseType = 2
	AlertResponseTypeLowPowerAlert2                 AlertResponseType = 3
	AlertResponseTypeDataErrorAlert                 AlertResponseType = 4
	AlertResponseTypeAutoOffAlert                   AlertResponseType = 5
	AlertResponseTypeMaxBasalRateAlert              AlertResponseType = 6
	AlertResponseTypePowerSourceAlert               AlertResponseType = 7
	AlertResponseTypeMinBasalAlert                  AlertResponseType = 8
	AlertResponseTypeConnectionErrorAlert           AlertResponseType = 9
	AlertResponseTypeConnectionErrorAlert2          AlertResponseType = 10
	AlertResponseTypeIncompleteBolusAlert           AlertResponseType = 11
	AlertResponseTypeIncompleteTempRateAlert        AlertResponseType = 12
	AlertResponseTypeIncompleteCartridgeChangeAlert AlertResponseType = 13
	AlertResponseTypeIncompleteFillTubingAlert      AlertResponseType = 14
	AlertResponseTypeIncompleteFillCannulaAlert     AlertResponseType = 15
	AlertResponseTypeIncompleteSettingAlert         AlertResponseType = 16
	AlertResponseTypeLowInsulinAlert2               AlertResponseType = 17
	AlertResponseTypeMaxBasalAlert                  AlertResponseType = 18
	AlertResponseTypeLowTransmitterAlert            AlertResponseType = 19
	AlertResponseTypeTransmitterAlert               AlertResponseType = 20
	AlertResponseTypeFsl2SupportPackageErrorAlert   AlertResponseType = 21
	AlertResponseTypeSensorExpiringAlert            AlertResponseType = 22
	AlertResponseTypePumpRebootingAlert             AlertResponseType = 23
	AlertResponseTypeDeviceConnectionError          AlertResponseType = 24
	AlertResponseTypeCGMGraphRemoved                AlertResponseType = 25
	AlertResponseTypeMinBasalAlert2                 AlertResponseType = 26
	AlertResponseTypeIncompleteCalibration          AlertResponseType = 27
	AlertResponseTypeCalibrationTimeout             AlertResponseType = 28
	AlertResponseTypeInvalidTransmitterID           AlertResponseType = 29
	AlertResponseTypeDebugInfoDeclaredAlert         AlertResponseType = 30
	AlertResponseTypeOpenSesameAlert                AlertResponseType = 32
	AlertResponseTypeButtonAlert                    AlertResponseType = 33
	AlertResponseTypeQuickBolusAlert                AlertResponseType = 34
	AlertResponseTypeBasalIQAlert                   AlertResponseType = 35
	AlertResponseTypeStkExchangeReqAlert            AlertResponseType = 36
	AlertResponseTypeStkOdysseyReqAlert             AlertResponseType = 37
	AlertResponseTypeStkInvalidAlert                AlertResponseType = 38
	AlertResponseTypeTransmitterEndOfLife           AlertResponseType = 39
	AlertResponseTypeCGMError                       AlertResponseType = 40
	AlertResponseTypeCGMError2                      AlertResponseType = 41
	AlertResponseTypeCGMError3                      AlertResponseType = 42
	AlertResponseTypeCGMIncompatibleAlert           AlertResponseType = 43
	AlertResponseTypeTransmitterExpiringAlert       AlertResponseType = 44
	AlertResponseTypeTransmitterExpiringAlert2      AlertResponseType = 45
	AlertResponseTypeTransmitterExpiringAlert3      AlertResponseType = 46
	AlertResponseTypeTzIncompleteSettingAlert       AlertResponseType = 47
	AlertResponseTypeCGMUnavailable                 AlertResponseType = 48
	AlertResponseTypeFillTubingStillInProgress      AlertResponseType = 49
	AlertResponseTypeTzHyperLightAlert              AlertResponseType = 50
	AlertResponseTypeControlIQLow                   AlertResponseType = 51
	AlertResponseTypeTzIidmAlert                    AlertResponseType = 52
	AlertResponseTypeBLECrashDumpAlert              AlertResponseType = 53
	AlertResponseTypeDevicePaired                   AlertResponseType = 54
	AlertResponseTypeBLEHwErrorAlert                AlertResponseType = 55
	AlertResponseTypeUserSetMaxBasalAlert           AlertResponseType = 56
	AlertResponseTypeSensorAlert                    AlertResponseType = 57
	AlertResponseTypeComLogErrorLoggedAlert         AlertResponseType = 58
	AlertResponseTypeFclAutomationOffAlert          AlertResponseType = 59
	AlertResponseTypeControllerConnectionLostAlert  AlertResponseType = 60
	AlertResponseTypeLipoSuperLowAlert              AlertResponseType = 61
	AlertResponseTypeLipoInsuffPowerAlert           AlertResponseType = 62
	AlertResponseTypeBtlePairingTimeoutAlert        AlertResponseType = 63
)

// AlertResponseTypes are the alerts in pumpX2's declaration order.
var AlertResponseTypes = []AlertResponseType{
	AlertResponseTypeLowInsulinAlert, AlertResponseTypeUSBConnectionAlert, AlertResponseTypeLowPowerAlert,
	AlertResponseTypeLowPowerAlert2, AlertResponseTypeDataErrorAlert, AlertResponseTypeAutoOffAlert,
	AlertResponseTypeMaxBasalRateAlert, AlertResponseTypePowerSourceAlert, AlertResponseTypeMinBasalAlert,
	AlertResponseTypeConnectionErrorAlert, AlertResponseTypeConnectionErrorAlert2, AlertResponseTypeIncompleteBolusAlert,
	AlertResponseTypeIncompleteTempRateAlert, AlertResponseTypeIncompleteCartridgeChangeAlert,
	AlertResponseTypeIncompleteFillTubingAlert, AlertResponseTypeIncompleteFillCannulaAlert,
	AlertResponseTypeIncompleteSettingAlert, AlertResponseTypeLowInsulinAlert2, AlertResponseTypeMaxBasalAlert,
	AlertResponseTypeLowTransmitterAlert, AlertResponseTypeTransmitterAlert,
	AlertResponseTypeFsl2SupportPackageErrorAlert, AlertResponseTypeSensorExpiringAlert,
	AlertResponseTypePumpRebootingAlert, AlertResponseTypeDeviceConnectionError, AlertResponseTypeCGMGraphRemoved,
	AlertResponseTypeMinBasalAlert2, AlertResponseTypeIncompleteCalibration, AlertResponseTypeCalibrationTimeout,
	AlertResponseTypeInvalidTransmitterID, AlertResponseTypeDebugInfoDeclaredAlert, AlertResponseTypeOpenSesameAlert,
	AlertResponseTypeButtonAlert, AlertResponseTypeQuickBolusAlert, AlertResponseTypeBasalIQAlert,
	AlertResponseTypeStkExchangeReqAlert, AlertResponseTypeStkOdysseyReqAlert, AlertResponseTypeStkInvalidAlert,
	AlertResponseTypeTransmitterEndOfLife, AlertResponseTypeCGMError, AlertResponseTypeCGMError2,
	AlertResponseTypeCGMError3, AlertResponseTypeCGMIncompatibleAlert, AlertResponseTypeTransmitterExpiringAlert,
	AlertResponseTypeTransmitterExpiringAlert2, AlertResponseTypeTransmitterExpiringAlert3,
	AlertResponseTypeTzIncompleteSettingAlert, AlertResponseTypeCGMUnavailable,
	AlertResponseTypeFillTubingStillInProgress, AlertResponseTypeTzHyperLightAlert, AlertResponseTypeControlIQLow,
	AlertResponseTypeTzIidmAlert, AlertResponseTypeBLECrashDumpAlert, AlertResponseTypeDevicePaired,
	AlertResponseTypeBLEHwErrorAlert, AlertResponseTypeUserSetMaxBasalAlert, AlertResponseTypeSensorAlert,
	AlertResponseTypeComLogErrorLoggedAlert, AlertResponseTypeFclAutomationOffAlert,
	AlertResponseTypeControllerConnectionLostAlert, AlertResponseTypeLipoSuperLowAlert,
	AlertResponseTypeLipoInsuffPowerAlert, AlertResponseTypeBtlePairingTimeoutAlert,
}

var alertResponseTypeNames = [64]string{
	0:  "LOW_INSULIN_ALERT",
	1:  "USB_CONNECTION_ALERT",
	2:  "LOW_POWER_ALERT",
	3:  "LOW_POWER_ALERT2",
	4:  "DATA_ERROR_ALERT",
	5:  "AUTO_OFF_ALERT",
	6:  "MAX_BASAL_RATE_ALERT",
	7:  "POWER_SOURCE_ALERT",
	8:  "MIN_BASAL_ALERT",
	9:  "CONNECTION_ERROR_ALERT",
	10: "CONNECTION_ERROR_ALERT2",
	11: "INCOMPLETE_BOLUS_ALERT",
	12: "INCOMPLETE_TEMP_RATE_ALERT",
	13: "INCOMPLETE_CARTRIDGE_CHANGE_ALERT",
	14: "INCOMPLETE_FILL_TUBING_ALERT",
	15: "INCOMPLETE_FILL_CANNULA_ALERT",
	16: "INCOMPLETE_SETTING_ALERT",
	17: "LOW_INSULIN_ALERT2",
	18: "MAX_BASAL_ALERT",
	19: "LOW_TRANSMITTER_ALERT",
	20: "TRANSMITTER_ALERT",
	21: "FSL2_SUPPORT_PACKAGE_ERROR_ALERT",
	22: "SENSOR_EXPIRING_ALERT",
	23: "PUMP_REBOOTING_ALERT",
	24: "DEVICE_CONNECTION_ERROR",
	25: "CGM_GRAPH_REMOVED",
	26: "MIN_BASAL_ALERT2",
	27: "INCOMPLETE_CALIBRATION",
	28: "CALIBRATION_TIMEOUT",
	29: "INVALID_TRANSMITTER_ID",
	30: "DEBUG_INFO_DECLARED_ALERT",
	32: "OPEN_SESAME_ALERT",
	33: "BUTTON_ALERT",
	34: "QUICK_BOLUS_ALERT",
	35: "BASAL_IQ_ALERT",
	36: "STK_EXCHANGE_REQ_ALERT",
	37: "STK_ODYSSEY_REQ_ALERT",
	38: "STK_INVALID_ALERT",
	39: "TRANSMITTER_END_OF_LIFE",
	40: "CGM_ERROR",
	41: "CGM_ERROR2",
	42: "CGM_ERROR3",
	43: "CGM_INCOMPATIBLE_ALERT",
	44: "TRANSMITTER_EXPIRING_ALERT",
	45: "TRANSMITTER_EXPIRING_ALERT2",
	46: "TRANSMITTER_EXPIRING_ALERT3",
	47: "TZ_INCOMPLETE_SETTING_ALERT",
	48: "CGM_UNAVAILABLE",
	49: "FILL_TUBING_STILL_IN_PROGRESS",
	50: "TZ_HYPER_LIGHT_ALERT",
	51: "CONTROL_IQ_LOW",
	52: "TZ_IIDM_ALERT",
	53: "BLE_CRASH_DUMP_ALERT",
	54: "DEVICE_PAIRED",
	55: "BLE_HW_ERROR_ALERT",
	56: "USER_SET_MAX_BASAL_ALERT",
	57: "SENSOR_ALERT",
	58: "COM_LOG_ERROR_LOGGED_ALERT",
	59: "FCL_AUTOMATION_OFF_ALERT",
	60: "CONTROLLER_CONNECTION_LOST_ALERT",
	61: "LIPO_SUPER_LOW_ALERT",
	62: "LIPO_INSUFF_POWER_ALERT",
	63: "BTLE_PAIRING_TIMEOUT_ALERT",
}

var alertResponseTypeDescriptions = [64]string{
	AlertResponseTypeLowInsulinAlert:                "Low amount of insulin remaining in the cartridge.",
	AlertResponseTypeUSBConnectionAlert:             "Pump is not charging over USB.",
	AlertResponseTypeLowPowerAlert:                  "Power level is low and the pump needs to be charged.",
	AlertResponseTypeLowPowerAlert2:                 "Power level is low and the pump needs to be charged.",
	AlertResponseTypeAutoOffAlert:                   "The pump is about to turn off due to the configured auto-off interval.",
	AlertResponseTypeMaxBasalRateAlert:              "The pump is delivering at the maximum allowed basal rate.",
	AlertResponseTypePowerSourceAlert:               "The power source provided is not able to charge the pump.",
	AlertResponseTypeMinBasalAlert:                  "The pump is delivering at the minimum allowed basal rate.",
	AlertResponseTypeIncompleteBolusAlert:           "The bolus window was opened but a bolus was not started.",
	AlertResponseTypeIncompleteTempRateAlert:        "The temp rate window was opened but the temp rate was not started.",
	AlertResponseTypeIncompleteCartridgeChangeAlert: "The cartridge change was started but not completed.",
	AlertResponseTypeIncompleteFillTubingAlert:      "Fill tubing was started but not completed.",
	AlertResponseTypeIncompleteFillCannulaAlert:     "Fill cannula was started but not completed.",
	AlertResponseTypeLowInsulinAlert2:               "Low amount of insulin remaining in the cartridge.",
	AlertResponseTypeMaxBasalAlert:                  "The maxmium basal was reached.",
	AlertResponseTypeLowTransmitterAlert:            "The CGM transmitter battery is low.",
	AlertResponseTypeTransmitterAlert:               "There is an alert from the CGM transmitter.",
	AlertResponseTypeSensorExpiringAlert:            "The CGM sensor is expiring soon.",
	AlertResponseTypePumpRebootingAlert:             "The pump is rebooting.",
	AlertResponseTypeCGMGraphRemoved:                "It has been 24 hours since your last sensor session ended, so your current glucose reading now displays the last glucose value entered in the bolus calculator.",
	AlertResponseTypeMinBasalAlert2:                 "The pump is delivering at the minimum allowed basal rate.",
	AlertResponseTypeIncompleteCalibration:          "The CGM calibration was incomplete.",
	AlertResponseTypeCalibrationTimeout:             "The timeout was reached for CGM calibration.",
	AlertResponseTypeInvalidTransmitterID:           "The Dexcom G6 or G7 transmitter ID is invalid.",
	AlertResponseTypeButtonAlert:                    "The pump button was held down for too long and is temporarily disabled to avoid accidental delivery of insulin.",
	AlertResponseTypeQuickBolusAlert:                "Quick bolus mode was entered but no bolus was started.",
	AlertResponseTypeBasalIQAlert:                   "BasalIQ has reduced basal insulin to avoid a low.",
	AlertResponseTypeTransmitterEndOfLife:           "The CGM transmitter is reaching its end of life and cannot be used.",
	AlertResponseTypeCGMError:                       "CGM error reported",
	AlertResponseTypeCGMError2:                      "CGM error reported",
	AlertResponseTypeCGMError3:                      "CGM error reported",
	AlertResponseTypeTransmitterExpiringAlert:       "The CGM transmitter is expiring soon.",
	AlertResponseTypeTransmitterExpiringAlert2:      "The CGM transmitter is expiring soon.",
	AlertResponseTypeTransmitterExpiringAlert3:      "The CGM transmitter is expiring soon.",
	AlertResponseTypeCGMUnavailable:                 "The CGM is unavailable due to a problem with the sensor.",
	AlertResponseTypeFillTubingStillInProgress:      "The fill tubing process is still in progress and was not completed.",
	AlertResponseTypeControlIQLow:                   "Control-IQ has reduced basal insulin delivery due to a low or predicted low glucose value.",
	AlertResponseTypeDevicePaired:                   "The pump was paired successfully to a Bluetooth device.",
}

// AlertResponseTypeFromBitmask is the alerts whose bits intMap sets, in
// declaration order; never nil.
func AlertResponseTypeFromBitmask(intMap *big.Int) []AlertResponseType {
	out := []AlertResponseType{}
	for _, t := range AlertResponseTypes {
		if intMap.Bit(int(t)) != 0 {
			out = append(out, t)
		}
	}
	return out
}

// AlertResponseTypeFromSingularID is the alert at bit id, or nil.
func AlertResponseTypeFromSingularID(id int64) *AlertResponseType {
	if id < 0 || id >= int64(len(alertResponseTypeNames)) || alertResponseTypeNames[id] == "" {
		return nil
	}
	t := AlertResponseType(id)
	return &t
}

// WithBit is the mask with only the alert's bit set.
func (t AlertResponseType) WithBit() *big.Int {
	return new(big.Int).SetBit(new(big.Int), int(t), 1)
}

// Description is what the alert means, for the alerts pumpX2 describes, or
// "".
func (t AlertResponseType) Description() string {
	if t >= 0 && int(t) < len(alertResponseTypeDescriptions) {
		return alertResponseTypeDescriptions[t]
	}
	return ""
}

// IsKnown reports whether pumpX2 describes the alert.
func (t AlertResponseType) IsKnown() bool { return t.Description() != "" }

func (t AlertResponseType) String() string {
	if t >= 0 && int(t) < len(alertResponseTypeNames) && alertResponseTypeNames[t] != "" {
		return alertResponseTypeNames[t]
	}
	return fmt.Sprintf("AlertResponseType(%d)", int32(t))
}
