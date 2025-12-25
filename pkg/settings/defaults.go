package settings

import log "github.com/sirupsen/logrus"

// RegisterDefaults registers default configurations for common message types
func RegisterDefaults(manager *Manager) {
	// BasalIQSettingsRequest - constant response
	manager.RegisterDefault("BasalIQSettingsRequest", &ResponseConfig{
		Mode: ModeConstant,
		Value: map[string]interface{}{
			"enabled":    true,
			"targetBG":   110.0, // mg/dL
			"correction": true,
		},
	})

	// ControlIQSettingsRequest - constant response
	manager.RegisterDefault("ControlIQSettingsRequest", &ResponseConfig{
		Mode: ModeConstant,
		Value: map[string]interface{}{
			"enabled":      true,
			"sleepMode":    false,
			"exerciseMode": false,
		},
	})

	// PumpGlobalsRequest - constant response
	manager.RegisterDefault("PumpGlobalsRequest", &ResponseConfig{
		Mode: ModeConstant,
		Value: map[string]interface{}{
			"maxBasalRate": 5.0,  // U/hr
			"maxBolus":     25.0, // U
			"insulinType":  "Humalog",
		},
	})

	// TherapySettingsGlobalsRequest - constant response
	manager.RegisterDefault("TherapySettingsGlobalsRequest", &ResponseConfig{
		Mode: ModeConstant,
		Value: map[string]interface{}{
			"maxBasalRate":     5.0,  // U/hr
			"maxBolus":         25.0, // U
			"insulinType":      "Humalog",
			"carbRatio":        12.0,  // g/U
			"correctionFactor": 50.0,  // mg/dL/U
			"targetBG":         100.0, // mg/dL
			"insulinDuration":  4,     // hours
		},
	})

	// ControlIQGlobalsRequest - constant response
	manager.RegisterDefault("ControlIQGlobalsRequest", &ResponseConfig{
		Mode: ModeConstant,
		Value: map[string]interface{}{
			"maxBasalRate":     5.0,
			"maxBolus":         25.0,
			"targetBG":         110.0,
			"correctionFactor": 50.0,
		},
	})

	// Example of incremental mode - battery level that decreases
	manager.RegisterDefault("BatteryStatusRequest", &ResponseConfig{
		Mode: ModeIncremental,
		Values: []map[string]interface{}{
			{"percentage": 100, "voltage": 3.0},
			{"percentage": 75, "voltage": 2.8},
			{"percentage": 50, "voltage": 2.6},
			{"percentage": 25, "voltage": 2.4},
		},
	})

	// Example of time-based mode - reservoir level that decreases over time
	manager.RegisterDefault("ReservoirStatusRequest", &ResponseConfig{
		Mode:          ModeTimeBased,
		TimingSeconds: []int{0, 60, 120, 180}, // 0s, 1min, 2min, 3min
		Values: []map[string]interface{}{
			{"units": 200.0, "percentage": 100},
			{"units": 150.0, "percentage": 75},
			{"units": 100.0, "percentage": 50},
			{"units": 50.0, "percentage": 25},
		},
	})

	log.Info("Registered default settings configurations")
}
