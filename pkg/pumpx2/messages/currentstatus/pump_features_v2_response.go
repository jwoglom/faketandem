package currentstatus

import (
	"fmt"
	"math/big"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// PumpFeaturesV2Response is one of the pump's feature bitmasks, the one
// PumpFeaturesV2Request asked for by its SupportedFeatureIndex. Of the
// feature sets, only the one for that index is set; the rest are nil.
// Status is 0 on success.
type PumpFeaturesV2Response struct {
	messages.Base
	Status                  int32                     `pumpx2:"status"`
	SupportedFeatureIndexID int32                     `pumpx2:"supportedFeatureIndexId"`
	PumpFeaturesBitmask     int64                     `pumpx2:"pumpFeaturesBitmask"`
	PrimaryFeatures         []PumpFeatureType         `pumpx2:"primaryFeatures"`
	ControlIQProFeatures    []ControlIQProFeatureType `pumpx2:"controlIqProFeatures"`
	ControlFeatures         []ControlFeatureType      `pumpx2:"controlFeatures"`
	ControlIQFeatures       []ControlIQFeatureType    `pumpx2:"controlIqFeatures"`
	DexcomFeatures          []DexcomFeatureType       `pumpx2:"dexcomFeatures"`
	AbbottFeatures          []AbbottFeatureType       `pumpx2:"abbottFeatures"`
	SecondaryFeatures       []SecondaryFeatureType    `pumpx2:"secondaryFeatures"`
}

var pumpFeaturesV2ResponseProps = messages.Props{OpCode: -95, Size: 6, Type: messages.Response, MinAPI: "API_V2_5", Request: "PumpFeaturesV2Request"}

func init() {
	messages.Register("PumpFeaturesV2Response", &PumpFeaturesV2Response{}, pumpFeaturesV2ResponseProps,
		messages.Constructor{Params: "int status, int supportedFeatureIndex, long pumpFeaturesBitmask", Func: NewPumpFeaturesV2Response},
	)
}

// NewPumpFeaturesV2Response builds the response; supportedFeatureIndex is
// a SupportedFeatureIndex ID.
func NewPumpFeaturesV2Response(status, supportedFeatureIndex int32, pumpFeaturesBitmask int64) *PumpFeaturesV2Response {
	m := &PumpFeaturesV2Response{Status: status, SupportedFeatureIndexID: supportedFeatureIndex, PumpFeaturesBitmask: pumpFeaturesBitmask}
	m.SetCargo(helpers.Combine([]byte{byte(status), byte(supportedFeatureIndex)}, helpers.ToUint32(pumpFeaturesBitmask)))
	m.setFeatures()
	return m
}

// Parse implements messages.Message.
func (m *PumpFeaturesV2Response) Parse(raw []byte) error {
	if len(raw) != pumpFeaturesV2ResponseProps.Size {
		return messages.InvalidLength("PumpFeaturesV2Response", len(raw), pumpFeaturesV2ResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	m.SupportedFeatureIndexID = int32(int8(raw[1]))
	m.PumpFeaturesBitmask = helpers.ReadUint32(raw, 2)
	m.setFeatures()
	return nil
}

func (m *PumpFeaturesV2Response) setFeatures() {
	m.PrimaryFeatures, m.ControlIQProFeatures, m.ControlFeatures, m.ControlIQFeatures = nil, nil, nil, nil
	m.DexcomFeatures, m.AbbottFeatures, m.SecondaryFeatures = nil, nil, nil
	mask := m.PumpFeaturesBitmask
	switch m.SupportedFeatureIndex() {
	case SupportedFeatureIndexMainFeatures:
		m.PrimaryFeatures = PumpFeatureTypeFromBitmask(big.NewInt(mask))
	case SupportedFeatureIndexControlIQProFeatures:
		m.ControlIQProFeatures = ControlIQProFeatureTypeFromBitmask(mask)
	case SupportedFeatureIndexControlFeatures:
		m.ControlFeatures = ControlFeatureTypeFromBitmask(mask)
	case SupportedFeatureIndexControlIQFeatures:
		m.ControlIQFeatures = ControlIQFeatureTypeFromBitmask(mask)
	case SupportedFeatureIndexDexcomFeatures:
		m.DexcomFeatures = DexcomFeatureTypeFromBitmask(mask)
	case SupportedFeatureIndexAbbottFeatures:
		m.AbbottFeatures = AbbottFeatureTypeFromBitmask(mask)
	case SupportedFeatureIndexSecondaryFeatures:
		m.SecondaryFeatures = SecondaryFeatureTypeFromBitmask(mask)
	}
}

// SupportedFeatureIndex is the feature bitmask the response carries.
func (m *PumpFeaturesV2Response) SupportedFeatureIndex() SupportedFeatureIndex {
	return SupportedFeatureIndexFromID(m.SupportedFeatureIndexID)
}

// SupportedFeatureIndex names one of the pump's feature bitmasks.
type SupportedFeatureIndex int32

// The feature bitmasks, by their IDs.
const (
	SupportedFeatureIndexMainFeatures         SupportedFeatureIndex = 0
	SupportedFeatureIndexControlIQProFeatures SupportedFeatureIndex = 1
	SupportedFeatureIndexControlFeatures      SupportedFeatureIndex = 2
	SupportedFeatureIndexControlIQFeatures    SupportedFeatureIndex = 3
	SupportedFeatureIndexDexcomFeatures       SupportedFeatureIndex = 4
	SupportedFeatureIndexAbbottFeatures       SupportedFeatureIndex = 5
	SupportedFeatureIndexSecondaryFeatures    SupportedFeatureIndex = 6
	SupportedFeatureIndexUnknown              SupportedFeatureIndex = -1
)

var supportedFeatureIndexNames = []string{
	"MAIN_FEATURES", "CONTROL_IQ_PRO_FEATURES", "CONTROL_FEATURES", "CONTROL_IQ_FEATURES", "DEXCOM_FEATURES", "ABBOTT_FEATURES", "SECONDARY_FEATURES",
}

// SupportedFeatureIndexFromID is the bitmask with ID id, or Unknown.
func SupportedFeatureIndexFromID(id int32) SupportedFeatureIndex {
	if id >= 0 && int(id) < len(supportedFeatureIndexNames) {
		return SupportedFeatureIndex(id)
	}
	return SupportedFeatureIndexUnknown
}

// ID is the bitmask's ID.
func (s SupportedFeatureIndex) ID() int32 { return int32(s) }

func (s SupportedFeatureIndex) String() string {
	switch {
	case s == SupportedFeatureIndexUnknown:
		return "UNKNOWN"
	case s >= 0 && int(s) < len(supportedFeatureIndexNames):
		return supportedFeatureIndexNames[s]
	}
	return fmt.Sprintf("SupportedFeatureIndex(%d)", int32(s))
}

// ControlIQProFeatureType is a Control-IQ+ feature, as its bit of the
// CONTROL_IQ_PRO_FEATURES bitmask.
type ControlIQProFeatureType int32

// The Control-IQ+ features, by their bits.
const (
	ControlIQProFeatureBLEControl ControlIQProFeatureType = 1 << iota
	ControlIQProFeatureGUIChanges
	ControlIQProFeatureRangeChanges
	ControlIQProFeatureNVMParam
	ControlIQProFeatureExtendedBolusDuration
	ControlIQProFeatureAutoBolus
)

// ControlIQProFeatureTypes are the Control-IQ+ features in pumpX2's
// declaration order.
var ControlIQProFeatureTypes = []ControlIQProFeatureType{
	ControlIQProFeatureBLEControl, ControlIQProFeatureGUIChanges, ControlIQProFeatureRangeChanges, ControlIQProFeatureNVMParam,
	ControlIQProFeatureExtendedBolusDuration, ControlIQProFeatureAutoBolus,
}

var controlIQProFeatureTypeNames = map[ControlIQProFeatureType]string{
	ControlIQProFeatureBLEControl:            "BLE_CONTROL",
	ControlIQProFeatureGUIChanges:            "GUI_CHANGES",
	ControlIQProFeatureRangeChanges:          "RANGE_CHANGES",
	ControlIQProFeatureNVMParam:              "NVM_PARAM",
	ControlIQProFeatureExtendedBolusDuration: "EXTENDED_BOLUS_DURATION",
	ControlIQProFeatureAutoBolus:             "AUTO_BOLUS",
}

// ControlIQProFeatureTypeFromBitmask is the features whose bits mask sets,
// in declaration order; never nil.
func ControlIQProFeatureTypeFromBitmask(mask int64) []ControlIQProFeatureType {
	out := []ControlIQProFeatureType{}
	for _, f := range ControlIQProFeatureTypes {
		if mask&int64(f) != 0 {
			out = append(out, f)
		}
	}
	return out
}

func (f ControlIQProFeatureType) String() string {
	if name, ok := controlIQProFeatureTypeNames[f]; ok {
		return name
	}
	return fmt.Sprintf("ControlIQProFeatureType(%d)", int32(f))
}

// ControlFeatureType is something the pump lets a phone control, as its bit
// of the CONTROL_FEATURES bitmask.
type ControlFeatureType int32

// The control features, by their bits.
const (
	ControlFeatureChangeCartridgeControl ControlFeatureType = 1 << iota
	ControlFeatureBasalDeliveryControl
	ControlFeatureStandardBolusControl
	ControlFeatureExtendedBolusControl
	ControlFeatureTempRateControl
	ControlFeatureCGMControl
	ControlFeatureAAMControl
	ControlFeatureShelfModeControl
	ControlFeatureIDPControl
	ControlFeatureLEDPatternControl
	ControlFeatureUnregisterControl
	ControlFeaturePumpTimeControl
	ControlFeaturePLGSControl
	ControlFeatureTZControl
	ControlFeatureReminderSettingsControl
	ControlFeatureAlertSettingsControl
	ControlFeatureAnnumControl
	ControlFeatureAAMNotifyMeOnControl
	ControlFeaturePumpGlobalSettingsControl
	ControlFeatureFactoryConfigControl
	ControlFeatureSnoozeSettingsControl
	ControlFeatureUpdatedSettingsQEControl
	ControlFeatureFindMyPumpControl
	ControlFeatureBolusOverflowControl
	ControlFeaturePreflightControl
	ControlFeatureUserInteractionControl
	ControlFeatureBloodGlucoseEntryControl
	ControlFeatureHLogCreatorControl
	ControlFeatureFeatureManagement
	ControlFeatureMobiResearchPump
	ControlFeatureTipsControlGenericTest
)

// ControlFeatureTypes are the control features in pumpX2's declaration
// order.
var ControlFeatureTypes = []ControlFeatureType{
	ControlFeatureChangeCartridgeControl, ControlFeatureBasalDeliveryControl, ControlFeatureStandardBolusControl,
	ControlFeatureExtendedBolusControl, ControlFeatureTempRateControl, ControlFeatureCGMControl, ControlFeatureAAMControl,
	ControlFeatureShelfModeControl, ControlFeatureIDPControl, ControlFeatureLEDPatternControl, ControlFeatureUnregisterControl,
	ControlFeaturePumpTimeControl, ControlFeaturePLGSControl, ControlFeatureTZControl, ControlFeatureReminderSettingsControl,
	ControlFeatureAlertSettingsControl, ControlFeatureAnnumControl, ControlFeatureAAMNotifyMeOnControl,
	ControlFeaturePumpGlobalSettingsControl, ControlFeatureFactoryConfigControl, ControlFeatureSnoozeSettingsControl,
	ControlFeatureUpdatedSettingsQEControl, ControlFeatureFindMyPumpControl, ControlFeatureBolusOverflowControl,
	ControlFeaturePreflightControl, ControlFeatureUserInteractionControl, ControlFeatureBloodGlucoseEntryControl,
	ControlFeatureHLogCreatorControl, ControlFeatureFeatureManagement, ControlFeatureMobiResearchPump,
	ControlFeatureTipsControlGenericTest,
}

var controlFeatureTypeNames = map[ControlFeatureType]string{
	ControlFeatureChangeCartridgeControl:    "CHANGE_CARTRIDGE_CONTROL",
	ControlFeatureBasalDeliveryControl:      "BASAL_DELIVERY_CONTROL",
	ControlFeatureStandardBolusControl:      "STANDARD_BOLUS_CONTROL",
	ControlFeatureExtendedBolusControl:      "EXTENDED_BOLUS_CONTROL",
	ControlFeatureTempRateControl:           "TEMP_RATE_CONTROL",
	ControlFeatureCGMControl:                "CGM_CONTROL",
	ControlFeatureAAMControl:                "AAM_CONTROL",
	ControlFeatureShelfModeControl:          "SHELF_MODE_CONTROL",
	ControlFeatureIDPControl:                "IDP_CONTROL",
	ControlFeatureLEDPatternControl:         "LED_PATTERN_CONTROL",
	ControlFeatureUnregisterControl:         "UNREGISTER_CONTROL",
	ControlFeaturePumpTimeControl:           "PUMP_TIME_CONTROL",
	ControlFeaturePLGSControl:               "PLGS_CONTROL",
	ControlFeatureTZControl:                 "TZ_CONTROL",
	ControlFeatureReminderSettingsControl:   "REMINDER_SETTINGS_CONTROL",
	ControlFeatureAlertSettingsControl:      "ALERT_SETTINGS_CONTROL",
	ControlFeatureAnnumControl:              "ANNUM_CONTROL",
	ControlFeatureAAMNotifyMeOnControl:      "AAM_NOTIFY_ME_ON_CONTROL",
	ControlFeaturePumpGlobalSettingsControl: "PUMP_GLOBAL_SETTINGS_CONTROL",
	ControlFeatureFactoryConfigControl:      "FACTORY_CONFIG_CONTROL",
	ControlFeatureSnoozeSettingsControl:     "SNOOZE_SETTINGS_CONTROL",
	ControlFeatureUpdatedSettingsQEControl:  "UPDATED_SETTINGS_QE_CONTROL",
	ControlFeatureFindMyPumpControl:         "FIND_MY_PUMP_CONTROL",
	ControlFeatureBolusOverflowControl:      "BOLUS_OVERFLOW_CONTROL",
	ControlFeaturePreflightControl:          "PREFLIGHT_CONTROL",
	ControlFeatureUserInteractionControl:    "USER_INTERACTION_CONTROL",
	ControlFeatureBloodGlucoseEntryControl:  "BLOOD_GLUCOSE_ENTRY_CONTROL",
	ControlFeatureHLogCreatorControl:        "HLOG_CREATOR_CONTROL",
	ControlFeatureFeatureManagement:         "FEATURE_MANAGEMENT",
	ControlFeatureMobiResearchPump:          "MOBI_RESEARCH_PUMP",
	ControlFeatureTipsControlGenericTest:    "TIPS_CONTROL_GENERIC_TEST",
}

// ControlFeatureTypeFromBitmask is the features whose bits mask sets, in
// declaration order; never nil.
func ControlFeatureTypeFromBitmask(mask int64) []ControlFeatureType {
	out := []ControlFeatureType{}
	for _, f := range ControlFeatureTypes {
		if mask&int64(f) != 0 {
			out = append(out, f)
		}
	}
	return out
}

func (f ControlFeatureType) String() string {
	if name, ok := controlFeatureTypeNames[f]; ok {
		return name
	}
	return fmt.Sprintf("ControlFeatureType(%d)", int32(f))
}

// ControlIQFeatureType is a Control-IQ feature, as its bit of the
// CONTROL_IQ_FEATURES bitmask.
type ControlIQFeatureType int32

// The Control-IQ features, by their bits.
const (
	ControlIQFeatureTimedExercise           ControlIQFeatureType = 1 << 0
	ControlIQFeatureControlIQVersionOneFive ControlIQFeatureType = 1 << 8
	ControlIQFeatureTempRateEnabled         ControlIQFeatureType = 1 << 9
	ControlIQFeatureExtendedBolusEnabled    ControlIQFeatureType = 1 << 11
)

// ControlIQFeatureTypes are the Control-IQ features in pumpX2's declaration
// order.
var ControlIQFeatureTypes = []ControlIQFeatureType{
	ControlIQFeatureTimedExercise, ControlIQFeatureControlIQVersionOneFive, ControlIQFeatureTempRateEnabled, ControlIQFeatureExtendedBolusEnabled,
}

var controlIQFeatureTypeNames = map[ControlIQFeatureType]string{
	ControlIQFeatureTimedExercise:           "TIMED_EXERCISE",
	ControlIQFeatureControlIQVersionOneFive: "CONTROL_IQ_VERSION_ONE_FIVE",
	ControlIQFeatureTempRateEnabled:         "TEMP_RATE_ENABLED",
	ControlIQFeatureExtendedBolusEnabled:    "EXTENDED_BOLUS_ENABLED",
}

// ControlIQFeatureTypeFromBitmask is the features whose bits mask sets, in
// declaration order; never nil.
func ControlIQFeatureTypeFromBitmask(mask int64) []ControlIQFeatureType {
	out := []ControlIQFeatureType{}
	for _, f := range ControlIQFeatureTypes {
		if mask&int64(f) != 0 {
			out = append(out, f)
		}
	}
	return out
}

func (f ControlIQFeatureType) String() string {
	if name, ok := controlIQFeatureTypeNames[f]; ok {
		return name
	}
	return fmt.Sprintf("ControlIQFeatureType(%d)", int32(f))
}

// DexcomFeatureType is a Dexcom CGM feature, as its bit of the
// DEXCOM_FEATURES bitmask.
type DexcomFeatureType int32

// The Dexcom features, by their bits.
const (
	DexcomFeatureG5CGM DexcomFeatureType = 1 << iota
	DexcomFeatureG6CGM
	DexcomFeatureG7CGM
	DexcomFeatureAutoCal
	DexcomFeatureDisableGraphSmoothing
)

// DexcomFeatureTypes are the Dexcom features in pumpX2's declaration order.
var DexcomFeatureTypes = []DexcomFeatureType{
	DexcomFeatureG5CGM, DexcomFeatureG6CGM, DexcomFeatureG7CGM, DexcomFeatureAutoCal, DexcomFeatureDisableGraphSmoothing,
}

var dexcomFeatureTypeNames = map[DexcomFeatureType]string{
	DexcomFeatureG5CGM:                 "G5_CGM",
	DexcomFeatureG6CGM:                 "G6_CGM",
	DexcomFeatureG7CGM:                 "G7_CGM",
	DexcomFeatureAutoCal:               "AUTO_CAL",
	DexcomFeatureDisableGraphSmoothing: "DISABLE_GRAPH_SMOOTHING",
}

// DexcomFeatureTypeFromBitmask is the features whose bits mask sets, in
// declaration order; never nil.
func DexcomFeatureTypeFromBitmask(mask int64) []DexcomFeatureType {
	out := []DexcomFeatureType{}
	for _, f := range DexcomFeatureTypes {
		if mask&int64(f) != 0 {
			out = append(out, f)
		}
	}
	return out
}

func (f DexcomFeatureType) String() string {
	if name, ok := dexcomFeatureTypeNames[f]; ok {
		return name
	}
	return fmt.Sprintf("DexcomFeatureType(%d)", int32(f))
}

// AbbottFeatureType is an Abbott CGM feature, as its bit of the
// ABBOTT_FEATURES bitmask.
type AbbottFeatureType int32

// AbbottFeatureIsFSL2 is the only Abbott feature pumpX2 names.
const AbbottFeatureIsFSL2 AbbottFeatureType = 1

// AbbottFeatureTypes are the Abbott features in pumpX2's declaration order.
var AbbottFeatureTypes = []AbbottFeatureType{AbbottFeatureIsFSL2}

// AbbottFeatureTypeFromBitmask is the features whose bits mask sets, in
// declaration order; never nil.
func AbbottFeatureTypeFromBitmask(mask int64) []AbbottFeatureType {
	out := []AbbottFeatureType{}
	for _, f := range AbbottFeatureTypes {
		if mask&int64(f) != 0 {
			out = append(out, f)
		}
	}
	return out
}

func (f AbbottFeatureType) String() string {
	if f == AbbottFeatureIsFSL2 {
		return "IS_FSL2"
	}
	return fmt.Sprintf("AbbottFeatureType(%d)", int32(f))
}

// SecondaryFeatureType is one of the pump's other features, as its bit of
// the SECONDARY_FEATURES bitmask.
type SecondaryFeatureType int32

// The secondary features, by their bits.
const (
	SecondaryFeatureBaselineProfiles SecondaryFeatureType = 1 << iota
	SecondaryFeatureMultiCGM
	SecondaryFeatureBLETDU
)

// SecondaryFeatureTypes are the secondary features in pumpX2's declaration
// order.
var SecondaryFeatureTypes = []SecondaryFeatureType{SecondaryFeatureBaselineProfiles, SecondaryFeatureMultiCGM, SecondaryFeatureBLETDU}

var secondaryFeatureTypeNames = map[SecondaryFeatureType]string{
	SecondaryFeatureBaselineProfiles: "BASELINE_PROFILES",
	SecondaryFeatureMultiCGM:         "MULTI_CGM",
	SecondaryFeatureBLETDU:           "BLE_TDU",
}

// SecondaryFeatureTypeFromBitmask is the features whose bits mask sets, in
// declaration order; never nil.
func SecondaryFeatureTypeFromBitmask(mask int64) []SecondaryFeatureType {
	out := []SecondaryFeatureType{}
	for _, f := range SecondaryFeatureTypes {
		if mask&int64(f) != 0 {
			out = append(out, f)
		}
	}
	return out
}

func (f SecondaryFeatureType) String() string {
	if name, ok := secondaryFeatureTypeNames[f]; ok {
		return name
	}
	return fmt.Sprintf("SecondaryFeatureType(%d)", int32(f))
}
