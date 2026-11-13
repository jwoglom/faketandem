package state

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
)

// NotificationCategory is which of the pump's notification bitmasks a standing
// Alert sets: the five the pump reports, each through its own status message
// (AlertStatus, AlarmStatus, MalfunctionStatus, ReminderStatus, CGMAlertStatus).
type NotificationCategory int

// The categories. NotificationNone is an alert this emulator tracks that sets
// no bit (AlertCartridgeExpired).
const (
	NotificationNone NotificationCategory = iota
	NotificationReminder
	NotificationAlert
	NotificationAlarm
	NotificationCGMAlert
	NotificationMalfunction
)

// NotificationCategories are the categories that set a bit, highest priority
// first: the order HighestAam picks from.
var NotificationCategories = []NotificationCategory{
	NotificationMalfunction, NotificationAlarm, NotificationAlert, NotificationCGMAlert, NotificationReminder,
}

func (c NotificationCategory) String() string {
	switch c {
	case NotificationReminder:
		return "reminder"
	case NotificationAlert:
		return "alert"
	case NotificationAlarm:
		return "alarm"
	case NotificationCGMAlert:
		return "cgm_alert"
	case NotificationMalfunction:
		return "malfunction"
	default:
		return "none"
	}
}

// AamType is the category's id in ActiveAamBits (pumpX2 AamType); the first four are also
// DismissNotificationRequest's NotificationType ids. -1 for NotificationNone.
func (c NotificationCategory) AamType() int {
	switch c {
	case NotificationReminder:
		return 0
	case NotificationAlert:
		return 1
	case NotificationAlarm:
		return 2
	case NotificationCGMAlert:
		return 3
	case NotificationMalfunction:
		return 4
	default:
		return -1
	}
}

// NotificationCategoryFromAamType is the category with that AamType id.
func NotificationCategoryFromAamType(id int) (NotificationCategory, bool) {
	for _, c := range NotificationCategories {
		if c.AamType() == id {
			return c, true
		}
	}
	return NotificationNone, false
}

// ParseNotificationCategory reads a category name as the HTTP API spells it.
func ParseNotificationCategory(name string) (NotificationCategory, bool) {
	switch strings.ToLower(strings.ReplaceAll(name, "-", "_")) {
	case "reminder":
		return NotificationReminder, true
	case "alert":
		return NotificationAlert, true
	case "alarm":
		return NotificationAlarm, true
	case "cgm_alert", "cgmalert":
		return NotificationCGMAlert, true
	case "malfunction":
		return NotificationMalfunction, true
	}
	return NotificationNone, false
}

// NotificationName is pumpX2's (and TandemKit's) enum name for bit in category c.
func NotificationName(c NotificationCategory, bit uint) string {
	switch c {
	case NotificationAlert:
		return currentstatus.AlertResponseType(bit).String()
	case NotificationAlarm:
		return currentstatus.AlarmResponseType(bit).String()
	case NotificationMalfunction:
		return currentstatus.MalfunctionType(bit).String()
	case NotificationReminder:
		return currentstatus.ReminderType(bit).String()
	case NotificationCGMAlert:
		return currentstatus.CGMAlert(bit).String()
	}
	return ""
}

// notificationSuffixes are the endings a name may leave off: "LOW_INSULIN" is
// LOW_INSULIN_ALERT, "RESUME_PUMP" RESUME_PUMP_ALARM.
var notificationSuffixes = map[NotificationCategory]string{
	NotificationAlert:    "_ALERT",
	NotificationAlarm:    "_ALARM",
	NotificationReminder: "_REMINDER",
	NotificationCGMAlert: "_CGM_ALERT",
}

// ParseNotificationBit resolves a notification of category c given as its enum
// name (case-insensitive, its category suffix optional) or its bit number.
func ParseNotificationBit(c NotificationCategory, name string) (uint, error) {
	name = strings.TrimSpace(name)
	if n, err := strconv.Atoi(name); err == nil {
		if n < 0 || n > 63 {
			return 0, fmt.Errorf("bit %d is outside 0-63", n)
		}
		return uint(n), nil
	}
	want := strings.ToUpper(name)
	suffix := notificationSuffixes[c]
	for bit := uint(0); bit < 64; bit++ {
		known := NotificationName(c, bit)
		if known == want || (suffix != "" && known == want+suffix) {
			return bit, nil
		}
	}
	return 0, fmt.Errorf("no %s named %q", c, name)
}

// defaultNotifications are the bits of the alerts this emulator raises itself.
var defaultNotifications = map[AlertType]struct {
	category NotificationCategory
	bit      uint
}{
	AlertLowReservoir: {NotificationAlert, uint(currentstatus.AlertResponseTypeLowInsulinAlert)},
	AlertLowBattery:   {NotificationAlert, uint(currentstatus.AlertResponseTypeLowPowerAlert)},
	AlertOcclusion:    {NotificationAlarm, uint(currentstatus.AlarmResponseTypeOcclusionAlarm)},
	// Modeled, not confirmed: the alarm a pump stopped by an alarm asks to be resumed with.
	AlertBasalSuspended: {NotificationAlarm, uint(currentstatus.AlarmResponseTypeResumePumpAlarm)},
}

// resolveNotification fills in the category and bit of an alert raised by type
// alone.
func resolveNotification(alert Alert) Alert {
	if alert.Category != NotificationNone {
		return alert
	}
	if def, ok := defaultNotifications[alert.Type]; ok {
		alert.Category = def.category
		alert.Bit = def.bit
	}
	return alert
}

// Name is pumpX2's enum name for the alert's bit, or its emulator type name
// when it sets none.
func (a Alert) Name() string {
	if a.Category == NotificationNone {
		return a.Type.String()
	}
	return NotificationName(a.Category, a.Bit)
}

// TypeName is the alert's emulator type, or its pumpX2 name for one raised by
// name.
func (a Alert) TypeName() string {
	if a.Type == AlertNotification {
		return a.Name()
	}
	return a.Type.String()
}

// allocateAlertIDLocked returns the next alert id; the caller holds ps.mutex.
func (ps *PumpState) allocateAlertIDLocked() uint32 {
	for _, a := range ps.ActiveAlerts {
		if a.ID > ps.lastAlertID {
			ps.lastAlertID = a.ID
		}
	}
	ps.lastAlertID++
	return ps.lastAlertID
}

// standingLocked reports whether bit stands in category c; the caller holds ps.mutex.
func (ps *PumpState) standingLocked(c NotificationCategory, bit uint) bool {
	for _, a := range ps.ActiveAlerts {
		if a.Category == c && a.Bit == bit {
			return true
		}
	}
	return false
}

// RaiseNotification puts a notification on the pump: the alert stands (and
// sets its bit) until dismissed or cleared. It fills in the id, the time (the
// pump's clock when zero) and, for an alert raised by type alone, its category
// and bit. It writes the activation record and returns the alert, or false
// when that bit already stands.
func (ps *PumpState) RaiseNotification(alert Alert, reason string) (Alert, bool) {
	if alert.Timestamp.IsZero() {
		alert.Timestamp = ps.Now()
	}
	ps.mutex.Lock()
	alert, ok := ps.raiseNotificationLocked(alert)
	ps.mutex.Unlock()
	if ok {
		ps.RecordNotificationActivated(alert, reason)
	}
	return alert, ok
}

// raiseNotificationLocked adds the alert without writing history; the caller
// holds ps.mutex and has set the timestamp.
func (ps *PumpState) raiseNotificationLocked(alert Alert) (Alert, bool) {
	alert = resolveNotification(alert)
	if alert.Category != NotificationNone && ps.standingLocked(alert.Category, alert.Bit) {
		return alert, false
	}
	if alert.ID == 0 {
		alert.ID = ps.allocateAlertIDLocked()
	}
	ps.ActiveAlerts = append(ps.ActiveAlerts, alert)
	return alert, true
}

// ClearNotification takes bit of category c off the pump, as acknowledging it
// does: every alert standing on it, and for an alarm the bit set directly
// (SetAlarm). Each cleared alert moves into the cleared-alarm log stamped
// `when` (the pump's clock when zero) and is returned, oldest first. No record
// is written; RecordNotificationCleared is the caller's.
func (ps *PumpState) ClearNotification(c NotificationCategory, bit uint, when time.Time) []ClearedAlert {
	if bit > 63 {
		return nil
	}
	if when.IsZero() {
		when = ps.Now()
	}

	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if c == NotificationAlarm {
		ps.AlarmBitmask &^= uint64(1) << bit
	}
	var cleared []ClearedAlert
	standing := make([]Alert, 0, len(ps.ActiveAlerts))
	for _, a := range ps.ActiveAlerts {
		if a.Category == c && a.Bit == bit {
			cleared = append(cleared, ClearedAlert{Alert: a, ClearedAt: when})
			continue
		}
		standing = append(standing, a)
	}
	ps.ActiveAlerts = standing
	ps.clearedAlerts = append(ps.clearedAlerts, cleared...)
	return cleared
}

// IsNotificationStanding reports whether bit of category c is set.
func (ps *PumpState) IsNotificationStanding(c NotificationCategory, bit uint) bool {
	return bit < 64 && ps.NotificationBitmask(c)&(uint64(1)<<bit) != 0
}

// NotificationBitmask is the pump's live bitmask for category c, as its status
// message carries it: bit N for each standing alert on N, and for alarms the
// bits set directly with SetAlarm/SetAlarmBitmask.
func (ps *PumpState) NotificationBitmask(c NotificationCategory) uint64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.notificationBitmaskLocked(c)
}

func (ps *PumpState) notificationBitmaskLocked(c NotificationCategory) uint64 {
	var mask uint64
	if c == NotificationAlarm {
		mask = ps.AlarmBitmask
	}
	for _, a := range ps.ActiveAlerts {
		if a.Category == c && c != NotificationNone && a.Bit < 64 {
			mask |= uint64(1) << a.Bit
		}
	}
	return mask
}

// HighestNotification is what HighestAamResponse reports: the standing
// notification of the highest category (malfunction, alarm, alert, CGM alert,
// reminder), the lowest bit within it. False when nothing stands.
func (ps *PumpState) HighestNotification() (Alert, bool) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	for _, c := range NotificationCategories {
		mask := ps.notificationBitmaskLocked(c)
		if mask == 0 {
			continue
		}
		for bit := uint(0); bit < 64; bit++ {
			if mask&(uint64(1)<<bit) == 0 {
				continue
			}
			for _, a := range ps.ActiveAlerts {
				if a.Category == c && a.Bit == bit {
					return a, true
				}
			}
			return Alert{Type: AlertNotification, Category: c, Bit: bit}, true
		}
	}
	return Alert{}, false
}

// ResumeBlockingAlarms are the standing alarms a pump refuses ResumePumping
// under until each is dismissed: RESUME_PUMP_ALARM and RESUME_PUMP_ALARM2.
func (ps *PumpState) ResumeBlockingAlarms() []uint {
	mask := ps.NotificationBitmask(NotificationAlarm)
	var out []uint
	for _, bit := range []uint{
		uint(currentstatus.AlarmResponseTypeResumePumpAlarm),
		uint(currentstatus.AlarmResponseTypeResumePumpAlarm2),
	} {
		if mask&(uint64(1)<<bit) != 0 {
			out = append(out, bit)
		}
	}
	return out
}
