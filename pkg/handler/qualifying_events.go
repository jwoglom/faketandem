package handler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/reqlog"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// Qualifying event bit values, matching pumpX2's
// response.qualifyingEvent.QualifyingEvent enum. The real protocol sends
// these as a raw little-endian uint32 bitmask directly on the
// QualifyingEvents characteristic -- it is NOT a cliparser-encoded Message
// with its own packet framing, and there is no per-occurrence payload. On
// receiving a nonzero value the client re-requests details via the
// suggested follow-up status-request messages (e.g. AlertStatusRequest for
// qualifyingEventAlert), which are already served by the handlers
// registered elsewhere in this package.
const (
	qualifyingEventPumpSuspend      uint32 = 64
	qualifyingEventPumpResume       uint32 = 128
	qualifyingEventBasalChange      uint32 = 512
	qualifyingEventBolusChange      uint32 = 1024
	qualifyingEventRemainingInsulin uint32 = 262144
	qualifyingEventBattery          uint32 = 65536
)

// QualifyingEventsNotifier sends qualifying event bitmask notifications
type QualifyingEventsNotifier struct {
	ble       bluetooth.Transport
	pumpState *state.PumpState
	// requestLog, when set, records every bitmask that went out (or failed to)
	// so a harness can assert on which qualifying events the pump raised.
	requestLog *reqlog.Log
}

// SetRequestLog attaches the pump-side message record.
func (qe *QualifyingEventsNotifier) SetRequestLog(l *reqlog.Log) {
	qe.requestLog = l
}

// NotifyBitmask sends an arbitrary qualifying-event bitmask. A scenario uses
// it to raise exactly the events a real pump would raise for a pump-initiated
// change, including combinations the emulator has no internal trigger for.
func (qe *QualifyingEventsNotifier) NotifyBitmask(bits uint32) error {
	return qe.sendBitmask(bits)
}

// NewQualifyingEventsNotifier creates a new qualifying events notifier
func NewQualifyingEventsNotifier(ble bluetooth.Transport, pumpState *state.PumpState) *QualifyingEventsNotifier {
	return &QualifyingEventsNotifier{
		ble:       ble,
		pumpState: pumpState,
	}
}

// NotifyBolusStart sends the BOLUS_CHANGE qualifying event for a bolus start
func (qe *QualifyingEventsNotifier) NotifyBolusStart(bolusID uint32, units float64) error {
	log.Infof("Sending BOLUS_CHANGE qualifying event (bolus start): bolusID=%d, units=%.2f", bolusID, units)
	return qe.sendBitmask(qualifyingEventBolusChange)
}

// NotifyBolusComplete sends the BOLUS_CHANGE qualifying event for a bolus completion
func (qe *QualifyingEventsNotifier) NotifyBolusComplete(bolusID uint32, delivered float64, total float64) error {
	log.Infof("Sending BOLUS_CHANGE qualifying event (bolus complete): bolusID=%d, delivered=%.2f/%.2f",
		bolusID, delivered, total)
	return qe.sendBitmask(qualifyingEventBolusChange)
}

// NotifyBolusCanceled sends the BOLUS_CHANGE qualifying event for a bolus cancellation
func (qe *QualifyingEventsNotifier) NotifyBolusCanceled(bolusID uint32, delivered float64, total float64) error {
	log.Infof("Sending BOLUS_CHANGE qualifying event (bolus canceled): bolusID=%d, delivered=%.2f/%.2f",
		bolusID, delivered, total)
	return qe.sendBitmask(qualifyingEventBolusChange)
}

// NotifyAlert sends the qualifying event of the alert's category: ALERT,
// ALARM, MALFUNCTION, REMINDER or CGM_ALERT, which is the bit a driver keys
// its follow-up status read on.
func (qe *QualifyingEventsNotifier) NotifyAlert(alert state.Alert) error {
	bits := NotificationQualifyingEvent(alert.Category)
	log.Infof("Sending qualifying event 0x%x for %s %s (id=%d): %s",
		bits, alert.Category, alert.Name(), alert.ID, alert.Message)
	return qe.sendBitmask(bits)
}

// NotifyAlertCleared sends the alert's category qualifying event again, so the
// client re-reads that status and sees it has cleared.
func (qe *QualifyingEventsNotifier) NotifyAlertCleared(alert state.Alert) error {
	bits := NotificationQualifyingEvent(alert.Category)
	log.Infof("Sending qualifying event 0x%x (%s %s cleared)", bits, alert.Category, alert.Name())
	return qe.sendBitmask(bits)
}

// NotificationQualifyingEvent is the qualifying event a change in category c
// raises. An alert with no bit raises ALERT, as every alert did before
// categories existed.
func NotificationQualifyingEvent(c state.NotificationCategory) uint32 {
	switch c {
	case state.NotificationAlarm:
		return QEAlarm
	case state.NotificationMalfunction:
		return QEMalfunction
	case state.NotificationReminder:
		return QEReminder
	case state.NotificationCGMAlert:
		return QECGMAlert
	default:
		return QEAlert
	}
}

// NotifyBasalRateChange sends the BASAL_CHANGE qualifying event
func (qe *QualifyingEventsNotifier) NotifyBasalRateChange(oldRate, newRate float64, tempBasal bool) error {
	log.Infof("Sending BASAL_CHANGE qualifying event: %.2f -> %.2f (temp: %v)",
		oldRate, newRate, tempBasal)
	return qe.sendBitmask(qualifyingEventBasalChange)
}

// NotifyReservoirLow sends the REMAINING_INSULIN qualifying event
func (qe *QualifyingEventsNotifier) NotifyReservoirLow(units float64) error {
	log.Infof("Sending REMAINING_INSULIN qualifying event: %.1f units remaining", units)
	return qe.sendBitmask(qualifyingEventRemainingInsulin)
}

// NotifyBatteryLow sends the BATTERY qualifying event
func (qe *QualifyingEventsNotifier) NotifyBatteryLow(percentage int) error {
	log.Infof("Sending BATTERY qualifying event: %d%% remaining", percentage)
	return qe.sendBitmask(qualifyingEventBattery)
}

// NotifyPumpSuspended sends the PUMP_SUSPEND qualifying event
func (qe *QualifyingEventsNotifier) NotifyPumpSuspended(reason string) error {
	log.Infof("Sending PUMP_SUSPEND qualifying event: reason=%s", reason)
	return qe.sendBitmask(qualifyingEventPumpSuspend)
}

// NotifyPumpResumed sends the PUMP_RESUME qualifying event
func (qe *QualifyingEventsNotifier) NotifyPumpResumed() error {
	log.Info("Sending PUMP_RESUME qualifying event")
	return qe.sendBitmask(qualifyingEventPumpResume)
}

// sendBitmask sends a raw little-endian uint32 qualifying event bitmask
// notification on the QualifyingEvents characteristic
func (qe *QualifyingEventsNotifier) sendBitmask(bits uint32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, bits)

	log.Debugf("Sending qualifying event bitmask 0x%08x on %s", bits, bluetooth.CharQualifyingEvents)

	err := qe.ble.Notify(bluetooth.CharQualifyingEvents, buf)
	if qe.requestLog != nil {
		entry := reqlog.Entry{
			Kind:           reqlog.KindNotification,
			Characteristic: bluetooth.CharQualifyingEvents.String(),
			Message:        fmt.Sprintf("QualifyingEvents(0x%08x)", bits),
			Fragments:      []string{hex.EncodeToString(buf)},
			FragmentsSent:  1,
		}
		if err != nil {
			entry.FragmentsSent = 0
			entry.Note = "send error: " + err.Error()
		}
		qe.requestLog.Append(entry)
	}
	if err != nil {
		return fmt.Errorf("failed to send qualifying event notification: %w", err)
	}

	return nil
}
