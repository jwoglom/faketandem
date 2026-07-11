package handler

import (
	"encoding/binary"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
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
	qualifyingEventAlert            uint32 = 1
	qualifyingEventPumpSuspend      uint32 = 64
	qualifyingEventPumpResume       uint32 = 128
	qualifyingEventBasalChange      uint32 = 512
	qualifyingEventBolusChange      uint32 = 1024
	qualifyingEventRemainingInsulin uint32 = 262144
	qualifyingEventBattery          uint32 = 65536
)

// QualifyingEventsNotifier sends qualifying event bitmask notifications
type QualifyingEventsNotifier struct {
	ble       *bluetooth.Ble
	pumpState *state.PumpState
}

// NewQualifyingEventsNotifier creates a new qualifying events notifier
func NewQualifyingEventsNotifier(ble *bluetooth.Ble, pumpState *state.PumpState) *QualifyingEventsNotifier {
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

// NotifyAlert sends the ALERT qualifying event
func (qe *QualifyingEventsNotifier) NotifyAlert(alert state.Alert) error {
	log.Infof("Sending ALERT qualifying event: type=%d, priority=%d, message=%s",
		alert.Type, alert.Priority, alert.Message)
	return qe.sendBitmask(qualifyingEventAlert)
}

// NotifyAlertCleared sends the ALERT qualifying event to prompt the client
// to re-poll alert status and observe it has cleared
func (qe *QualifyingEventsNotifier) NotifyAlertCleared(alertID uint32) error {
	log.Infof("Sending ALERT qualifying event (alert cleared): alertID=%d", alertID)
	return qe.sendBitmask(qualifyingEventAlert)
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

	if err := qe.ble.Notify(bluetooth.CharQualifyingEvents, buf); err != nil {
		return fmt.Errorf("failed to send qualifying event notification: %w", err)
	}

	return nil
}
