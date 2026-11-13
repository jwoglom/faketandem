package state

// Refusals are the commands a pump refuses that a test may want the emulator
// to accept instead. Each defaults to on, the pump's behavior; turning one off
// restores the emulator's earlier leniency for a test written against it.
type Refusals struct {
	// TempRateUnderControlIQ refuses SetTempRate while Control-IQ (closed
	// loop) is on: the pump takes no temp rate from the host then.
	TempRateUnderControlIQ bool `json:"temp_rate_under_control_iq"`
	// TempRateWhileRunning refuses SetTempRate while a temp rate runs: the
	// pump wants StopTempRate first.
	TempRateWhileRunning bool `json:"temp_rate_while_running"`
	// TempRateBelowMinimum refuses a non-zero percent whose share of the
	// profile rate is under MinimumTempRateUnitsPerHour.
	TempRateBelowMinimum bool `json:"temp_rate_below_minimum"`
	// TempRateAboveMaxBasal refuses a temp rate above the max basal limit.
	TempRateAboveMaxBasal bool `json:"temp_rate_above_max_basal"`
	// BolusWhileBusy refuses BolusPermission and InitiateBolus while a bolus
	// runs or delivery is suspended.
	BolusWhileBusy bool `json:"bolus_while_busy"`
}

// MinimumTempRateUnitsPerHour is the lowest non-zero temp rate a pump runs: a
// Mobi refused 17% of 0.58 U/hr (0.0986 U/hr).
const MinimumTempRateUnitsPerHour = 0.1

// DefaultRefusals has every refusal on.
func DefaultRefusals() Refusals {
	return Refusals{
		TempRateUnderControlIQ: true,
		TempRateWhileRunning:   true,
		TempRateBelowMinimum:   true,
		TempRateAboveMaxBasal:  true,
		BolusWhileBusy:         true,
	}
}

// GetRefusals returns which refusals are on.
func (ps *PumpState) GetRefusals() Refusals {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.refusals
}

// SetRefusals replaces the refusal switches.
func (ps *PumpState) SetRefusals(r Refusals) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.refusals = r
}
