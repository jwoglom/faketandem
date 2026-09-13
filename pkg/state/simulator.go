package state

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Simulator handles background state evolution
type Simulator struct {
	pumpState      *PumpState
	eventNotifier  EventNotifier
	running        bool
	stopChan       chan bool
	ticker         *time.Ticker
	updateInterval time.Duration
	// lastUpdate is the pump-clock instant the previous update ran at. Basal
	// delivery, battery drain and IOB decay integrate over the elapsed pump
	// time rather than over the ticker interval, so a harness that advances a
	// manual clock by an hour and calls Tick once gets an hour's worth of
	// simulation instead of one tick's worth.
	lastUpdate time.Time
	mutex      sync.Mutex
}

// NewSimulator creates a new background simulator
func NewSimulator(pumpState *PumpState, updateInterval time.Duration) *Simulator {
	return &Simulator{
		pumpState:      pumpState,
		eventNotifier:  &NoOpEventNotifier{}, // Default to no-op
		running:        false,
		stopChan:       make(chan bool),
		updateInterval: updateInterval,
	}
}

// SetEventNotifier sets the event notifier for qualifying events
func (s *Simulator) SetEventNotifier(notifier EventNotifier) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.eventNotifier = notifier
}

// Start begins the background simulation
func (s *Simulator) Start() {
	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return
	}
	s.running = true
	s.ticker = time.NewTicker(s.updateInterval)
	s.mutex.Unlock()

	log.Infof("Starting background simulator with update interval: %v", s.updateInterval)

	go s.simulationLoop()
}

// Stop halts the background simulation
func (s *Simulator) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	log.Info("Stopping background simulator")
	s.running = false
	s.ticker.Stop()
	s.stopChan <- true
}

// simulationLoop runs the background simulation
func (s *Simulator) simulationLoop() {
	for {
		select {
		case <-s.ticker.C:
			s.update()
		case <-s.stopChan:
			return
		}
	}
}

// Tick runs exactly one simulation update, synchronously.
//
// It is what a harness calls after stepping a ManualClock: with the background
// ticker stopped (or simply ignored), Advance + Tick gives a fully
// deterministic pump with no wall-clock waiting anywhere.
func (s *Simulator) Tick() {
	s.update()
}

// elapsedSincePrevious returns how much pump time has passed since the last
// update, and records this one. The first update after Start has no previous
// instant, so it counts as one interval.
func (s *Simulator) elapsedSincePrevious(now time.Time) time.Duration {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	prev := s.lastUpdate
	s.lastUpdate = now
	if prev.IsZero() {
		return s.updateInterval
	}
	elapsed := now.Sub(prev)
	if elapsed < 0 {
		// The clock was moved backwards; deliver nothing rather than
		// un-delivering insulin.
		return 0
	}
	return elapsed
}

// update performs a single simulation update
func (s *Simulator) update() {
	elapsed := s.elapsedSincePrevious(s.pumpState.Now())

	// Update time
	s.pumpState.UpdateTimeSinceReset()

	// Update bolus delivery. A bolus that finished on this tick is recorded
	// after the pump state mutex has been released.
	if completed := s.updateBolusDelivery(); completed != nil {
		s.pumpState.RecordLastBolus(*completed)
		// The end reason goes on the record: it is the wire's
		// completionStatusId, and leaving it off made a bolus that ran to
		// completion look, in the log, exactly like one that was canceled.
		s.pumpState.RecordBolusCompleted(
			completed.BolusID, completed.DeliveredUnits, completed.RequestedUnits, completed.EndReasonID)
	}

	// End a temp rate whose time is up before delivering any basal, so the
	// insulin for this interval goes in at the rate that was actually running.
	s.expireTempRate()

	// Update basal delivery
	s.updateBasalDelivery(elapsed)

	// Update battery
	s.updateBattery(elapsed)

	// Check for alerts
	s.checkAlerts()
}

// updateBolusDelivery simulates bolus insulin delivery. It returns the record
// of a bolus that finished on this tick, if any; recording it has to happen
// after the pump state mutex is released, since RecordLastBolus takes it.
func (s *Simulator) updateBolusDelivery() *LastBolusRecord {
	s.pumpState.mutex.Lock()
	defer s.pumpState.mutex.Unlock()

	if !s.pumpState.Bolus.Active {
		return nil
	}

	// A stalled bolus stays open and keeps answering CurrentBolusStatus, but
	// makes no further progress.
	if s.pumpState.Bolus.Stalled {
		return nil
	}

	// Delivery speed is configurable (see PumpState.BolusRateUnitsPerSecond):
	// a real pump is much slower than the 0.05 U/s default, and driver
	// behavior that depends on a bolus still being unfinished is only
	// reproducible when the two agree.
	deliveryRate := s.pumpState.BolusRateUnitsPerSecond
	if deliveryRate <= 0 {
		deliveryRate = DefaultBolusRateUnitsPerSecond
	}
	elapsed := s.pumpState.Now().Sub(s.pumpState.Bolus.StartTime).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	expectedDelivered := deliveryRate * elapsed

	if expectedDelivered > s.pumpState.Bolus.UnitsTotal {
		expectedDelivered = s.pumpState.Bolus.UnitsTotal
	}

	// Update delivered amount
	oldDelivered := s.pumpState.Bolus.UnitsDelivered
	s.pumpState.Bolus.UnitsDelivered = expectedDelivered

	// Deduct from reservoir
	deltaDelivered := s.pumpState.Bolus.UnitsDelivered - oldDelivered
	if deltaDelivered > 0 {
		s.pumpState.Reservoir.CurrentUnits -= deltaDelivered
		if s.pumpState.Reservoir.CurrentUnits < 0 {
			s.pumpState.Reservoir.CurrentUnits = 0
		}
	}

	// Check if bolus is complete
	if s.pumpState.Bolus.UnitsDelivered >= s.pumpState.Bolus.UnitsTotal {
		bolusID := s.pumpState.Bolus.BolusID
		unitsDelivered := s.pumpState.Bolus.UnitsDelivered
		unitsTotal := s.pumpState.Bolus.UnitsTotal

		sourceID := s.pumpState.Bolus.SourceID
		typeBitmask := s.pumpState.Bolus.TypeBitmask

		s.pumpState.Bolus.Active = false
		log.Infof("Bolus delivery complete: %.2f units delivered", s.pumpState.Bolus.UnitsDelivered)

		// Update IOB (simple calculation - in reality this would decay over time)
		s.pumpState.IOB += s.pumpState.Bolus.UnitsTotal
		s.pumpState.TDD += s.pumpState.Bolus.UnitsTotal

		// Record the completed bolus so LastBolusStatus can report it. This is
		// how a driver finalizes the dose it commanded: it matches bolusId and
		// takes the delivered volume and end time from that record.
		completed := LastBolusRecord{
			BolusID:        bolusID,
			RequestedUnits: unitsTotal,
			DeliveredUnits: unitsDelivered,
			SourceID:       sourceID,
			TypeBitmask:    typeBitmask,
			EndReasonID:    BolusEndReasonCompleted,
			EndTime:        s.pumpState.Now(),
		}

		// Notify qualifying event
		if s.eventNotifier != nil {
			if err := s.eventNotifier.NotifyBolusComplete(bolusID, unitsDelivered, unitsTotal); err != nil {
				log.Warnf("Failed to notify bolus complete: %v", err)
			}
		}

		return &completed
	}

	return nil
}

// expireTempRate ends a temp rate whose programmed duration has run out.
//
// PumpState.EndTempRate writes the one TempRateCompleted record, stamped at the
// programmed end rather than at the tick that noticed it: a coarse tick must
// not move a temp rate's recorded end second. It reports false when something
// else -- a driver's StopTempRate, a replacement temp rate, a suspend -- ended
// this temp rate first, in which case that path already wrote the record and
// this tick writes nothing.
func (s *Simulator) expireTempRate() {
	temp := s.pumpState.GetTempRate()
	if !temp.Active || !s.pumpState.Now().After(temp.EndTime) {
		return
	}

	ended, profileRate, ok := s.pumpState.EndTempRate(temp.EndTime)
	if !ok {
		return
	}
	log.Info("Temp basal expired, returning to normal basal rate")

	s.mutex.Lock()
	notifier := s.eventNotifier
	s.mutex.Unlock()
	if notifier != nil {
		if err := notifier.NotifyBasalRateChange(ended.Rate, profileRate, false); err != nil {
			log.Warnf("Failed to notify temp rate expired: %v", err)
		}
	}
}

// updateBasalDelivery simulates basal insulin delivery over elapsed pump time.
func (s *Simulator) updateBasalDelivery(elapsed time.Duration) {
	s.pumpState.mutex.Lock()
	defer s.pumpState.mutex.Unlock()

	// Calculate basal delivery since last update
	basalRate := s.pumpState.Basal.CurrentRate
	if s.pumpState.Basal.TempBasalActive {
		basalRate = s.pumpState.Basal.TempBasalRate
	}

	// A suspended pump delivers no basal at all.
	if s.pumpState.PumpingSuspended {
		basalRate = 0
	}

	// Basal rate is in units/hour, convert to units/second
	basalPerSecond := basalRate / 3600.0

	// Deliver basal for the pump time that has actually elapsed
	basalDelivered := basalPerSecond * elapsed.Seconds()

	// Deduct from reservoir
	s.pumpState.Reservoir.CurrentUnits -= basalDelivered
	if s.pumpState.Reservoir.CurrentUnits < 0 {
		s.pumpState.Reservoir.CurrentUnits = 0
	}

	// Update IOB and TDD
	s.pumpState.IOB += basalDelivered
	s.pumpState.TDD += basalDelivered

	// Decay IOB slightly (very simplified - real IOB calculation is complex)
	// Assume insulin action time of ~4 hours
	iobDecayPerSecond := s.pumpState.IOB / (4.0 * 3600.0)
	s.pumpState.IOB -= iobDecayPerSecond * elapsed.Seconds()
	if s.pumpState.IOB < 0 {
		s.pumpState.IOB = 0
	}
}

// updateBattery simulates battery drain over elapsed pump time.
func (s *Simulator) updateBattery(elapsed time.Duration) {
	s.pumpState.mutex.Lock()
	defer s.pumpState.mutex.Unlock()

	// Simple battery drain simulation
	// Assume battery lasts ~7 days (168 hours)
	// Drain 100% over 168 hours = ~0.595% per hour = ~0.0001653% per second
	drainPerSecond := 100.0 / (7.0 * 24.0 * 3600.0)
	drainAmount := drainPerSecond * elapsed.Seconds()

	s.pumpState.Battery.Percentage -= int(drainAmount * 100) // Scale for percentage
	if s.pumpState.Battery.Percentage < 0 {
		s.pumpState.Battery.Percentage = 0
	}

	// Log battery level changes at significant thresholds
	if s.pumpState.Battery.Percentage == 50 || s.pumpState.Battery.Percentage == 20 || s.pumpState.Battery.Percentage == 10 {
		log.Infof("Battery level: %d%%", s.pumpState.Battery.Percentage)
	}
}

// checkAlerts checks for alert conditions
func (s *Simulator) checkAlerts() {
	s.pumpState.mutex.Lock()
	defer s.pumpState.mutex.Unlock()

	s.checkReservoirAlert()
	s.checkBatteryAlerts()
}

// checkReservoirAlert checks for low reservoir conditions
func (s *Simulator) checkReservoirAlert() {
	if s.pumpState.Reservoir.CurrentUnits < 20.0 && !s.hasAlert(AlertLowReservoir) {
		log.Warnf("Low reservoir alert: %.1f units remaining", s.pumpState.Reservoir.CurrentUnits)
		alert := s.addAlert(AlertLowReservoir, PriorityWarning, "Low reservoir")
		s.notifyAlert(alert)
		s.notifyReservoirLow(s.pumpState.Reservoir.CurrentUnits)
	}
}

// checkBatteryAlerts checks for low battery conditions
func (s *Simulator) checkBatteryAlerts() {
	batteryPct := s.pumpState.Battery.Percentage

	if batteryPct < 10 && !s.hasAlert(AlertLowBattery) {
		log.Errorf("Critical battery alert: %d%% remaining", batteryPct)
		alert := s.addAlert(AlertLowBattery, PriorityCritical, "Critical battery")
		s.notifyAlert(alert)
		s.notifyBatteryLow(batteryPct)
	} else if batteryPct < 20 && !s.hasAlert(AlertLowBattery) {
		log.Warnf("Low battery alert: %d%% remaining", batteryPct)
		alert := s.addAlert(AlertLowBattery, PriorityWarning, "Low battery")
		s.notifyAlert(alert)
		s.notifyBatteryLow(batteryPct)
	}
}

// notifyAlert sends an alert notification
func (s *Simulator) notifyAlert(alert Alert) {
	if s.eventNotifier != nil {
		if err := s.eventNotifier.NotifyAlert(alert); err != nil {
			log.Warnf("Failed to notify alert: %v", err)
		}
	}
}

// notifyReservoirLow sends a reservoir low notification
func (s *Simulator) notifyReservoirLow(units float64) {
	if s.eventNotifier != nil {
		if err := s.eventNotifier.NotifyReservoirLow(units); err != nil {
			log.Warnf("Failed to notify reservoir low: %v", err)
		}
	}
}

// notifyBatteryLow sends a battery low notification
func (s *Simulator) notifyBatteryLow(percentage int) {
	if s.eventNotifier != nil {
		if err := s.eventNotifier.NotifyBatteryLow(percentage); err != nil {
			log.Warnf("Failed to notify battery low: %v", err)
		}
	}
}

// hasAlert checks if an alert type is already active (must hold mutex)
func (s *Simulator) hasAlert(alertType AlertType) bool {
	for _, alert := range s.pumpState.ActiveAlerts {
		if alert.Type == alertType && !alert.Acknowledged {
			return true
		}
	}
	return false
}

// addAlert adds a new alert (must hold mutex) and returns the alert
func (s *Simulator) addAlert(alertType AlertType, priority AlertPriority, message string) Alert {
	alert := Alert{
		ID:           uint32(len(s.pumpState.ActiveAlerts) + 1),
		Type:         alertType,
		Priority:     priority,
		Message:      message,
		Timestamp:    s.pumpState.Now(),
		Acknowledged: false,
	}
	s.pumpState.ActiveAlerts = append(s.pumpState.ActiveAlerts, alert)
	return alert
}

// GetStats returns simulator statistics
func (s *Simulator) GetStats() map[string]interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return map[string]interface{}{
		"running":        s.running,
		"updateInterval": s.updateInterval.String(),
	}
}
