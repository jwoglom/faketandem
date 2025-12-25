package handler

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// QualifyingEventsNotifier handles sending qualifying event notifications
type QualifyingEventsNotifier struct {
	bridge    *pumpx2.Bridge
	ble       *bluetooth.Ble
	pumpState *state.PumpState
	txID      uint8
}

// NewQualifyingEventsNotifier creates a new qualifying events notifier
func NewQualifyingEventsNotifier(bridge *pumpx2.Bridge, ble *bluetooth.Ble, pumpState *state.PumpState) *QualifyingEventsNotifier {
	return &QualifyingEventsNotifier{
		bridge:    bridge,
		ble:       ble,
		pumpState: pumpState,
		txID:      200, // Start qualifying events at txID 200 to avoid conflicts
	}
}

// nextTxID returns the next transaction ID for qualifying events
func (qe *QualifyingEventsNotifier) nextTxID() uint8 {
	qe.txID++
	return qe.txID
}

// NotifyBolusStart sends a bolus start qualifying event
func (qe *QualifyingEventsNotifier) NotifyBolusStart(bolusID uint32, units float64) error {
	log.Infof("Sending BolusStart qualifying event: bolusID=%d, units=%.2f", bolusID, units)

	// Build the event message
	params := map[string]interface{}{
		"bolusId":    bolusID,
		"units":      units,
		"timestamp":  time.Now().Unix(),
		"bolusType":  "normal", // Could be "normal", "extended", "dual"
		"eventType":  "start",
	}

	return qe.sendEvent("BolusStartEvent", params)
}

// NotifyBolusComplete sends a bolus completion qualifying event
func (qe *QualifyingEventsNotifier) NotifyBolusComplete(bolusID uint32, delivered float64, total float64) error {
	log.Infof("Sending BolusComplete qualifying event: bolusID=%d, delivered=%.2f/%.2f",
		bolusID, delivered, total)

	params := map[string]interface{}{
		"bolusId":   bolusID,
		"delivered": delivered,
		"total":     total,
		"timestamp": time.Now().Unix(),
		"eventType": "complete",
	}

	return qe.sendEvent("BolusCompleteEvent", params)
}

// NotifyBolusCanceled sends a bolus cancellation qualifying event
func (qe *QualifyingEventsNotifier) NotifyBolusCanceled(bolusID uint32, delivered float64, total float64) error {
	log.Infof("Sending BolusCanceled qualifying event: bolusID=%d, delivered=%.2f/%.2f",
		bolusID, delivered, total)

	params := map[string]interface{}{
		"bolusId":   bolusID,
		"delivered": delivered,
		"total":     total,
		"timestamp": time.Now().Unix(),
		"eventType": "canceled",
	}

	return qe.sendEvent("BolusCanceledEvent", params)
}

// NotifyAlert sends an alert qualifying event
func (qe *QualifyingEventsNotifier) NotifyAlert(alert state.Alert) error {
	log.Infof("Sending Alert qualifying event: type=%d, priority=%d, message=%s",
		alert.Type, alert.Priority, alert.Message)

	params := map[string]interface{}{
		"alertId":   alert.ID,
		"alertType": int(alert.Type),
		"priority":  int(alert.Priority),
		"message":   alert.Message,
		"timestamp": alert.Timestamp.Unix(),
		"eventType": "alert",
	}

	return qe.sendEvent("AlertEvent", params)
}

// NotifyAlertCleared sends an alert cleared qualifying event
func (qe *QualifyingEventsNotifier) NotifyAlertCleared(alertID uint32) error {
	log.Infof("Sending AlertCleared qualifying event: alertID=%d", alertID)

	params := map[string]interface{}{
		"alertId":   alertID,
		"timestamp": time.Now().Unix(),
		"eventType": "alertCleared",
	}

	return qe.sendEvent("AlertClearedEvent", params)
}

// NotifyBasalRateChange sends a basal rate change qualifying event
func (qe *QualifyingEventsNotifier) NotifyBasalRateChange(oldRate, newRate float64, tempBasal bool) error {
	log.Infof("Sending BasalRateChange qualifying event: %.2f -> %.2f (temp: %v)",
		oldRate, newRate, tempBasal)

	params := map[string]interface{}{
		"oldRate":   oldRate,
		"newRate":   newRate,
		"tempBasal": tempBasal,
		"timestamp": time.Now().Unix(),
		"eventType": "basalChange",
	}

	return qe.sendEvent("BasalRateChangeEvent", params)
}

// NotifyReservoirLow sends a reservoir low qualifying event
func (qe *QualifyingEventsNotifier) NotifyReservoirLow(units float64) error {
	log.Infof("Sending ReservoirLow qualifying event: %.1f units remaining", units)

	params := map[string]interface{}{
		"unitsRemaining": units,
		"timestamp":      time.Now().Unix(),
		"eventType":      "reservoirLow",
	}

	return qe.sendEvent("ReservoirLowEvent", params)
}

// NotifyBatteryLow sends a battery low qualifying event
func (qe *QualifyingEventsNotifier) NotifyBatteryLow(percentage int) error {
	log.Infof("Sending BatteryLow qualifying event: %d%% remaining", percentage)

	params := map[string]interface{}{
		"percentage": percentage,
		"timestamp":  time.Now().Unix(),
		"eventType":  "batteryLow",
	}

	return qe.sendEvent("BatteryLowEvent", params)
}

// NotifyPumpSuspended sends a pump suspended qualifying event
func (qe *QualifyingEventsNotifier) NotifyPumpSuspended(reason string) error {
	log.Infof("Sending PumpSuspended qualifying event: reason=%s", reason)

	params := map[string]interface{}{
		"reason":    reason,
		"timestamp": time.Now().Unix(),
		"eventType": "suspended",
	}

	return qe.sendEvent("PumpSuspendedEvent", params)
}

// NotifyPumpResumed sends a pump resumed qualifying event
func (qe *QualifyingEventsNotifier) NotifyPumpResumed() error {
	log.Info("Sending PumpResumed qualifying event")

	params := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"eventType": "resumed",
	}

	return qe.sendEvent("PumpResumedEvent", params)
}

// sendEvent sends a qualifying event notification
func (qe *QualifyingEventsNotifier) sendEvent(eventType string, params map[string]interface{}) error {
	// Get next transaction ID
	txID := qe.nextTxID()

	// Encode the event message
	encoded, err := qe.bridge.EncodeMessage(int(txID), eventType, params)
	if err != nil {
		log.Errorf("Failed to encode %s: %v", eventType, err)
		return err
	}

	// Send on QualifyingEvents characteristic
	if err := qe.sendNotification(bluetooth.CharQualifyingEvents, encoded); err != nil {
		log.Errorf("Failed to send %s notification: %v", eventType, err)
		return err
	}

	return nil
}

// sendNotification sends an encoded message as a notification
func (qe *QualifyingEventsNotifier) sendNotification(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage) error {
	log.Debugf("Sending qualifying event notification: %s on %s, %d packet(s)",
		msg.MessageType, charType, len(msg.Packets))

	for i, packetHex := range msg.Packets {
		// Decode hex packet
		packetData := make([]byte, len(packetHex)/2)
		for j := 0; j < len(packetHex); j += 2 {
			var b byte
			_, err := fmt.Sscanf(packetHex[j:j+2], "%02x", &b)
			if err != nil {
				return fmt.Errorf("failed to decode packet %d: %w", i, err)
			}
			packetData[j/2] = b
		}

		// Send via BLE notification
		if err := qe.ble.Notify(charType, packetData); err != nil {
			return fmt.Errorf("failed to send packet %d: %w", i, err)
		}

		log.Tracef("Sent qualifying event packet %d/%d", i+1, len(msg.Packets))
	}

	return nil
}
