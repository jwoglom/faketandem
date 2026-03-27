package settings

import log "github.com/sirupsen/logrus"

// registerConstant is a helper to register a constant-mode default with error logging
func registerConstant(manager *Manager, name string, value map[string]interface{}) {
	if err := manager.RegisterDefault(name, &ResponseConfig{
		Mode:  ModeConstant,
		Value: value,
	}); err != nil {
		log.Warnf("Failed to register %s: %v", name, err)
	}
}

// RegisterDefaults registers default configurations for common message types
func RegisterDefaults(manager *Manager) {
	registerSettingsDefaults(manager)
	registerPollingDefaults(manager)
	registerQualifyingEventDefaults(manager)
	registerPumpInfoDefaults(manager)
	registerExampleDefaults(manager)

	log.Info("Registered default settings configurations")
}

// registerSettingsDefaults registers defaults for settings-related messages
func registerSettingsDefaults(manager *Manager) {
	registerConstant(manager, "BasalIQSettingsRequest", map[string]interface{}{
		"enabled":    true,
		"targetBG":   110.0,
		"correction": true,
	})

	registerConstant(manager, "ControlIQSettingsRequest", map[string]interface{}{
		"enabled":      true,
		"sleepMode":    false,
		"exerciseMode": false,
	})

	registerConstant(manager, "PumpGlobalsRequest", map[string]interface{}{
		"maxBasalRate": 5.0,
		"maxBolus":     25.0,
		"insulinType":  "Humalog",
	})

	registerConstant(manager, "TherapySettingsGlobalsRequest", map[string]interface{}{
		"maxBasalRate":     5.0,
		"maxBolus":         25.0,
		"insulinType":      "Humalog",
		"carbRatio":        12.0,
		"correctionFactor": 50.0,
		"targetBG":         100.0,
		"insulinDuration":  4,
	})

	registerConstant(manager, "ControlIQGlobalsRequest", map[string]interface{}{
		"maxBasalRate":     5.0,
		"maxBolus":         25.0,
		"targetBG":         110.0,
		"correctionFactor": 50.0,
	})
}

// registerPollingDefaults registers defaults for messages controlX2 polls every 5 min
func registerPollingDefaults(manager *Manager) {
	registerConstant(manager, "CurrentBatteryV2Request", map[string]interface{}{
		"batteryLevelPercent": 85,
		"batteryVoltage":     2800,
		"isCharging":         false,
	})

	registerConstant(manager, "ControlIQIOBRequest", map[string]interface{}{
		"pumpDisplayedIOB": 250, // 2.5 units * 100
	})

	registerConstant(manager, "InsulinStatusRequest", map[string]interface{}{
		"currentInsulinAmount": 20000, // 200.0 units * 100
		"isLoaded":             true,
	})
}

// registerQualifyingEventDefaults registers defaults for qualifying event handler messages
func registerQualifyingEventDefaults(manager *Manager) {
	registerConstant(manager, "CurrentBasalStatusRequest", map[string]interface{}{
		"currentBasalRate": 85, // 0.85 U/hr * 100
		"isTempBasal":      false,
	})

	registerConstant(manager, "CurrentBolusStatusRequest", map[string]interface{}{
		"bolusStatus": 0,
	})

	registerConstant(manager, "CurrentEGVGuiDataRequest", map[string]interface{}{
		"cgmEGV":          120,
		"cgmTrendRate":    0,
		"cgmAlertStatus":  0,
		"cgmDisplayState": 1,
	})

	registerConstant(manager, "HomeScreenMirrorRequest", map[string]interface{}{
		"status": 0,
	})

	registerConstant(manager, "CGMStatusRequest", map[string]interface{}{
		"sessionState":    1,
		"sensorType":      1,
		"transmitterPair": true,
	})

	registerConstant(manager, "AlertStatusRequest", map[string]interface{}{
		"alertBitmask": 0,
	})

	registerConstant(manager, "AlarmStatusRequest", map[string]interface{}{
		"alarmBitmask": 0,
	})

	registerConstant(manager, "LoadStatusRequest", map[string]interface{}{
		"isLoaded":     true,
		"loadProgress": 100,
	})

	registerConstant(manager, "ProfileStatusRequest", map[string]interface{}{
		"activeProfileIndex": 0,
		"numProfiles":        1,
	})

	registerConstant(manager, "LastBolusStatusV2Request", map[string]interface{}{
		"lastBolusStatus": 0,
	})
}

// registerPumpInfoDefaults registers defaults for pump info messages
func registerPumpInfoDefaults(manager *Manager) {
	registerConstant(manager, "PumpFeaturesV2Request", map[string]interface{}{
		"featureBitmask": 0,
	})

	registerConstant(manager, "PumpVersionRequest", map[string]interface{}{
		"armSwVer":  "7.6.0.0",
		"mspSwVer":  "1.0.0.0",
		"configVer": "1.0.0.0",
	})

	registerConstant(manager, "BleSoftwareInfoRequest", map[string]interface{}{
		"bleSwVer": "1.0.0.0",
	})

	registerConstant(manager, "CommonSoftwareInfoRequest", map[string]interface{}{
		"commonSwVer": "1.0.0.0",
	})
}

// registerExampleDefaults registers example defaults for incremental/time-based modes
func registerExampleDefaults(manager *Manager) {
	if err := manager.RegisterDefault("BatteryStatusRequest", &ResponseConfig{
		Mode: ModeIncremental,
		Values: []map[string]interface{}{
			{"percentage": 100, "voltage": 3.0},
			{"percentage": 75, "voltage": 2.8},
			{"percentage": 50, "voltage": 2.6},
			{"percentage": 25, "voltage": 2.4},
		},
	}); err != nil {
		log.Warnf("Failed to register BatteryStatusRequest: %v", err)
	}

	if err := manager.RegisterDefault("ReservoirStatusRequest", &ResponseConfig{
		Mode:          ModeTimeBased,
		TimingSeconds: []int{0, 60, 120, 180},
		Values: []map[string]interface{}{
			{"units": 200.0, "percentage": 100},
			{"units": 150.0, "percentage": 75},
			{"units": 100.0, "percentage": 50},
			{"units": 50.0, "percentage": 25},
		},
	}); err != nil {
		log.Warnf("Failed to register ReservoirStatusRequest: %v", err)
	}
}
