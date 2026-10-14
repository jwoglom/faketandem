package currentstatus

import (
	"fmt"
	"math/big"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// ReminderStatusResponse is the reminders the pump is showing, as a 64-bit
// bitmask of ReminderType IDs.
type ReminderStatusResponse struct {
	messages.Base
	// IntMap is the bitmask; pumpX2 leaves it out of toString.
	IntMap *big.Int
}

var reminderStatusResponseProps = messages.Props{OpCode: 73, Size: 8, Type: messages.Response, Request: "ReminderStatusRequest"}

func init() {
	messages.Register("ReminderStatusResponse", &ReminderStatusResponse{}, reminderStatusResponseProps,
		messages.Constructor{Params: "BigInteger reminderBitmask", Func: NewReminderStatusResponse},
	)
}

// NewReminderStatusResponse builds the response from its bitmask, whose low
// 64 bits it sends. It fails on a nil bitmask.
func NewReminderStatusResponse(reminderBitmask *big.Int) (*ReminderStatusResponse, error) {
	if reminderBitmask == nil {
		return nil, messages.Invalid("ReminderStatusResponse: no reminderBitmask")
	}
	m := &ReminderStatusResponse{IntMap: reminderBitmask}
	m.SetCargo(helpers.ToUint64(helpers.LongValue(reminderBitmask)))
	return m, nil
}

// Parse implements messages.Message.
func (m *ReminderStatusResponse) Parse(raw []byte) error {
	if len(raw) != reminderStatusResponseProps.Size {
		return messages.InvalidLength("ReminderStatusResponse", len(raw), reminderStatusResponseProps.Size)
	}
	m.SetCargo(raw)
	m.IntMap = helpers.ReadUint64(raw, 0)
	return nil
}

// BitMap is the bitmask's low 64 bits.
func (m *ReminderStatusResponse) BitMap() int64 { return helpers.LongValue(m.IntMap) }

// Reminders are the reminders the bitmask sets, in ID order.
func (m *ReminderStatusResponse) Reminders() []ReminderType { return ReminderTypeFromBitmask(m.IntMap) }

// Size is how many reminders the bitmask sets.
func (m *ReminderStatusResponse) Size() int { return len(m.Reminders()) }

// NotificationIDs are the IDs of the known reminders the bitmask sets, in
// ascending order.
func (m *ReminderStatusResponse) NotificationIDs() []int32 {
	ids := []int32{}
	for _, r := range m.Reminders() {
		if r.IsKnown() {
			ids = append(ids, r.ID())
		}
	}
	return ids
}

// ReminderType is a reminder, by its bit of ReminderStatusResponse's
// bitmask. pumpX2 names bits 0 to 8; it calls the rest, up to 63,
// DEFAULT_REMINDER_<bit>.
type ReminderType int32

// The reminders pumpX2 names, by their IDs.
const (
	ReminderLowBG ReminderType = iota
	ReminderHighBG
	ReminderSiteChange
	ReminderMissedMeal
	ReminderMissedMeal1
	ReminderMissedMeal2
	ReminderMissedMeal3
	ReminderAfterBolusBG
	ReminderAdditionalBolus
)

const reminderTypeCount = 64

var reminderTypeNames = []string{
	"LOW_BG_REMINDER", "HIGH_BG_REMINDER", "SITE_CHANGE_REMINDER", "MISSED_MEAL_REMINDER", "MISSED_MEAL_REMINDER1", "MISSED_MEAL_REMINDER2",
	"MISSED_MEAL_REMINDER3", "AFTER_BOLUS_BG_REMINDER", "ADDITIONAL_BOLUS_REMINDER",
}

// ReminderTypeFromBitmask is the reminders whose bits mask sets, in ID
// order; never nil. A negative mask is two's complement, as a BigInteger's
// is.
func ReminderTypeFromBitmask(mask *big.Int) []ReminderType {
	out := []ReminderType{}
	for id := 0; id < reminderTypeCount; id++ {
		if mask.Bit(id) != 0 {
			out = append(out, ReminderType(id))
		}
	}
	return out
}

// ReminderTypeToBitmask is the bitmask with the reminders' bits set.
func ReminderTypeToBitmask(reminders ...ReminderType) *big.Int {
	mask := new(big.Int)
	for _, r := range reminders {
		mask.SetBit(mask, int(r), 1)
	}
	return mask
}

// ID is the reminder's bit.
func (r ReminderType) ID() int32 { return int32(r) }

// IsKnown reports whether pumpX2 names the reminder.
func (r ReminderType) IsKnown() bool { return r >= 0 && int(r) < len(reminderTypeNames) }

func (r ReminderType) String() string {
	switch {
	case r.IsKnown():
		return reminderTypeNames[r]
	case r >= 0 && r < reminderTypeCount:
		return fmt.Sprintf("DEFAULT_REMINDER_%d", int32(r))
	}
	return fmt.Sprintf("ReminderType(%d)", int32(r))
}
