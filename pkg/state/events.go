package state

// EventNotifier defines an interface for notifying about pump events
// This allows the simulator to notify without depending on the handler package
type EventNotifier interface {
	// NotifyBolusStart notifies that a bolus has started
	NotifyBolusStart(bolusID uint32, units float64) error

	// NotifyBolusComplete notifies that a bolus has completed
	NotifyBolusComplete(bolusID uint32, delivered float64, total float64) error

	// NotifyBolusCanceled notifies that a bolus was canceled
	NotifyBolusCanceled(bolusID uint32, delivered float64, total float64) error

	// NotifyAlert notifies about an alert
	NotifyAlert(alert Alert) error

	// NotifyAlertCleared notifies that an alert was cleared
	NotifyAlertCleared(alertID uint32) error

	// NotifyBasalRateChange notifies about a basal rate change
	NotifyBasalRateChange(oldRate, newRate float64, tempBasal bool) error

	// NotifyReservoirLow notifies about low reservoir
	NotifyReservoirLow(units float64) error

	// NotifyBatteryLow notifies about low battery
	NotifyBatteryLow(percentage int) error

	// NotifyPumpSuspended notifies that the pump was suspended
	NotifyPumpSuspended(reason string) error

	// NotifyPumpResumed notifies that the pump was resumed
	NotifyPumpResumed() error
}

// NoOpEventNotifier is a no-op implementation of EventNotifier
type NoOpEventNotifier struct{}

// NotifyBolusStart is a no-op implementation
func (n *NoOpEventNotifier) NotifyBolusStart(bolusID uint32, units float64) error {
	return nil
}

// NotifyBolusComplete is a no-op implementation
func (n *NoOpEventNotifier) NotifyBolusComplete(bolusID uint32, delivered float64, total float64) error {
	return nil
}

// NotifyBolusCanceled is a no-op implementation
func (n *NoOpEventNotifier) NotifyBolusCanceled(bolusID uint32, delivered float64, total float64) error {
	return nil
}

// NotifyAlert is a no-op implementation
func (n *NoOpEventNotifier) NotifyAlert(alert Alert) error {
	return nil
}

// NotifyAlertCleared is a no-op implementation
func (n *NoOpEventNotifier) NotifyAlertCleared(alertID uint32) error {
	return nil
}

// NotifyBasalRateChange is a no-op implementation
func (n *NoOpEventNotifier) NotifyBasalRateChange(oldRate, newRate float64, tempBasal bool) error {
	return nil
}

// NotifyReservoirLow is a no-op implementation
func (n *NoOpEventNotifier) NotifyReservoirLow(units float64) error {
	return nil
}

// NotifyBatteryLow is a no-op implementation
func (n *NoOpEventNotifier) NotifyBatteryLow(percentage int) error {
	return nil
}

// NotifyPumpSuspended is a no-op implementation
func (n *NoOpEventNotifier) NotifyPumpSuspended(reason string) error {
	return nil
}

// NotifyPumpResumed is a no-op implementation
func (n *NoOpEventNotifier) NotifyPumpResumed() error {
	return nil
}
