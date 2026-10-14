package currentstatus

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// HomeScreenMirrorResponse is the icons the pump's home screen shows. Each
// icon is given by its wire ID and as its enum, which is nil for an ID
// pumpX2 does not know.
type HomeScreenMirrorResponse struct {
	messages.Base
	CgmTrendIconID           int32               `pumpx2:"cgmTrendIconId"`
	CgmAlertIconID           int32               `pumpx2:"cgmAlertIconId"`
	StatusIcon0ID            int32               `pumpx2:"statusIcon0Id"`
	StatusIcon1ID            int32               `pumpx2:"statusIcon1Id"`
	BolusStatusIconID        int32               `pumpx2:"bolusStatusIconId"`
	BasalStatusIconID        int32               `pumpx2:"basalStatusIconId"`
	ApControlStateIconID     int32               `pumpx2:"apControlStateIconId"`
	CgmTrendIcon             *CGMTrendIcon       `pumpx2:"cgmTrendIcon"`
	CgmAlertIcon             *CGMAlertIcon       `pumpx2:"cgmAlertIcon"`
	StatusIcon0              *StatusIcon0        `pumpx2:"statusIcon0"`
	StatusIcon1              *StatusIcon1        `pumpx2:"statusIcon1"`
	BolusStatusIcon          *BolusStatusIcon    `pumpx2:"bolusStatusIcon"`
	BasalStatusIcon          *BasalStatusIcon    `pumpx2:"basalStatusIcon"`
	ApControlStateIcon       *ApControlStateIcon `pumpx2:"apControlStateIcon"`
	RemainingInsulinPlusIcon bool                `pumpx2:"remainingInsulinPlusIcon"`
	CgmDisplayData           bool                `pumpx2:"cgmDisplayData"`
}

var homeScreenMirrorResponseProps = messages.Props{OpCode: 57, Size: 9, Type: messages.Response, Request: "HomeScreenMirrorRequest"}

func init() {
	messages.Register("HomeScreenMirrorResponse", &HomeScreenMirrorResponse{}, homeScreenMirrorResponseProps,
		messages.Constructor{
			Params: "int cgmTrendIconId, int cgmAlertIconId, int statusIcon0Id, int statusIcon1Id, int bolusStatusIconId, int basalStatusIconId, " +
				"int apControlStateIconId, boolean remainingInsulinPlusIcon, boolean cgmDisplayData",
			Func: NewHomeScreenMirrorResponse,
		},
	)
}

// NewHomeScreenMirrorResponse builds the response from the icons' wire IDs,
// each one byte, and reads it back as Parse does.
func NewHomeScreenMirrorResponse(cgmTrendIconID, cgmAlertIconID, statusIcon0ID, statusIcon1ID, bolusStatusIconID, basalStatusIconID, apControlStateIconID int32,
	remainingInsulinPlusIcon, cgmDisplayData bool) *HomeScreenMirrorResponse {
	m := &HomeScreenMirrorResponse{}
	m.decode([]byte{byte(cgmTrendIconID), byte(cgmAlertIconID), byte(statusIcon0ID), byte(statusIcon1ID), byte(bolusStatusIconID),
		byte(basalStatusIconID), byte(apControlStateIconID), helpers.Bool(remainingInsulinPlusIcon), helpers.Bool(cgmDisplayData)})
	return m
}

// Parse implements messages.Message.
func (m *HomeScreenMirrorResponse) Parse(raw []byte) error {
	if len(raw) != homeScreenMirrorResponseProps.Size {
		return messages.InvalidLength("HomeScreenMirrorResponse", len(raw), homeScreenMirrorResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *HomeScreenMirrorResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.CgmTrendIconID = int32(raw[0])
	m.CgmAlertIconID = int32(raw[1])
	m.StatusIcon0ID = int32(raw[2])
	m.StatusIcon1ID = int32(raw[3])
	m.BolusStatusIconID = int32(raw[4])
	m.BasalStatusIconID = int32(raw[5])
	m.ApControlStateIconID = int32(raw[6])
	m.RemainingInsulinPlusIcon = raw[7] != 0
	m.CgmDisplayData = raw[8] != 0
	m.CgmTrendIcon = CGMTrendIconFromID(m.CgmTrendIconID)
	m.CgmAlertIcon = CGMAlertIconFromID(m.CgmAlertIconID)
	m.StatusIcon0 = StatusIcon0FromID(m.StatusIcon0ID)
	m.StatusIcon1 = StatusIcon1FromID(m.StatusIcon1ID)
	m.BolusStatusIcon = BolusStatusIconFromID(m.BolusStatusIconID)
	m.BasalStatusIcon = BasalStatusIconFromID(m.BasalStatusIconID)
	m.ApControlStateIcon = ApControlStateIconFromID(m.ApControlStateIconID)
}

// iconName is the name of the icon with wire ID id in names, or typ(id).
func iconName(names map[int32]string, typ string, id int32) string {
	if name, ok := names[id]; ok {
		return name
	}
	return fmt.Sprintf("%s(%d)", typ, id)
}

// CGMTrendIcon is the CGM trend arrow.
type CGMTrendIcon int32

// The CGM trend arrows, by their wire IDs.
const (
	CGMTrendIconNoArrow         CGMTrendIcon = 0
	CGMTrendIconArrowDoubleUp   CGMTrendIcon = 1
	CGMTrendIconArrowUp         CGMTrendIcon = 2
	CGMTrendIconArrowUpRight    CGMTrendIcon = 3
	CGMTrendIconArrowFlat       CGMTrendIcon = 4
	CGMTrendIconArrowDownRight  CGMTrendIcon = 5
	CGMTrendIconArrowDown       CGMTrendIcon = 6
	CGMTrendIconArrowDoubleDown CGMTrendIcon = 7
)

var (
	cgmTrendIconNames = []string{
		"NO_ARROW", "ARROW_DOUBLE_UP", "ARROW_UP", "ARROW_UP_RIGHT", "ARROW_FLAT", "ARROW_DOWN_RIGHT", "ARROW_DOWN", "ARROW_DOUBLE_DOWN",
	}
	cgmTrendIconArrows = []string{"", "▲▲", "▲", "⬈", "▶", "⬊", "▼", "▼▼"}
)

// CGMTrendIconFromID is the arrow with wire ID id, or nil for an ID that is
// none.
func CGMTrendIconFromID(id int32) *CGMTrendIcon {
	if id < 0 || int(id) >= len(cgmTrendIconNames) {
		return nil
	}
	i := CGMTrendIcon(id)
	return &i
}

// ID is the arrow's wire ID.
func (i CGMTrendIcon) ID() int32 { return int32(i) }

// Arrow is the arrow as text, empty for NO_ARROW.
func (i CGMTrendIcon) Arrow() string {
	if i >= 0 && int(i) < len(cgmTrendIconArrows) {
		return cgmTrendIconArrows[i]
	}
	return ""
}

func (i CGMTrendIcon) String() string {
	if i >= 0 && int(i) < len(cgmTrendIconNames) {
		return cgmTrendIconNames[i]
	}
	return fmt.Sprintf("CGMTrendIcon(%d)", int32(i))
}

// CGMAlertIcon is the CGM's alert or session icon.
type CGMAlertIcon int32

// The CGM alert icons, by their wire IDs.
const (
	CGMAlertIconStartup1           CGMAlertIcon = 0
	CGMAlertIconStartup2           CGMAlertIcon = 1
	CGMAlertIconStartup3           CGMAlertIcon = 2
	CGMAlertIconStartup4           CGMAlertIcon = 3
	CGMAlertIconCalibrate          CGMAlertIcon = 4
	CGMAlertIconStartupCalibrate   CGMAlertIcon = 5
	CGMAlertIconCheckmarkBloodDrop CGMAlertIcon = 6
	CGMAlertIconErrorHighWedge     CGMAlertIcon = 7
	CGMAlertIconErrorLowWedge      CGMAlertIcon = 8
	CGMAlertIconReplaceSensor      CGMAlertIcon = 11
	CGMAlertIconReplaceTransmitter CGMAlertIcon = 12
	CGMAlertIconOutOfRange         CGMAlertIcon = 13
	CGMAlertIconFailedSensor       CGMAlertIcon = 14
	CGMAlertIconTripleDashes       CGMAlertIcon = 15
	CGMAlertIconLow                CGMAlertIcon = 16
	CGMAlertIconHigh               CGMAlertIcon = 17
	CGMAlertIconNoError            CGMAlertIcon = 255
)

var cgmAlertIconNames = map[int32]string{
	0: "STARTUP_1", 1: "STARTUP_2", 2: "STARTUP_3", 3: "STARTUP_4", 4: "CALIBRATE", 5: "STARTUP_CALIBRATE", 6: "CHECKMARK_BLOOD_DROP",
	7: "ERROR_HIGH_WEDGE", 8: "ERROR_LOW_WEDGE", 11: "REPLACE_SENSOR", 12: "REPLACE_TRANSMITTER", 13: "OUT_OF_RANGE",
	14: "FAILED_SENSOR", 15: "TRIPLE_DASHES", 16: "LOW", 17: "HIGH", 255: "NO_ERROR",
}

// CGMAlertIconFromID is the icon with wire ID id, or nil for an ID that is
// none.
func CGMAlertIconFromID(id int32) *CGMAlertIcon {
	if _, ok := cgmAlertIconNames[id]; !ok {
		return nil
	}
	i := CGMAlertIcon(id)
	return &i
}

// ID is the icon's wire ID.
func (i CGMAlertIcon) ID() int32 { return int32(i) }

func (i CGMAlertIcon) String() string { return iconName(cgmAlertIconNames, "CGMAlertIcon", int32(i)) }

// StatusIcon0 is the first status icon.
type StatusIcon0 int32

// The first status icons, by their wire IDs.
const (
	StatusIcon0AAM               StatusIcon0 = 0
	StatusIcon0CGMTransmitter    StatusIcon0 = 1
	StatusIcon0CGMTransmitterOOR StatusIcon0 = 4
	StatusIcon0HideIcon          StatusIcon0 = 200
)

var statusIcon0Names = map[int32]string{0: "AAM", 1: "CGM_TRANSMITTER", 4: "CGM_TRANSMITTER_OOR", 200: "HIDE_ICON"}

// StatusIcon0FromID is the icon with wire ID id, or nil for an ID that is
// none.
func StatusIcon0FromID(id int32) *StatusIcon0 {
	if _, ok := statusIcon0Names[id]; !ok {
		return nil
	}
	i := StatusIcon0(id)
	return &i
}

// ID is the icon's wire ID.
func (i StatusIcon0) ID() int32 { return int32(i) }

func (i StatusIcon0) String() string { return iconName(statusIcon0Names, "StatusIcon0", int32(i)) }

// StatusIcon1 is the second status icon.
type StatusIcon1 int32

// The second status icons, by their wire IDs.
const (
	StatusIcon1AAM       StatusIcon1 = 0
	StatusIcon1BloodDrop StatusIcon1 = 1
	StatusIcon1HideIcon  StatusIcon1 = 200
)

var statusIcon1Names = map[int32]string{0: "AAM", 1: "BLOOD_DROP", 200: "HIDE_ICON"}

// StatusIcon1FromID is the icon with wire ID id, or nil for an ID that is
// none.
func StatusIcon1FromID(id int32) *StatusIcon1 {
	if _, ok := statusIcon1Names[id]; !ok {
		return nil
	}
	i := StatusIcon1(id)
	return &i
}

// ID is the icon's wire ID.
func (i StatusIcon1) ID() int32 { return int32(i) }

func (i StatusIcon1) String() string { return iconName(statusIcon1Names, "StatusIcon1", int32(i)) }

// BolusStatusIcon is the bolus icon.
type BolusStatusIcon int32

// The bolus icons, by their wire IDs.
const (
	BolusStatusIconBolus                             BolusStatusIcon = 0
	BolusStatusIconAutoBolus                         BolusStatusIcon = 1
	BolusStatusIconBlackLockSecurityPin              BolusStatusIcon = 2
	BolusStatusIconGreyLockSecurityPinCGMGraphActive BolusStatusIcon = 3
	BolusStatusIconHideIcon                          BolusStatusIcon = 200
)

var bolusStatusIconNames = map[int32]string{
	0: "BOLUS", 1: "AUTO_BOLUS", 2: "BLACK_LOCK_SECURITY_PIN", 3: "GREY_LOCK_SECURITY_PIN_CGM_GRAPH_ACTIVE", 200: "HIDE_ICON",
}

// BolusStatusIconFromID is the icon with wire ID id, or nil for an ID that
// is none.
func BolusStatusIconFromID(id int32) *BolusStatusIcon {
	if _, ok := bolusStatusIconNames[id]; !ok {
		return nil
	}
	i := BolusStatusIcon(id)
	return &i
}

// ID is the icon's wire ID.
func (i BolusStatusIcon) ID() int32 { return int32(i) }

func (i BolusStatusIcon) String() string {
	return iconName(bolusStatusIconNames, "BolusStatusIcon", int32(i))
}

// BasalStatusIcon is the basal icon.
type BasalStatusIcon int32

// The basal icons, by their wire IDs.
const (
	BasalStatusIconBasal              BasalStatusIcon = 0
	BasalStatusIconZeroBasal          BasalStatusIcon = 1
	BasalStatusIconTempRate           BasalStatusIcon = 2
	BasalStatusIconZeroTempRate       BasalStatusIcon = 3
	BasalStatusIconSuspend            BasalStatusIcon = 4
	BasalStatusIconHypoSuspendBasalIQ BasalStatusIcon = 5
	BasalStatusIconIncreaseBasal      BasalStatusIcon = 6
	BasalStatusIconAttenuatedBasal    BasalStatusIcon = 7
	BasalStatusIconHide               BasalStatusIcon = 200
)

var basalStatusIconNames = map[int32]string{
	0: "BASAL", 1: "ZERO_BASAL", 2: "TEMP_RATE", 3: "ZERO_TEMP_RATE", 4: "SUSPEND", 5: "HYPO_SUSPEND_BASAL_IQ",
	6: "INCREASE_BASAL", 7: "ATTENUATED_BASAL", 200: "HIDE",
}

// BasalStatusIconFromID is the icon with wire ID id, or nil for an ID that
// is none.
func BasalStatusIconFromID(id int32) *BasalStatusIcon {
	if _, ok := basalStatusIconNames[id]; !ok {
		return nil
	}
	i := BasalStatusIcon(id)
	return &i
}

// ID is the icon's wire ID.
func (i BasalStatusIcon) ID() int32 { return int32(i) }

func (i BasalStatusIcon) String() string {
	return iconName(basalStatusIconNames, "BasalStatusIcon", int32(i))
}

// ApControlStateIcon is the Basal-IQ or Control-IQ state icon.
type ApControlStateIcon int32

// The Basal-IQ and Control-IQ state icons, by their wire IDs.
const (
	ApControlStateIconStateGray                          ApControlStateIcon = 0
	ApControlStateIconStateGrayRedBiqCiqBasalSuspended   ApControlStateIcon = 1
	ApControlStateIconStateGrayBlueCiqIncreaseBasal      ApControlStateIcon = 2
	ApControlStateIconStateGrayOrangeCiqAttenuationBasal ApControlStateIcon = 3
	ApControlStateIconHideIcon                           ApControlStateIcon = 200
)

var apControlStateIconNames = map[int32]string{
	0: "STATE_GRAY", 1: "STATE_GRAY_RED_BIQ_CIQ_BASAL_SUSPENDED", 2: "STATE_GRAY_BLUE_CIQ_INCREASE_BASAL",
	3: "STATE_GRAY_ORANGE_CIQ_ATTENUATION_BASAL", 200: "HIDE_ICON",
}

// ApControlStateIconFromID is the icon with wire ID id, or nil for an ID
// that is none.
func ApControlStateIconFromID(id int32) *ApControlStateIcon {
	if _, ok := apControlStateIconNames[id]; !ok {
		return nil
	}
	i := ApControlStateIcon(id)
	return &i
}

// ID is the icon's wire ID.
func (i ApControlStateIcon) ID() int32 { return int32(i) }

func (i ApControlStateIcon) String() string {
	return iconName(apControlStateIconNames, "ApControlStateIcon", int32(i))
}
