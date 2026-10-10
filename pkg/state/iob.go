package state

import (
	"math"
	"time"
)

// Insulin on board, as a Tandem pump reports it in ControlIQIOBResponse.
//
// The pump keeps two figures and reports both. Its "Mudaliar" IOB counts bolus
// insulin alone, over the active profile's insulin duration; it is the one shown
// with Control-IQ off (iobType 0). Its "Swan 6hr" IOB also counts basal
// delivered above and below the profile rate; it is the one shown with
// Control-IQ on (iobType 1). Tandem does not publish either curve, so each dose
// here decays linearly over its duration. That keeps what a driver may rely on:
// what is left of a bolus is never more than the bolus, and none of it is left
// once the duration has passed.

// IOB types ControlIQIOBResponse carries.
const (
	IOBTypeMudaliar = 0
	IOBTypeSwan6Hr  = 1
)

// swanDuration is after pumpX2's name for the model.
const swanDuration = 6 * time.Hour

// defaultInsulinDuration applies while the pump holds no profile.
const defaultInsulinDuration = 5 * time.Hour

// iobRetention bounds the doses kept: longer than any insulin duration a pump
// accepts (2 to 8 hours) and than the Swan model's.
const iobRetention = 24 * time.Hour

type iobDose struct {
	at    time.Time
	units float64
	// bolusID is 0 for a basal deviation or a staged amount.
	bolusID uint32
}

// insulinOnBoard is guarded by the pump state's mutex.
type insulinOnBoard struct {
	boluses []iobDose
	// deviations are basal delivered minus the profile's, per 5-minute cycle,
	// so the continuous delivery path's per-tick amounts are folded together.
	deviations []iobDose
}

// IOBReading is the pump's IOB at one instant, in units.
type IOBReading struct {
	Type int
	// Mudaliar is bolus insulin on board. MudaliarTotal is the full size of the
	// boluses it counts, and TimeRemaining how long until the last of them is
	// no longer counted.
	Mudaliar      float64
	MudaliarTotal float64
	TimeRemaining time.Duration
	// Swan6Hr never goes below zero, though basal below the profile counts
	// against it.
	Swan6Hr float64
}

// Displayed is the IOB the pump shows, as pumpX2's getPumpDisplayedIOB picks it.
func (r IOBReading) Displayed() float64 {
	if r.Type == IOBTypeSwan6Hr {
		return r.Swan6Hr
	}
	return r.Mudaliar
}

// recordBolus sets what a bolus has delivered so far, dated at its start.
func (iob *insulinOnBoard) recordBolus(bolusID uint32, start time.Time, delivered float64) {
	for i := range iob.boluses {
		if iob.boluses[i].bolusID == bolusID {
			iob.boluses[i].units = delivered
			return
		}
	}
	iob.prune(start)
	iob.boluses = append(iob.boluses, iobDose{at: start, units: delivered, bolusID: bolusID})
}

func (iob *insulinOnBoard) recordBasalDeviation(at time.Time, units float64) {
	if math.Abs(units) < 1e-9 {
		return
	}
	if n := len(iob.deviations); n > 0 {
		if last := &iob.deviations[n-1]; at.Sub(last.at) < BasalCycleLength {
			last.units += units
			return
		}
	}
	iob.prune(at)
	iob.deviations = append(iob.deviations, iobDose{at: at, units: units})
}

// stage replaces every dose with one of units given at at, so a pump can start
// with insulin on board from before the emulator ran.
func (iob *insulinOnBoard) stage(at time.Time, units float64) {
	iob.boluses = []iobDose{{at: at, units: units}}
	iob.deviations = nil
}

func (iob *insulinOnBoard) prune(now time.Time) {
	keep := func(doses []iobDose) []iobDose {
		kept := doses[:0]
		for _, d := range doses {
			if now.Sub(d.at) < iobRetention {
				kept = append(kept, d)
			}
		}
		return kept
	}
	iob.boluses = keep(iob.boluses)
	iob.deviations = keep(iob.deviations)
}

func (iob *insulinOnBoard) read(now time.Time, insulinDuration time.Duration, closedLoop bool) IOBReading {
	r := IOBReading{Type: IOBTypeMudaliar}
	if closedLoop {
		r.Type = IOBTypeSwan6Hr
	}
	for _, d := range iob.boluses {
		if left := d.remaining(now, insulinDuration); left > 0 {
			r.Mudaliar += left
			r.MudaliarTotal += d.units
			if until := d.at.Add(insulinDuration).Sub(now); until > r.TimeRemaining {
				r.TimeRemaining = until
			}
		}
		r.Swan6Hr += d.remaining(now, swanDuration)
	}
	for _, d := range iob.deviations {
		r.Swan6Hr += d.remaining(now, swanDuration)
	}
	r.Swan6Hr = math.Max(0, r.Swan6Hr)
	return r
}

// remaining counts a dose dated after now (the clock was set back) in full.
func (d iobDose) remaining(now time.Time, duration time.Duration) float64 {
	elapsed := now.Sub(d.at)
	if elapsed >= duration {
		return 0
	}
	if elapsed < 0 {
		elapsed = 0
	}
	return d.units * (1 - float64(elapsed)/float64(duration))
}

// ReadIOB returns the pump's IOB now.
func (ps *PumpState) ReadIOB() IOBReading {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.ReadIOBUnlocked()
}

// ReadIOBUnlocked is ReadIOB for a caller already holding the state lock.
func (ps *PumpState) ReadIOBUnlocked() IOBReading {
	return ps.iob.read(ps.Now(), ps.insulinDurationUnlocked(), ps.ClosedLoopEnabled)
}

// GetIOB returns the IOB the pump shows, in units.
func (ps *PumpState) GetIOB() float64 {
	return ps.ReadIOB().Displayed()
}

// SetIOB stages units of insulin on board, given now, in place of every dose
// the pump has counted.
func (ps *PumpState) SetIOB(units float64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.iob.stage(ps.Now(), units)
}

// insulinDurationUnlocked is the active profile's insulin duration.
func (ps *PumpState) insulinDurationUnlocked() time.Duration {
	if len(ps.idpProfiles) > 0 && ps.idpProfiles[0].InsulinDuration > 0 {
		return time.Duration(ps.idpProfiles[0].InsulinDuration) * time.Minute
	}
	return defaultInsulinDuration
}

// recordBolusDeliveryUnlocked counts what the bolus in progress has delivered.
func (ps *PumpState) recordBolusDeliveryUnlocked() {
	ps.iob.recordBolus(ps.Bolus.BolusID, ps.Bolus.StartTime, ps.Bolus.UnitsDelivered)
}
