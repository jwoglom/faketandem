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

	// Qualifying events notifier
	qeNotifier *QualifyingEventsNotifier

	// Default handler for unknown messages
	defaultHandler MessageHandler
}

// NewRouter creates a new message router
func NewRouter(bridge *pumpx2.Bridge, pumpState *state.PumpState, ble *bluetooth.Ble, txManager *protocol.TransactionManager) *Router {
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
	r.RegisterHandler(NewApiVersionHandler(r.bridge))
	r.RegisterHandler(NewTimeSinceResetHandler(r.bridge))

	// Authentication handlers
	r.RegisterHandler(NewCentralChallengeHandler(r.bridge))
	r.RegisterHandler(NewPumpChallengeHandler(r.bridge))

	// JPAKE authentication handlers (rounds 1-4)
	// Note: The exact message types may vary - these are placeholders
	r.RegisterHandler(NewJPAKEHandler(r.bridge, "JPAKERound1Request", 1))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, "JPAKERound2Request", 2))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, "JPAKERound3Request", 3))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, "JPAKERound4Request", 4))

	// Status and data handlers
	r.RegisterHandler(NewCurrentStatusHandler(r.bridge))
	r.RegisterHandler(NewHistoryLogHandler(r.bridge))

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
func (r *Router) sendResponse(requestCharType bluetooth.CharacteristicType, response *HandlerResponse) error {
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
		if authKey, ok := change.Data.([]byte); ok {
			r.pumpState.SetAuthenticated(authKey)
			// Update bridge with auth key
			r.bridge.SetAuthenticationKey(hex.EncodeToString(authKey))
		}

	case StateChangeTime:
		r.pumpState.UpdateTimeSinceReset()

	case StateChangeBolus:
		if bolusState, ok := change.Data.(*state.BolusState); ok {
			if bolusState.Active {
				r.pumpState.StartBolus(bolusState.UnitsTotal, bolusState.BolusID)
				// Notify bolus start
				if r.qeNotifier != nil {
					r.qeNotifier.NotifyBolusStart(bolusState.BolusID, bolusState.UnitsTotal)
				}
			} else {
				// Get current bolus info before stopping
				currentBolus := r.pumpState.Bolus
				r.pumpState.StopBolus()
				// Notify bolus canceled
				if r.qeNotifier != nil && currentBolus.Active {
					r.qeNotifier.NotifyBolusCanceled(
						currentBolus.BolusID,
						currentBolus.UnitsDelivered,
						currentBolus.UnitsTotal,
					)
				}
			}
		}

	// TODO: Implement other state changes (basal, reservoir, battery, alerts)
	default:
		log.Warnf("Unknown state change type: %d", change.Type)
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
