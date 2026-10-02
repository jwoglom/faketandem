package state

// DeliveryLimits are the pump's own ceilings, in milliunits (per hour for basal), as
// GlobalMaxBolusSettingsResponse and BasalLimitSettingsResponse carry them.
type DeliveryLimits struct {
	MaxBolusMilliunits        int
	MaxBolusDefaultMilliunits int
	MaxBasalMilliunits        int
	MaxBasalDefaultMilliunits int
}

// DefaultDeliveryLimits are the values in pumpX2's test vectors for the two responses.
func DefaultDeliveryLimits() DeliveryLimits {
	return DeliveryLimits{
		MaxBolusMilliunits:        25000,
		MaxBolusDefaultMilliunits: 10000,
		MaxBasalMilliunits:        5000,
		MaxBasalDefaultMilliunits: 3000,
	}
}

// GetDeliveryLimits returns the pump's delivery limits.
func (ps *PumpState) GetDeliveryLimits() DeliveryLimits {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.deliveryLimits
}

// SetMaxBolusMilliunits sets the pump's max bolus.
func (ps *PumpState) SetMaxBolusMilliunits(milliunits int) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.deliveryLimits.MaxBolusMilliunits = milliunits
}

// SetMaxBasalMilliunits sets the pump's max basal rate, in milliunits per hour.
func (ps *PumpState) SetMaxBasalMilliunits(milliunits int) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.deliveryLimits.MaxBasalMilliunits = milliunits
}
