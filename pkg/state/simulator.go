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
	mutex          sync.Mutex
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

// update performs a single simulation update
func (s *Simulator) update() {
	// Update time
	s.pumpState.UpdateTimeSinceReset()

	// Update bolus delivery
	s.updateBolusDelivery()

	// Update basal delivery
	s.updateBasalDelivery()

	// Update battery
	s.updateBattery()

	// Check for alerts
	s.checkAlerts()
}

// updateBolusDelivery simulates bolus insulin delivery
func (s *Simulator) updateBolusDelivery() {
	s.pumpState.mutex.Lock()
	defer s.pumpState.mutex.Unlock()

	if !s.pumpState.Bolus.Active {
		return
	}

	// Calculate delivery rate (units per second)
	// Assume bolus delivers at 0.05 units/second (3 units/minute)
	deliveryRate := 0.05 // units/second
	elapsed := time.Since(s.pumpState.Bolus.StartTime).Seconds()
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

		s.pumpState.Bolus.Active = false
		log.Infof("Bolus delivery complete: %.2f units delivered", s.pumpState.Bolus.UnitsDelivered)

		// Update IOB (simple calculation - in reality this would decay over time)
		s.pumpState.IOB += s.pumpState.Bolus.UnitsTotal
		s.pumpState.TDD += s.pumpState.Bolus.UnitsTotal

		// Notify qualifying event
		if s.eventNotifier != nil {
			s.eventNotifier.NotifyBolusComplete(bolusID, unitsDelivered, unitsTotal)
		}
	}
}

// updateBasalDelivery simulates basal insulin delivery
func (s *Simulator) updateBasalDelivery() {
	s.pumpState.mutex.Lock()
	defer s.pumpState.mutex.Unlock()

	// Calculate basal delivery since last update
	basalRate := s.pumpState.Basal.CurrentRate
	if s.pumpState.Basal.TempBasalActive {
		basalRate = s.pumpState.Basal.TempBasalRate

		// Check if temp basal has expired
		if time.Now().After(s.pumpState.Basal.TempBasalEnd) {
			log.Info("Temp basal expired, returning to normal basal rate")
			s.pumpState.Basal.TempBasalActive = false
			basalRate = s.pumpState.Basal.CurrentRate
		}
	}

	// Basal rate is in units/hour, convert to units/second
	basalPerSecond := basalRate / 3600.0

	// Deliver basal for the update interval
	basalDelivered := basalPerSecond * s.updateInterval.Seconds()

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
	s.pumpState.IOB -= iobDecayPerSecond * s.updateInterval.Seconds()
	if s.pumpState.IOB < 0 {
		s.pumpState.IOB = 0
	}
}

// updateBattery simulates battery drain
func (s *Simulator) updateBattery() {
	s.pumpState.mutex.Lock()
	defer s.pumpState.mutex.Unlock()

	// Simple battery drain simulation
	// Assume battery lasts ~7 days (168 hours)
	// Drain 100% over 168 hours = ~0.595% per hour = ~0.0001653% per second
	drainPerSecond := 100.0 / (7.0 * 24.0 * 3600.0)
	drainAmount := drainPerSecond * s.updateInterval.Seconds()

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

	// Check low reservoir
	if s.pumpState.Reservoir.CurrentUnits < 20.0 && !s.hasAlert(AlertLowReservoir) {
		log.Warn("Low reservoir alert: %.1f units remaining", s.pumpState.Reservoir.CurrentUnits)
		alert := s.addAlert(AlertLowReservoir, PriorityWarning, "Low reservoir")
		if s.eventNotifier != nil {
			s.eventNotifier.NotifyAlert(alert)
			s.eventNotifier.NotifyReservoirLow(s.pumpState.Reservoir.CurrentUnits)
		}
	}

	// Check low battery
	if s.pumpState.Battery.Percentage < 20 && s.pumpState.Battery.Percentage >= 10 && !s.hasAlert(AlertLowBattery) {
		log.Warnf("Low battery alert: %d%% remaining", s.pumpState.Battery.Percentage)
		alert := s.addAlert(AlertLowBattery, PriorityWarning, "Low battery")
		if s.eventNotifier != nil {
			s.eventNotifier.NotifyAlert(alert)
			s.eventNotifier.NotifyBatteryLow(s.pumpState.Battery.Percentage)
		}
	}

	// Check very low battery
	if s.pumpState.Battery.Percentage < 10 && !s.hasAlert(AlertLowBattery) {
		log.Errorf("Critical battery alert: %d%% remaining", s.pumpState.Battery.Percentage)
		alert := s.addAlert(AlertLowBattery, PriorityCritical, "Critical battery")
		if s.eventNotifier != nil {
			s.eventNotifier.NotifyAlert(alert)
			s.eventNotifier.NotifyBatteryLow(s.pumpState.Battery.Percentage)
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
		Timestamp:    time.Now(),
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
