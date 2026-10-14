package control

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// The quick bolus increments the pump supports: milliunits in units mode,
// milligrams in carbs mode.
var (
	SupportedQuickBolusIncrementUnits = []int32{500, 1000, 2000, 5000}
	SupportedQuickBolusIncrementCarbs = []int32{2000, 5000, 10000, 15000}
)

// SetQuickBolusSettingsRequest sets the quick bolus button. ModeRaw is a
// QuickBolusMode, IncrementUnits is in milliunits and IncrementCarbs in
// milligrams. The pump applies only the fields ChangedFieldsRaw, a
// ChangedField bitmask, names, but as the Tandem app does, every other
// field must carry the pump's current value:
// NewSetQuickBolusSettingsRequestForChange builds such a request.
type SetQuickBolusSettingsRequest struct {
	messages.Base
	ChangedFieldsRaw int32 `pumpx2:"changedFieldsRaw"`
	Enabled          bool  `pumpx2:"enabled"`
	IncrementCarbs   int32 `pumpx2:"incrementCarbs"`
	IncrementUnits   int32 `pumpx2:"incrementUnits"`
	ModeRaw          int32 `pumpx2:"modeRaw"`
}

var setQuickBolusSettingsRequestProps = messages.Props{OpCode: -46, Size: 7, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetQuickBolusSettingsResponse"}

func init() {
	messages.Register("SetQuickBolusSettingsRequest", &SetQuickBolusSettingsRequest{}, setQuickBolusSettingsRequestProps,
		messages.Constructor{
			Params: "boolean enabled, int modeRaw, int incrementUnits, int incrementCarbs, int changedFieldsRaw",
			Func:   NewSetQuickBolusSettingsRequest,
		},
		messages.Constructor{Params: "SetQuickBolusSettingsRequest.QuickBolusIncrement increment", Func: NewSetQuickBolusSettingsRequestForIncrement},
		messages.Constructor{
			Params: "boolean enabled, SetQuickBolusSettingsRequest.QuickBolusMode mode, SetQuickBolusSettingsRequest.QuickBolusIncrement increment",
			Func:   NewSetQuickBolusSettingsRequestForModeAndIncrement,
		},
		messages.Constructor{Params: "byte[] raw", Func: NewSetQuickBolusSettingsRequestFromCargo},
	)
}

// NewSetQuickBolusSettingsRequest builds the request from its fields.
func NewSetQuickBolusSettingsRequest(enabled bool, modeRaw, incrementUnits, incrementCarbs, changedFieldsRaw int32) *SetQuickBolusSettingsRequest {
	m := &SetQuickBolusSettingsRequest{}
	m.decode(helpers.Combine(
		[]byte{helpers.Bool(enabled), byte(modeRaw)},
		helpers.FirstTwoBytesLittleEndian(incrementUnits),
		helpers.FirstTwoBytesLittleEndian(incrementCarbs),
		[]byte{byte(changedFieldsRaw)},
	))
	return m
}

// NewSetQuickBolusSettingsRequestForIncrement is the Tandem app's captured
// write for the increment, which applies only the fields that capture
// flags. NewSetQuickBolusSettingsRequestForIncrementChange reaches the
// increment from any pump state.
func NewSetQuickBolusSettingsRequestForIncrement(increment QuickBolusIncrement) *SetQuickBolusSettingsRequest {
	return capturedQuickBolusSettings(increment.Enabled(), increment.Mode(), increment)
}

// NewSetQuickBolusSettingsRequestForModeAndIncrement is the captured write
// for the increment, under the given enabled flag and mode. It fails for
// enabled with QuickBolusIncrementDisabled, or disabled with any other.
func NewSetQuickBolusSettingsRequestForModeAndIncrement(enabled bool, mode QuickBolusMode, increment QuickBolusIncrement) (*SetQuickBolusSettingsRequest, error) {
	if enabled && increment == QuickBolusIncrementDisabled {
		return nil, messages.Invalid("SetQuickBolusSettingsRequest: cannot specify QuickBolusIncrement.DISABLED when enabled")
	}
	if !enabled && increment != QuickBolusIncrementDisabled {
		return nil, messages.Invalid("SetQuickBolusSettingsRequest: must specify QuickBolusIncrement.DISABLED when disabled")
	}
	return capturedQuickBolusSettings(enabled, mode, increment), nil
}

func capturedQuickBolusSettings(enabled bool, mode QuickBolusMode, increment QuickBolusIncrement) *SetQuickBolusSettingsRequest {
	m := &SetQuickBolusSettingsRequest{}
	m.decode(helpers.Combine([]byte{helpers.Bool(enabled), byte(mode.Raw())}, quickBolusIncrements[increment].magic))
	return m
}

// NewSetQuickBolusSettingsRequestFromCargo parses a request's payload.
func NewSetQuickBolusSettingsRequestFromCargo(raw []byte) (*SetQuickBolusSettingsRequest, error) {
	m := &SetQuickBolusSettingsRequest{}
	return m, m.Parse(raw)
}

// NewSetQuickBolusSettingsRequestForChange is the request that moves the
// pump from current to the given settings, or one that changes no field
// when the pump already matches, which need not be sent. increment is in
// milliunits in units mode and milligrams in carbs mode; mode and increment
// are ignored when disabling. It fails on an increment the pump does not
// support.
func NewSetQuickBolusSettingsRequestForChange(current *currentstatus.PumpGlobalsResponse, enabled bool, mode QuickBolusMode,
	increment int32) (*SetQuickBolusSettingsRequest, error) {
	units, carbs, modeRaw := current.QuickBolusIncrementUnits, current.QuickBolusIncrementCarbs, current.QuickBolusEntryType
	var changed []ChangedField
	if current.QuickBolusEnabled() != enabled {
		changed = append(changed, ChangedFieldEnabled)
	}
	if enabled {
		if modeRaw != mode.Raw() {
			changed = append(changed, ChangedFieldMode)
			modeRaw = mode.Raw()
		}
		if mode == QuickBolusModeUnits {
			if !containsInt32(SupportedQuickBolusIncrementUnits, increment) {
				return nil, messages.Invalid("SetQuickBolusSettingsRequest: unsupported quick bolus units increment: %d", increment)
			}
			if units != increment {
				changed = append(changed, ChangedFieldIncrementUnits)
				units = increment
			}
		} else {
			if !containsInt32(SupportedQuickBolusIncrementCarbs, increment) {
				return nil, messages.Invalid("SetQuickBolusSettingsRequest: unsupported quick bolus carbs increment: %d", increment)
			}
			if carbs != increment {
				changed = append(changed, ChangedFieldIncrementCarbs)
				carbs = increment
			}
		}
	}
	return NewSetQuickBolusSettingsRequest(enabled, modeRaw, units, carbs, ChangedFieldToBitmask(changed...)), nil
}

// NewSetQuickBolusSettingsRequestForIncrementChange is
// NewSetQuickBolusSettingsRequestForChange to the increment's settings.
func NewSetQuickBolusSettingsRequestForIncrementChange(current *currentstatus.PumpGlobalsResponse,
	increment QuickBolusIncrement) (*SetQuickBolusSettingsRequest, error) {
	return NewSetQuickBolusSettingsRequestForChange(current, increment.Enabled(), increment.Mode(), increment.Increment())
}

func containsInt32(values []int32, v int32) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}

// Parse implements messages.Message.
func (m *SetQuickBolusSettingsRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setQuickBolusSettingsRequestProps, raw)
	if len(raw) != setQuickBolusSettingsRequestProps.Size {
		return messages.InvalidLength("SetQuickBolusSettingsRequest", len(raw), setQuickBolusSettingsRequestProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *SetQuickBolusSettingsRequest) decode(raw []byte) {
	m.SetCargo(raw)
	m.Enabled = raw[0] == 1
	m.ModeRaw = int32(int8(raw[1]))
	m.IncrementUnits = helpers.ReadShort(raw, 2)
	m.IncrementCarbs = helpers.ReadShort(raw, 4)
	m.ChangedFieldsRaw = int32(raw[6])
}

// Increment is the increment the pump is left at if it applies the request,
// or nil for one it does not support.
func (m *SetQuickBolusSettingsRequest) Increment() *QuickBolusIncrement {
	return QuickBolusIncrementForSettings(m.Enabled, m.ModeRaw, m.IncrementUnits, m.IncrementCarbs)
}

// Mode is ModeRaw as a mode, or nil for one that is none.
func (m *SetQuickBolusSettingsRequest) Mode() *QuickBolusMode { return QuickBolusModeForRaw(m.ModeRaw) }

// ChangedFields are the fields ChangedFieldsRaw names.
func (m *SetQuickBolusSettingsRequest) ChangedFields() []ChangedField {
	return ChangedFieldFromBitmask(m.ChangedFieldsRaw)
}

// QuickBolusIncrementOf is the increment a pump's quick bolus settings are
// at, or nil for one it does not support. It is pumpX2's
// PumpGlobalsResponse.getQuickBolusIncrement.
func QuickBolusIncrementOf(g *currentstatus.PumpGlobalsResponse) *QuickBolusIncrement {
	return QuickBolusIncrementForSettings(g.QuickBolusEnabled(), g.QuickBolusEntryType, g.QuickBolusIncrementUnits, g.QuickBolusIncrementCarbs)
}

// QuickBolusModeOf is a pump's quick bolus mode, or nil for one that is
// none. It is pumpX2's PumpGlobalsResponse.getQuickBolusMode.
func QuickBolusModeOf(g *currentstatus.PumpGlobalsResponse) *QuickBolusMode {
	return QuickBolusModeForRaw(g.QuickBolusEntryType)
}

// QuickBolusIncrement is a quick bolus button setting: off, or on at an
// increment of insulin or carbs.
type QuickBolusIncrement int32

// The quick bolus increments, in pumpX2's declaration order.
const (
	QuickBolusIncrementDisabled QuickBolusIncrement = iota
	QuickBolusIncrementUnits0_5
	QuickBolusIncrementUnits1_0
	QuickBolusIncrementUnits2_0
	QuickBolusIncrementUnits5_0
	QuickBolusIncrementCarbs2G
	QuickBolusIncrementCarbs5G
	QuickBolusIncrementCarbs10G
	QuickBolusIncrementCarbs15G
)

var quickBolusIncrements = []struct {
	name      string
	enabled   bool
	mode      QuickBolusMode
	increment int32
	// A captured Tandem app write's last five bytes, not an encoding of the
	// increment: Disabled and Units0_5 share one.
	magic []byte
}{
	{"DISABLED", false, QuickBolusModeUnits, 0, []byte{0xf4, 0x01, 0xd0, 0x07, 1}},
	{"UNITS_0_5", true, QuickBolusModeUnits, 500, []byte{0xf4, 0x01, 0xd0, 0x07, 1}},
	{"UNITS_1_0", true, QuickBolusModeUnits, 1000, []byte{0xe8, 0x03, 0xd0, 0x07, 4}},
	{"UNITS_2_0", true, QuickBolusModeUnits, 2000, []byte{0xd0, 0x07, 0xd0, 0x07, 4}},
	{"UNITS_5_0", true, QuickBolusModeUnits, 5000, []byte{0x88, 0x13, 0xd0, 0x07, 4}},
	{"CARBS_2G", true, QuickBolusModeCarbs, 2000, []byte{0x88, 0x13, 0xd0, 0x07, 8}},
	{"CARBS_5G", true, QuickBolusModeCarbs, 5000, []byte{0x88, 0x13, 0x88, 0x13, 8}},
	{"CARBS_10G", true, QuickBolusModeCarbs, 10000, []byte{0x88, 0x13, 0x10, 0x27, 8}},
	{"CARBS_15G", true, QuickBolusModeCarbs, 15000, []byte{0x88, 0x13, 0x98, 0x3a, 8}},
}

// QuickBolusIncrementForSettings is the increment of the given settings,
// Disabled whenever disabled, or nil when enabled at an increment or mode
// the pump does not support.
func QuickBolusIncrementForSettings(enabled bool, modeRaw, incrementUnits, incrementCarbs int32) *QuickBolusIncrement {
	if !enabled {
		d := QuickBolusIncrementDisabled
		return &d
	}
	mode := QuickBolusModeForRaw(modeRaw)
	if mode == nil {
		return nil
	}
	increment := incrementCarbs
	if *mode == QuickBolusModeUnits {
		increment = incrementUnits
	}
	for i, q := range quickBolusIncrements {
		if q.enabled && q.mode == *mode && q.increment == increment {
			found := QuickBolusIncrement(i)
			return &found
		}
	}
	return nil
}

// Enabled reports whether the setting turns the button on.
func (i QuickBolusIncrement) Enabled() bool { return quickBolusIncrements[i].enabled }

// Mode is the setting's mode; units for Disabled.
func (i QuickBolusIncrement) Mode() QuickBolusMode { return quickBolusIncrements[i].mode }

// Increment is in milliunits in units mode and milligrams in carbs mode; 0
// for Disabled.
func (i QuickBolusIncrement) Increment() int32 { return quickBolusIncrements[i].increment }

func (i QuickBolusIncrement) String() string {
	if i >= 0 && int(i) < len(quickBolusIncrements) {
		return quickBolusIncrements[i].name
	}
	return fmt.Sprintf("QuickBolusIncrement(%d)", int32(i))
}

// ChangedField is one field a SetQuickBolusSettingsRequest applies, as a
// bit of ChangedFieldsRaw.
type ChangedField int32

// The quick bolus fields, by their bits.
const (
	ChangedFieldEnabled        ChangedField = 0x01
	ChangedFieldMode           ChangedField = 0x02
	ChangedFieldIncrementUnits ChangedField = 0x04
	ChangedFieldIncrementCarbs ChangedField = 0x08
)

// ChangedFields are the quick bolus fields in pumpX2's declaration order.
var ChangedFields = []ChangedField{ChangedFieldEnabled, ChangedFieldMode, ChangedFieldIncrementUnits, ChangedFieldIncrementCarbs}

// ChangedFieldFromBitmask is the fields whose bits mask sets, in
// declaration order; never nil.
func ChangedFieldFromBitmask(mask int32) []ChangedField {
	out := []ChangedField{}
	for _, f := range ChangedFields {
		if mask&int32(f) != 0 {
			out = append(out, f)
		}
	}
	return out
}

// ChangedFieldToBitmask is the bits of fields.
func ChangedFieldToBitmask(fields ...ChangedField) int32 {
	var mask int32
	for _, f := range fields {
		mask |= int32(f)
	}
	return mask
}

// Mask is the field's bit.
func (f ChangedField) Mask() int32 { return int32(f) }

func (f ChangedField) String() string {
	switch f {
	case ChangedFieldEnabled:
		return "ENABLED"
	case ChangedFieldMode:
		return "MODE"
	case ChangedFieldIncrementUnits:
		return "INCREMENT_UNITS"
	case ChangedFieldIncrementCarbs:
		return "INCREMENT_CARBS"
	}
	return fmt.Sprintf("ChangedField(%d)", int32(f))
}

// QuickBolusMode is whether the quick bolus button counts insulin or carbs.
type QuickBolusMode int32

// The quick bolus modes, by their wire values.
const (
	QuickBolusModeUnits QuickBolusMode = 0
	QuickBolusModeCarbs QuickBolusMode = 1
)

// QuickBolusModeForRaw is the mode with that wire value, or nil for one
// that is none.
func QuickBolusModeForRaw(raw int32) *QuickBolusMode {
	if raw != int32(QuickBolusModeUnits) && raw != int32(QuickBolusModeCarbs) {
		return nil
	}
	m := QuickBolusMode(raw)
	return &m
}

// Raw is the mode's wire value.
func (m QuickBolusMode) Raw() int32 { return int32(m) }

func (m QuickBolusMode) String() string {
	switch m {
	case QuickBolusModeUnits:
		return "UNITS"
	case QuickBolusModeCarbs:
		return "CARBS"
	}
	return fmt.Sprintf("QuickBolusMode(%d)", int32(m))
}
