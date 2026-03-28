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
	registerNotificationDefaults(manager)
	registerControlIQDefaults(manager)
	registerBolusAndBasalDefaults(manager)
	registerGlobalSettingsDefaults(manager)
	registerPumpInfoDefaults(manager)
	registerExampleDefaults(manager)
	registerAdditionalStatusDefaults(manager)

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

// registerNotificationDefaults registers defaults for notification bundle / alarm / malfunction handlers
func registerNotificationDefaults(manager *Manager) {
	// HighestAamRequest (opcode 120) — from ALARM qualifying event
	registerConstant(manager, "HighestAamRequest", map[string]interface{}{
		"highestAam": 0, // no active alarms
	})

	// ActiveAamBitsRequest — from ALARM qualifying event (two variants with different params)
	registerConstant(manager, "ActiveAamBitsRequest", map[string]interface{}{
		"activeBitmask": 0, // no active alarm/alert bits
	})

	// MalfunctionStatusRequest (opcode 118) — from MALFUNCTION qualifying event
	registerConstant(manager, "MalfunctionStatusRequest", map[string]interface{}{
		"malfunctionBitmask": 0, // no malfunctions
	})

	// ReminderStatusRequest — from REMINDER qualifying event
	registerConstant(manager, "ReminderStatusRequest", map[string]interface{}{
		"reminderBitmask": 0, // no active reminders
	})

	// CGMAlertStatusRequest — from CGM_ALERT qualifying event
	registerConstant(manager, "CGMAlertStatusRequest", map[string]interface{}{
		"cgmAlertBitmask": 0, // no CGM alerts
	})
}

// registerControlIQDefaults registers defaults for ControlIQ info and sleep schedule
func registerControlIQDefaults(manager *Manager) {
	// ControlIQInfoV1Request — from HOME_SCREEN_CHANGE, CONTROL_IQ_INFO qualifying events
	registerConstant(manager, "ControlIQInfoV1Request", map[string]interface{}{
		"controlIQEnabled": true,
		"currentUserMode":  0, // normal mode
		"weight":           70,
		"totalDailyInsulin": 40,
	})

	// ControlIQInfoV2Request — newer variant
	registerConstant(manager, "ControlIQInfoV2Request", map[string]interface{}{
		"controlIQEnabled": true,
		"currentUserMode":  0,
		"weight":           70,
		"totalDailyInsulin": 40,
	})

	// ControlIQSleepScheduleRequest — from CONTROL_IQ_INFO, CONTROL_IQ_SLEEP qualifying events
	registerConstant(manager, "ControlIQSleepScheduleRequest", map[string]interface{}{
		"enabled":   false,
		"startTime": 0,
		"endTime":   0,
	})

	// BasalIQStatusRequest — from BASAL_IQ_STATUS qualifying event
	registerConstant(manager, "BasalIQStatusRequest", map[string]interface{}{
		"enabled": true,
		"status":  0,
	})

	// NonControlIQIOBRequest — IOB for non-ControlIQ pumps
	registerConstant(manager, "NonControlIQIOBRequest", map[string]interface{}{
		"pumpDisplayedIOB": 250,
	})
}

// registerBolusAndBasalDefaults registers defaults for extended bolus and temp rate handlers
func registerBolusAndBasalDefaults(manager *Manager) {
	// ExtendedBolusStatusRequest — from BOLUS_CHANGE qualifying event
	registerConstant(manager, "ExtendedBolusStatusRequest", map[string]interface{}{
		"status": 0, // no extended bolus active
	})

	// ExtendedBolusStatusV2Request — V2 variant
	registerConstant(manager, "ExtendedBolusStatusV2Request", map[string]interface{}{
		"status": 0,
	})

	// LastBolusStatusV3Request — V3 variant (renamed from LAST_BOLUS_STATUS_C)
	registerConstant(manager, "LastBolusStatusV3Request", map[string]interface{}{
		"lastBolusStatus": 0,
	})

	// TempRateRequest — from BASAL_CHANGE qualifying event
	registerConstant(manager, "TempRateRequest", map[string]interface{}{
		"tempRateActive": false,
	})

	// TempRateStatusRequest — temp rate status
	registerConstant(manager, "TempRateStatusRequest", map[string]interface{}{
		"tempRateActive": false,
	})

	// LastBGRequest — from BG qualifying event
	registerConstant(manager, "LastBGRequest", map[string]interface{}{
		"bgValue":   0,
		"timestamp": 0,
	})

	// BolusPermissionChangeReasonRequest — from BOLUS_PERMISSION_REVOKED qualifying event
	registerConstant(manager, "BolusPermissionChangeReasonRequest", map[string]interface{}{
		"reason": 0,
	})
}

// registerGlobalSettingsDefaults registers defaults for global pump settings messages
func registerGlobalSettingsDefaults(manager *Manager) {
	// GlobalMaxBolusSettingsRequest — from GLOBAL_PUMP_SETTINGS qualifying event
	registerConstant(manager, "GlobalMaxBolusSettingsRequest", map[string]interface{}{
		"maxBolus": 2500, // 25.0 units * 100
	})

	// BasalLimitSettingsRequest — from GLOBAL_PUMP_SETTINGS qualifying event
	registerConstant(manager, "BasalLimitSettingsRequest", map[string]interface{}{
		"maxBasalRate": 500, // 5.0 U/hr * 100
	})

	// LocalizationRequest — from GLOBAL_PUMP_SETTINGS qualifying event
	registerConstant(manager, "LocalizationRequest", map[string]interface{}{
		"language": 0, // English
		"units":    0, // mg/dL
	})

	// PumpSettingsRequest — from GLOBAL_PUMP_SETTINGS qualifying event
	registerConstant(manager, "PumpSettingsRequest", map[string]interface{}{
		"lowInsulinThreshold": 20,
		"autoOff":             false,
	})

	// SendTipsControlGenericTestRequest — control message
	registerConstant(manager, "SendTipsControlGenericTestRequest", map[string]interface{}{
		"status": 0,
	})
}

// registerAdditionalStatusDefaults registers defaults for additional status handlers
func registerAdditionalStatusDefaults(manager *Manager) {
	registerConstant(manager, "IDPSegmentRequest", map[string]interface{}{
		"status":     0,
		"idpId":      1,
		"segmentId":  0,
		"startTime":  0,
		"basalRate":  850,
		"carbRatio":  10000,
		"correction": 50,
		"targetBG":   110,
	})

	registerConstant(manager, "IDPSettingsRequest", map[string]interface{}{
		"status":       0,
		"name":         "Profile 1",
		"idpId":        1,
		"numSegments":  1,
		"maxBolus":     25000,
		"insulinDuration": 300,
	})

	registerConstant(manager, "GetSavedG7PairingCodeRequest", map[string]interface{}{
		"pairingCode": "",
	})

	registerConstant(manager, "CurrentActiveIdpValuesRequest", map[string]interface{}{
		"status":    0,
		"idpId":     1,
		"basalRate": 850,
		"carbRatio": 10000,
		"isf":       50,
		"targetBG":  110,
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
