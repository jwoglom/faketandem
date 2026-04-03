package handler

import (
	"encoding/hex"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// Router routes messages to appropriate handlers
type Router struct {
	handlers        map[string]MessageHandler
	bridge          *pumpx2.Bridge
	pumpState       *state.PumpState
	ble             *bluetooth.Ble
	txManager       *protocol.TransactionManager
	settingsManager *settings.Manager
	jpakeManager    *JPAKESessionManager

	// Qualifying events notifier
	qeNotifier *QualifyingEventsNotifier

	// Default handler for unknown messages
	defaultHandler MessageHandler
}

// NewRouter creates a new message router
func NewRouter(bridge *pumpx2.Bridge, pumpState *state.PumpState, ble *bluetooth.Ble, txManager *protocol.TransactionManager, jpakeMode, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd string) *Router {
	// Create and initialize settings manager
	settingsManager := settings.NewManager()
	settings.RegisterDefaults(settingsManager)

	r := &Router{
		handlers:        make(map[string]MessageHandler),
		bridge:          bridge,
		pumpState:       pumpState,
		ble:             ble,
		txManager:       txManager,
		settingsManager: settingsManager,
		jpakeManager:    NewJPAKESessionManager(jpakeMode, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd),
		qeNotifier:      NewQualifyingEventsNotifier(bridge, ble, pumpState),
	}

	// Register handlers
	r.registerHandlers()

	return r
}

// GetSettingsManager returns the settings manager
func (r *Router) GetSettingsManager() *settings.Manager {
	return r.settingsManager
}

// registerHandlers registers all message handlers
func (r *Router) registerHandlers() {
	// Core handlers
	r.RegisterHandler(NewAPIVersionHandler(r.bridge))
	r.RegisterHandler(NewTimeSinceResetHandler(r.bridge))

	// Authentication handlers
	r.RegisterHandler(NewCentralChallengeHandler(r.bridge))
	r.RegisterHandler(NewPumpChallengeHandler(r.bridge))

	// JPAKE authentication handlers (rounds 1-4)
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "JPAKERound1Request", 1))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "JPAKERound2Request", 2))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "JPAKERound3Request", 3))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "JPAKERound4Request", 4))

	// Status and data handlers
	r.RegisterHandler(NewCurrentStatusHandler(r.bridge))
	r.RegisterHandler(NewHistoryLogHandler(r.bridge))
	r.RegisterHandler(NewCreateHistoryLogHandler(r.bridge))
	r.RegisterHandler(NewHistoryLogStatusHandler(r.bridge))

	// Bolus handlers
	r.RegisterHandler(NewBolusPermissionHandler(r.bridge))
	r.RegisterHandler(NewBolusCalcDataSnapshotHandler(r.bridge))
	r.RegisterHandler(NewInitiateBolusHandler(r.bridge))
	r.RegisterHandler(NewBolusTerminationHandler(r.bridge))
	r.RegisterHandler(NewRemoteBgEntryHandler(r.bridge))
	r.RegisterHandler(NewRemoteCarbEntryHandler(r.bridge))
	r.RegisterHandler(NewBolusPermissionReleaseHandler(r.bridge))

	// Settings handlers - use generic settings manager
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BasalIQSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ControlIQSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "PumpGlobalsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "TherapySettingsGlobalsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ControlIQGlobalsRequest", true))

	// Dynamic polling status handlers (controlX2 polls every 5 min)
	r.RegisterHandler(NewCurrentBatteryHandler(r.bridge, "V2"))
	r.RegisterHandler(NewCurrentBatteryHandler(r.bridge, "V1"))
	r.RegisterHandler(NewControlIQIOBHandler(r.bridge, "ControlIQIOBRequest"))
	r.RegisterHandler(NewInsulinStatusHandler(r.bridge))

	// Dynamic qualifying event status handlers
	r.RegisterHandler(NewCurrentBasalStatusHandler(r.bridge))
	r.RegisterHandler(NewCurrentBolusStatusHandler(r.bridge))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CurrentEGVGuiDataRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "HomeScreenMirrorRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "AlertStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "AlarmStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "LoadStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ProfileStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "LastBolusStatusV2Request", true))

	// Notification/alarm/malfunction handlers
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "HighestAamRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ActiveAamBitsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "MalfunctionStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ReminderStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMAlertStatusRequest", true))

	// ControlIQ info and sleep schedule handlers
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ControlIQInfoV1Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ControlIQInfoV2Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ControlIQSleepScheduleRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BasalIQStatusRequest", true))
	r.RegisterHandler(NewControlIQIOBHandler(r.bridge, "NonControlIQIOBRequest"))

	// Bolus and basal handlers
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ExtendedBolusStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ExtendedBolusStatusV2Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "LastBolusStatusV3Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "TempRateRequest", true))
	r.RegisterHandler(NewTempRateStatusHandler(r.bridge))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "LastBGRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BolusPermissionChangeReasonRequest", true))

	// Global pump settings handlers
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "GlobalMaxBolusSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BasalLimitSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "LocalizationRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "PumpSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "SendTipsControlGenericTestRequest", true))

	// Pump info handlers
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "PumpFeaturesV2Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "PumpVersionRequest", false))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BleSoftwareInfoRequest", false))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CommonSoftwareInfoRequest", false))

	// Control handlers
	r.RegisterHandler(NewSetSensorTypeHandler(r.bridge))
	r.RegisterHandler(NewStreamDataReadinessHandler(r.bridge))
	r.RegisterHandler(NewFactoryResetBHandler(r.bridge))

	// Bolus cancel handler (used by controlX2 instead of BolusTermination)
	r.RegisterHandler(NewCancelBolusHandler(r.bridge))

	// Pump suspend/resume handlers
	r.RegisterHandler(NewSuspendPumpingHandler(r.bridge))
	r.RegisterHandler(NewResumePumpingHandler(r.bridge))

	// Temp rate control handlers
	r.RegisterHandler(NewSetTempRateHandler(r.bridge))
	r.RegisterHandler(NewStopTempRateHandler(r.bridge))

	// Cartridge change flow handlers
	r.RegisterHandler(NewCartridgeHandler(r.bridge, "EnterChangeCartridgeModeRequest"))
	r.RegisterHandler(NewCartridgeHandler(r.bridge, "ExitChangeCartridgeModeRequest"))
	r.RegisterHandler(NewCartridgeHandler(r.bridge, "EnterFillTubingModeRequest"))
	r.RegisterHandler(NewCartridgeHandler(r.bridge, "ExitFillTubingModeRequest"))
	r.RegisterHandler(NewCartridgeHandler(r.bridge, "FillCannulaRequest"))

	// Simple control handlers (log and return success)
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "DismissNotificationRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "PlaySoundRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "ChangeTimeDateRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "DisconnectPumpRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "UserInteractionRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "StreamDataPreflightRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "ActivateShelfModeRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "PrimeTubingSuspendRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "FactoryResetRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "AdditionalBolusRequest"))

	// Settings write handlers (write value, update readback)
	r.RegisterHandler(NewSetModesHandler(r.bridge))
	r.RegisterHandler(NewSettingsWriteHandler(r.bridge, r.settingsManager, "ChangeControlIQSettingsRequest", "ControlIQSettingsRequest"))
	r.RegisterHandler(NewSettingsWriteHandler(r.bridge, r.settingsManager, "SetMaxBolusLimitRequest", "GlobalMaxBolusSettingsRequest"))
	r.RegisterHandler(NewSettingsWriteHandler(r.bridge, r.settingsManager, "SetMaxBasalLimitRequest", "BasalLimitSettingsRequest"))
	r.RegisterHandler(NewSettingsWriteHandler(r.bridge, r.settingsManager, "SetSleepScheduleRequest", "ControlIQSleepScheduleRequest"))
	r.RegisterHandler(NewSettingsWriteHandler(r.bridge, r.settingsManager, "SetQuickBolusSettingsRequest", ""))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetPumpSoundsRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetPumpAlertSnoozeRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetAutoOffAlertRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetBgReminderRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetSiteChangeReminderRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetMissedMealBolusReminderRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetLowInsulinAlertRequest"))

	// IDP management handlers
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "CreateIDPRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetIDPSettingsRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetIDPSegmentRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetActiveIDPRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "DeleteIDPRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "RenameIDPRequest"))

	// CGM configuration handlers
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetDexcomG7PairingCodeRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "SetG6TransmitterIdRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "StartDexcomG6SensorSessionRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "StopDexcomCGMSensorSessionRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "CgmHighLowAlertRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "CgmRiseFallAlertRequest"))
	r.RegisterHandler(NewSimpleControlHandler(r.bridge, "CgmOutOfRangeAlertRequest"))

	// Additional status handlers used by controlX2
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "IDPSegmentRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "IDPSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "GetSavedG7PairingCodeRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CurrentActiveIdpValuesRequest", true))

	// Phase 5: Missing status query variants
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "PumpFeaturesV1Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "PumpVersionBRequest", false))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CgmStatusV2Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CurrentEgvGuiDataV2Request", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "LastBolusStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMHardwareInfoRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMGlucoseAlertSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMOORAlertSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMRateAlertSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BasalIQAlertInfoRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "RemindersRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "QuickBolusSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CgmSupportPackageStatusRequest", true))

	// Keep ProfileBasalHandler as custom since it uses pump state
	r.RegisterHandler(NewProfileBasalHandler(r.bridge))

	// Set default handler for unknown messages
	r.SetDefaultHandler(NewDefaultHandler(r.bridge))

	log.Infof("Registered %d message handlers", len(r.handlers))
}

// RegisterHandler registers a message handler
func (r *Router) RegisterHandler(handler MessageHandler) {
	messageType := handler.MessageType()
	r.handlers[messageType] = handler
	log.Debugf("Registered handler: %s (auth required: %v)", messageType, handler.RequiresAuth())
}

// SetDefaultHandler sets the default handler for unknown messages
func (r *Router) SetDefaultHandler(handler MessageHandler) {
	r.defaultHandler = handler
}

// RouteMessage routes a message to the appropriate handler
func (r *Router) RouteMessage(charType bluetooth.CharacteristicType, msg *pumpx2.ParsedMessage) error {
	log.Debugf("Routing message: type=%s, txID=%d, opcode=%d", msg.MessageType, msg.TxID, msg.Opcode)

	// Find handler
	handler, exists := r.handlers[msg.MessageType]
	if !exists {
		if r.defaultHandler != nil {
			log.Debugf("No specific handler for %s, using default handler", msg.MessageType)
			handler = r.defaultHandler
		} else {
			log.Warnf("No handler registered for message type: %s", msg.MessageType)
			return fmt.Errorf("no handler for message type: %s", msg.MessageType)
		}
	}

	// Check authentication requirement
	if handler.RequiresAuth() && !r.pumpState.IsAuthenticated {
		log.Warnf("Message %s requires authentication but pump is not authenticated", msg.MessageType)
		// TODO: Send authentication required response
		return fmt.Errorf("authentication required for %s", msg.MessageType)
	}

	// Handle the message
	response, err := handler.HandleMessage(msg, r.pumpState)
	if err != nil {
		log.Errorf("Handler error for %s: %v", msg.MessageType, err)
		return fmt.Errorf("handler error: %w", err)
	}

	// Process response
	if response != nil {
		if err := r.sendResponse(charType, response); err != nil {
			log.Errorf("Failed to send response: %v", err)
			return fmt.Errorf("failed to send response: %w", err)
		}
	}

	return nil
}

// sendResponse sends a handler response
func (r *Router) sendResponse(requestCharType bluetooth.CharacteristicType, response *Response) error {
	// Determine characteristic to use
	charType := response.Characteristic
	if charType == 0 {
		// Default to same as request
		charType = requestCharType
	}

	// Send main response if present
	if response.ResponseMessage != nil {
		if err := r.sendMessage(charType, response.ResponseMessage); err != nil {
			return fmt.Errorf("failed to send main response: %w", err)
		}
	}

	// Send notifications
	for _, notification := range response.Notifications {
		if err := r.sendMessage(notification.Characteristic, notification.Message); err != nil {
			log.Errorf("Failed to send notification on %s: %v", notification.Characteristic, err)
			// Continue with other notifications
		}
	}

	// Apply state changes
	for _, change := range response.StateChanges {
		r.applyStateChange(change)
	}

	return nil
}

// sendMessage sends an encoded message on a characteristic
func (r *Router) sendMessage(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage) error {
	log.Infof("Sending %s on %s: txID=%d, %d packet(s)",
		msg.MessageType, charType, msg.TxID, len(msg.Packets))

	for i, packetHex := range msg.Packets {
		packetData, err := hex.DecodeString(packetHex)
		if err != nil {
			return fmt.Errorf("failed to decode packet %d: %w", i, err)
		}

		protocol.LogPacket("TX", charType, packetData)

		// Send via notification
		if err := r.ble.Notify(charType, packetData); err != nil {
			return fmt.Errorf("failed to send packet %d: %w", i, err)
		}

		log.Tracef("Sent packet %d/%d: %s", i+1, len(msg.Packets), packetHex)
	}

	return nil
}

// applyStateChange applies a state change
func (r *Router) applyStateChange(change StateChange) {
	log.Debugf("Applying state change: type=%d", change.Type)

	switch change.Type {
	case StateChangeAuth:
		r.applyAuthChange(change)
	case StateChangeTime:
		r.pumpState.UpdateTimeSinceReset()
	case StateChangeBolus:
		r.applyBolusChange(change)
	case StateChangeBasal:
		r.applyBasalChange(change)
	case StateChangeReservoir:
		if level, ok := change.Data.(float64); ok {
			r.pumpState.SetReservoirLevel(level)
		}
	case StateChangeBattery:
		if pct, ok := change.Data.(int); ok {
			r.pumpState.SetBatteryLevel(pct)
		}
	case StateChangeAlert:
		r.applyAlertChange(change)
	case StateChangeSuspend:
		r.applySuspendChange(change)
	default:
		log.Warnf("Unknown state change type: %d", change.Type)
	}
}

func (r *Router) applyAuthChange(change StateChange) {
	if authKey, ok := change.Data.([]byte); ok {
		r.pumpState.SetAuthenticated(authKey)
		r.bridge.SetAuthenticationKey(hex.EncodeToString(authKey))
	}
}

func (r *Router) applyBolusChange(change StateChange) {
	bolusState, ok := change.Data.(*state.BolusState)
	if !ok {
		return
	}
	if bolusState.Active {
		r.pumpState.StartBolus(bolusState.UnitsTotal, bolusState.BolusID)
		if r.qeNotifier != nil {
			if err := r.qeNotifier.NotifyBolusStart(bolusState.BolusID, bolusState.UnitsTotal); err != nil {
				log.Warnf("Failed to notify bolus start: %v", err)
			}
		}
		return
	}
	currentBolus := r.pumpState.Bolus
	r.pumpState.StopBolus()
	if r.qeNotifier != nil && currentBolus.Active {
		if err := r.qeNotifier.NotifyBolusCanceled(
			currentBolus.BolusID, currentBolus.UnitsDelivered, currentBolus.UnitsTotal,
		); err != nil {
			log.Warnf("Failed to notify bolus canceled: %v", err)
		}
	}
}

func (r *Router) applyBasalChange(change StateChange) {
	basalState, ok := change.Data.(*state.BasalState)
	if !ok {
		return
	}
	oldRate := r.pumpState.GetBasalRate()
	r.pumpState.SetBasalState(basalState)
	newRate := r.pumpState.GetBasalRate()
	if r.qeNotifier != nil {
		if err := r.qeNotifier.NotifyBasalRateChange(oldRate, newRate, basalState.TempBasalActive); err != nil {
			log.Warnf("Failed to notify basal rate change: %v", err)
		}
	}
}

func (r *Router) applyAlertChange(change StateChange) {
	if alert, ok := change.Data.(state.Alert); ok {
		r.pumpState.AddAlert(alert)
		if r.qeNotifier != nil {
			if err := r.qeNotifier.NotifyAlert(alert); err != nil {
				log.Warnf("Failed to notify alert: %v", err)
			}
		}
	}
}

func (r *Router) applySuspendChange(change StateChange) {
	suspended, ok := change.Data.(bool)
	if !ok {
		return
	}
	r.pumpState.SetPumpingSuspended(suspended)
	if r.qeNotifier == nil {
		return
	}
	if suspended {
		if err := r.qeNotifier.NotifyPumpSuspended("user"); err != nil {
			log.Warnf("Failed to notify pump suspended: %v", err)
		}
	} else {
		if err := r.qeNotifier.NotifyPumpResumed(); err != nil {
			log.Warnf("Failed to notify pump resumed: %v", err)
		}
	}
}

// GetQualifyingEventsNotifier returns the qualifying events notifier
func (r *Router) GetQualifyingEventsNotifier() *QualifyingEventsNotifier {
	return r.qeNotifier
}

// GetStats returns router statistics
func (r *Router) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"registeredHandlers": len(r.handlers),
		"authenticated":      r.pumpState.IsAuthenticated,
	}
}
