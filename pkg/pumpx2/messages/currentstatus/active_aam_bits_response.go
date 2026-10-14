package currentstatus

import (
	"fmt"
	"math/big"
	"math/bits"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ActiveAamBitsResponse is a Mobi's active alerts, alarms or malfunctions
// (AAMs) of one AamType: 64-bit masks of those not yet acknowledged and of
// those active. For malfunctions each bit is a MalfunctionType.
type ActiveAamBitsResponse struct {
	messages.Base
	UnacknowledgedBitmask      *big.Int          `pumpx2:"unacknowledgedBitmask"`
	ActiveBitmask              *big.Int          `pumpx2:"activeBitmask"`
	AamType                    AamType           `pumpx2:"aamType"`
	UnacknowledgedMalfunctions []MalfunctionType `pumpx2:"unacknowledgedMalfunctions"`
	ActiveMalfunctions         []MalfunctionType `pumpx2:"activeMalfunctions"`
}

var activeAamBitsResponseProps = messages.Props{OpCode: -109, Size: 17, Type: messages.Response, MinAPI: "MOBI_API_V3_5", Request: "ActiveAamBitsRequest"}

func init() {
	messages.Register("ActiveAamBitsResponse", &ActiveAamBitsResponse{}, activeAamBitsResponseProps,
		messages.Constructor{Func: NewEmptyActiveAamBitsResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewActiveAamBitsResponseFromCargo},
		messages.Constructor{Params: "BigInteger unacknowledgedBitmask, BigInteger activeBitmask, ActiveAamBitsResponse.AamType aamType", Func: NewActiveAamBitsResponse},
	)
}

// NewEmptyActiveAamBitsResponse is the response with nothing active, of
// type Unknown, which pumpX2 parses into.
func NewEmptyActiveAamBitsResponse() *ActiveAamBitsResponse {
	m := &ActiveAamBitsResponse{
		UnacknowledgedBitmask:      big.NewInt(0),
		ActiveBitmask:              big.NewInt(0),
		AamType:                    AamTypeUnknown,
		UnacknowledgedMalfunctions: []MalfunctionType{},
		ActiveMalfunctions:         []MalfunctionType{},
	}
	m.SetCargo([]byte{})
	return m
}

// NewActiveAamBitsResponse is the response with the low 64 bits of each
// mask.
func NewActiveAamBitsResponse(unacknowledgedBitmask, activeBitmask *big.Int, aamType AamType) *ActiveAamBitsResponse {
	m := &ActiveAamBitsResponse{}
	m.decode(helpers.Combine(
		helpers.ToUint64(helpers.LongValue(unacknowledgedBitmask)),
		helpers.ToUint64(helpers.LongValue(activeBitmask)),
		[]byte{byte(aamType)},
	))
	return m
}

// NewActiveAamBitsResponseFromCargo parses a response's payload.
func NewActiveAamBitsResponseFromCargo(raw []byte) (*ActiveAamBitsResponse, error) {
	m := &ActiveAamBitsResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *ActiveAamBitsResponse) Parse(raw []byte) error {
	if len(raw) != activeAamBitsResponseProps.Size {
		return messages.InvalidLength("ActiveAamBitsResponse", len(raw), activeAamBitsResponseProps.Size)
	}
	m.decode(raw)
	return nil
}

func (m *ActiveAamBitsResponse) decode(raw []byte) {
	m.SetCargo(raw)
	m.UnacknowledgedBitmask = helpers.ReadUint64(raw, 0)
	m.ActiveBitmask = helpers.ReadUint64(raw, 8)
	m.AamType = AamTypeFromID(int32(raw[16]))
	if m.AamType == AamTypeMalfunction {
		m.UnacknowledgedMalfunctions = MalfunctionTypeFromBitmask(m.UnacknowledgedBitmask)
		m.ActiveMalfunctions = MalfunctionTypeFromBitmask(m.ActiveBitmask)
	} else {
		m.UnacknowledgedMalfunctions = []MalfunctionType{}
		m.ActiveMalfunctions = []MalfunctionType{}
	}
}

// HasActiveBit reports whether bit, 0 to 63, is set in the active mask.
func (m *ActiveAamBitsResponse) HasActiveBit(bit int32) bool {
	return bit >= 0 && bit < 64 && m.ActiveBitmask != nil && m.ActiveBitmask.Bit(int(bit)) != 0
}

// HasUnacknowledgedBit reports whether bit, 0 to 63, is set in the
// unacknowledged mask.
func (m *ActiveAamBitsResponse) HasUnacknowledgedBit(bit int32) bool {
	return bit >= 0 && bit < 64 && m.UnacknowledgedBitmask != nil && m.UnacknowledgedBitmask.Bit(int(bit)) != 0
}

// Size is how many bits the active mask sets.
func (m *ActiveAamBitsResponse) Size() int {
	if m.ActiveBitmask == nil {
		return 0
	}
	n := 0
	for _, w := range m.ActiveBitmask.Bits() {
		n += bits.OnesCount(uint(w))
	}
	return n
}

// NotificationIDs is empty, as pumpX2 leaves it.
func (m *ActiveAamBitsResponse) NotificationIDs() []int32 { return []int32{} }

// AamType is the kind of notification an ActiveAamBitsResponse reports.
type AamType int32

// The AAM types, by their wire IDs.
const (
	AamTypeReminder    AamType = 0
	AamTypeAlert       AamType = 1
	AamTypeAlarm       AamType = 2
	AamTypeCGMAlert    AamType = 3
	AamTypeMalfunction AamType = 4
	AamTypeNone        AamType = 5
	AamTypeUnknown     AamType = -1
)

var aamTypeNames = []string{"REMINDER", "ALERT", "ALARM", "CGM_ALERT", "MALFUNCTION", "NONE"}

// AamTypeFromID is the type with wire ID id, or Unknown.
func AamTypeFromID(id int32) AamType {
	if id >= 0 && int(id) < len(aamTypeNames) {
		return AamType(id)
	}
	return AamTypeUnknown
}

func (t AamType) String() string {
	switch {
	case t == AamTypeUnknown:
		return "UNKNOWN"
	case t >= 0 && int(t) < len(aamTypeNames):
		return aamTypeNames[t]
	}
	return fmt.Sprintf("AamType(%d)", int32(t))
}
