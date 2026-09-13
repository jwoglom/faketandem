package handler

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/faults"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/reqlog"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// Router routes messages to appropriate handlers
type Router struct {
	handlers        map[string]MessageHandler
	bridge          *pumpx2.Bridge
	pumpState       *state.PumpState
	ble             bluetooth.Transport
	txManager       *protocol.TransactionManager
	settingsManager *settings.Manager
	jpakeManager    *JPAKESessionManager

	// Qualifying events notifier
	qeNotifier *QualifyingEventsNotifier

	// faultRegistry, when set, is consulted before every response leaves the
	// pump. Nil means no faults are ever injected.
	faultRegistry *faults.Registry
	// requestLog, when set, records every message in and out.
	requestLog *reqlog.Log

	// Default handler for unknown messages
	defaultHandler MessageHandler
}

// NewRouter creates a new message router
func NewRouter(bridge *pumpx2.Bridge, pumpState *state.PumpState, ble bluetooth.Transport, txManager *protocol.TransactionManager, jpakeMode, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd, pumpX2JarPath string) *Router {
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
		jpakeManager:    NewJPAKESessionManager(jpakeMode, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd, pumpX2JarPath, pumpState),
		qeNotifier:      NewQualifyingEventsNotifier(ble, pumpState),
	}

	// Register handlers
	r.registerHandlers()

	return r
}

// GetSettingsManager returns the settings manager
func (r *Router) GetSettingsManager() *settings.Manager {
	return r.settingsManager
}

// SetFaultRegistry attaches the fault injector consulted on the response path.
func (r *Router) SetFaultRegistry(registry *faults.Registry) {
	r.faultRegistry = registry
	r.jpakeManager.SetFaultRegistry(registry)
}

// SetRequestLog attaches the pump-side record of messages in and out.
func (r *Router) SetRequestLog(l *reqlog.Log) {
	r.requestLog = l
	if r.qeNotifier != nil {
		r.qeNotifier.SetRequestLog(l)
	}
}

// GetPumpState returns the pump state this router serves.
func (r *Router) GetPumpState() *state.PumpState {
	return r.pumpState
}

// GetTransport returns the transport this router sends on.
func (r *Router) GetTransport() bluetooth.Transport {
	return r.ble
}

// registerHandlers registers all message handlers
func (r *Router) registerHandlers() {
	// Core handlers
	r.RegisterHandler(NewAPIVersionHandler(r.bridge))
	r.RegisterHandler(NewTimeSinceResetHandler(r.bridge))

	// Authentication handlers
	r.RegisterHandler(NewCentralChallengeHandler(r.bridge))
	r.RegisterHandler(NewPumpChallengeHandler(r.bridge))

	// JPAKE authentication handlers. Message names/opcodes match the real protocol
	// (pumpX2's request.authentication.Jpake1aRequest etc, opcodes 32/34/36/38/40),
	// which has 5 client messages, not 4. The "round" number passed here matches
	// PumpX2JPAKEAuthenticator.ProcessRound's dispatch (pkg/handler/jpake_pumpx2.go):
	// Jpake1aRequest and Jpake1bRequest both map to round 1 because
	// processRound1 handles both internally via its own state (the first call
	// sends Jpake1a and reads JPAKE_1B, the second sends Jpake1b and reads
	// JPAKE_2) -- they are two separate wire messages, not two rounds.
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "Jpake1aRequest", 1))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "Jpake1bRequest", 1))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "Jpake2Request", 2))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "Jpake3SessionKeyRequest", 3))
	r.RegisterHandler(NewJPAKEHandler(r.bridge, r.jpakeManager, "Jpake4KeyConfirmationRequest", 4))

	// Status and data handlers. CurrentStatusRequest/Response was removed: no
	// such class exists anywhere in pumpX2 (confirmed via the jar's own class
	// table), so a real Tandem app can never send a message that decodes to
	// this name -- the real protocol exposes this data via separate per-topic
	// messages (InsulinStatusResponse, CurrentBasalStatusResponse,
	// CurrentBolusStatusResponse, etc), already handled elsewhere.
	r.RegisterHandler(NewHistoryLogHandler(r.bridge))
	// CreateHistoryLogRequest/Response has no corresponding class anywhere in
	// pumpX2 -- not part of the real protocol, so no handler is registered.
	r.RegisterHandler(NewHistoryLogStatusHandler(r.bridge))

	// Bolus handlers
	r.RegisterHandler(NewBolusPermissionHandler(r.bridge))
	r.RegisterHandler(NewBolusCalcDataSnapshotHandler(r.bridge))
	r.RegisterHandler(NewInitiateBolusHandler(r.bridge))
	r.RegisterHandler(NewRemoteBgEntryHandler(r.bridge))
	r.RegisterHandler(NewRemoteCarbEntryHandler(r.bridge))
	r.RegisterHandler(NewBolusPermissionReleaseHandler(r.bridge))

	// Settings handlers - use generic settings manager. ControlIQSettingsRequest,
	// TherapySettingsGlobalsRequest, and ControlIQGlobalsRequest are not
	// registered: none of them have a real pumpX2 counterpart, so a real
	// Tandem app can never send a message that decodes to these names.
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BasalIQSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "PumpGlobalsRequest", true))

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
	// AlarmStatusRequest gets a dedicated handler rather than a static one:
	// cliparser cannot construct AlarmStatusResponse at all (see
	// alarm_status.go), so this one is built natively from PumpState.
	r.RegisterHandler(NewAlarmStatusHandler())
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "LoadStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ProfileStatusRequest", true))
	r.RegisterHandler(NewLastBolusStatusHandler(r.bridge, "LastBolusStatusV2Request"))

	// Notification/alarm/malfunction handlers
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "HighestAamRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ActiveAamBitsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "MalfunctionStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ReminderStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMAlertStatusRequest", true))

	// ControlIQ info and sleep schedule handlers
	r.RegisterHandler(NewControlIQInfoHandler(r.bridge, "ControlIQInfoV1Request"))
	r.RegisterHandler(NewControlIQInfoHandler(r.bridge, "ControlIQInfoV2Request"))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ControlIQSleepScheduleRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BasalIQStatusRequest", true))
	r.RegisterHandler(NewControlIQIOBHandler(r.bridge, "NonControlIQIOBRequest"))

	// Bolus and basal handlers
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ExtendedBolusStatusRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "ExtendedBolusStatusV2Request", true))
	r.RegisterHandler(NewLastBolusStatusHandler(r.bridge, "LastBolusStatusV3Request"))
	r.RegisterHandler(NewTempRateHandler(r.bridge))
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

	// Bolus cancel handler (used by controlX2)
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
	// NOTE: SetSleepScheduleResponse has both an (int status) and a (byte[]
	// raw) single-arg constructor of the same arity; cliparser currently
	// resolves this to the int ctor via JVM reflection order, but that
	// ordering isn't JLS-guaranteed, so this is fragile.
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
	r.RegisterHandler(NewLastBolusStatusHandler(r.bridge, "LastBolusStatusRequest"))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMHardwareInfoRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMGlucoseAlertSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMOORAlertSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CGMRateAlertSettingsRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "BasalIQAlertInfoRequest", true))
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "RemindersRequest", true))
	// QuickBolusSettingsRequest is not registered: no "QuickBolusSettingsResponse"
	// class exists anywhere in pumpX2. The only quick-bolus message pumpX2 has
	// is the write pair SetQuickBolusSettingsRequest/Response (registered
	// above via SettingsWriteHandler), not a "get".
	r.RegisterHandler(NewGenericSettingsHandler(r.bridge, r.settingsManager, "CgmSupportPackageStatusRequest", true))

	// ProfileBasalRequest/Response has no corresponding class anywhere in
	// pumpX2 -- not part of the real protocol, so no handler is registered.
	// (Basal profile data is exposed via the real ProfileStatusRequest, above.)

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

	if r.requestLog != nil {
		r.requestLog.RecordRequest(charType.String(), msg.MessageType, msg.Opcode, msg.TxID, msg.Cargo, msg.RawPacketsHex)
	}

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

// sendResponse sends a handler response.
//
// State changes are applied BEFORE anything goes on the wire. That ordering is
// deliberate and is what makes the "applied but the response was lost" case
// reachable: a real pump that acts on a request and then loses the reply
// leaves the driver believing nothing happened while the pump has already
// moved, and a drop_response fault has to reproduce exactly that, not "nothing
// happened at all". It also means a fault can never suppress a state change.
func (r *Router) sendResponse(requestCharType bluetooth.CharacteristicType, response *Response) error {
	// Determine characteristic to use
	charType := response.Characteristic
	if charType == 0 {
		// Default to same as request
		charType = requestCharType
	}

	// Apply state changes first -- see the ordering note above.
	for _, change := range response.StateChanges {
		r.applyStateChange(change)
	}

	// Send main response if present
	if response.ResponseMessage != nil {
		if err := r.sendWithFaults(charType, response.ResponseMessage); err != nil {
			return fmt.Errorf("failed to send main response: %w", err)
		}
	}

	// Send a natively-encoded response, if the handler built one. It carries
	// its own characteristic (pkg/protocol pins each message to the
	// characteristic pumpX2 declares for it), so charType is not consulted.
	if response.NativeResponse != nil {
		if err := r.SendNative(response.NativeResponse); err != nil {
			return fmt.Errorf("failed to send native response: %w", err)
		}
	}

	// Send notifications
	for _, notification := range response.Notifications {
		if err := r.sendWithFaults(notification.Characteristic, notification.Message); err != nil {
			log.Errorf("Failed to send notification on %s: %v", notification.Characteristic, err)
			// Continue with other notifications
		}
	}

	// Send natively-encoded follow-up notifications (history log streams).
	for _, native := range response.NativeNotifications {
		if err := r.SendNative(native); err != nil {
			log.Errorf("Failed to send native notification %s: %v", native.MessageType, err)
			// Continue with the rest of the stream.
		}
	}

	return nil
}

// resign rebuilds the signed trailer of a response the pumpX2 cliparser
// encoded, with this pump's real authentication key and time since reset.
//
// It is needed because the cliparser subprocess has neither. Its signing
// inputs arrive only through the environment, and it uses the raw ASCII bytes
// of PUMP_AUTHENTICATION_KEY as the HMAC key rather than decoding it -- so a
// binary JPAKE-derived session key cannot be handed to it at all, and with no
// key set it signs every signed message with the ASCII of its own
// "IGNORE_HMAC_SIGNATURE_EXCEPTION" placeholder. The cargo cliparser produced
// is kept verbatim; only the last 24 payload bytes and the CRC are recomputed,
// so this is a re-signing step and not a second encoder.
//
// Messages the catalog does not mark signed are returned untouched, byte for
// byte. So is a signed message the pump cannot sign yet: before
// authentication there is no session key, and every signed message is on a
// characteristic that requires authentication anyway, so this only arises for
// a handler answering out of turn -- worth a warning, not worth inventing a
// key for.
func (r *Router) resign(msg *pumpx2.EncodedMessage) *pumpx2.EncodedMessage {
	if msg == nil || !protocol.IsSignedMessage(msg.MessageType) {
		return msg
	}

	authKey := r.pumpState.GetAuthKey()
	if len(authKey) == 0 {
		log.Warnf("Cannot re-sign %s txID=%d: the pump has no authentication key; "+
			"sending the cliparser signature, which no driver will accept", msg.MessageType, msg.TxID)
		return msg
	}

	timeSinceReset := r.pumpState.GetTimeSinceReset()
	packets, err := protocol.ResignFragmentsHex(msg.Packets, authKey, timeSinceReset)
	if err != nil {
		log.Errorf("Cannot re-sign %s txID=%d: %v; sending it as cliparser encoded it",
			msg.MessageType, msg.TxID, err)
		return msg
	}

	log.Debugf("Re-signed %s txID=%d with the pump's own key (timeSinceReset=%d)",
		msg.MessageType, msg.TxID, timeSinceReset)

	resigned := *msg
	resigned.Packets = packets
	return &resigned
}

// chunkPayload is the per-fragment payload size responses are framed to, from
// the ATT MTU the transport reports. A transport that cannot report one (the
// Linux GATT transport, which lets gatt negotiate the MTU with the central and
// does not expose the result) gets the 23-byte-MTU default, which is the
// 18-byte chunking the emulator has always used.
func (r *Router) chunkPayload() int {
	if negotiator, ok := r.ble.(bluetooth.MTUNegotiator); ok {
		return protocol.ChunkPayloadForMTU(negotiator.ATTMTU())
	}
	return protocol.DefaultMaxChunkPayload
}

// refragment re-frames a response for the link's negotiated ATT MTU.
//
// The pumpX2 cliparser always emits 18-byte payload chunks, the most a 23-byte
// ATT MTU allows, and so does the native encoder by default. A real Mobi paired
// with the official iOS app negotiates a much larger MTU, so most responses
// reach the phone as a single notification. Both receivers read the
// remaining-fragment counter and never a fragment's length, so the larger
// framing is the same message in fewer packets -- a one-packet response simply
// arrives with remaining=0.
//
// At the default MTU this is a no-op and the bytes are untouched. It runs after
// re-signing, so the signature covers the message body rather than any
// particular framing of it, and before the fault path, so the request log and
// a partial-response fault both count the fragments that actually go out.
func (r *Router) refragment(msg *pumpx2.EncodedMessage) *pumpx2.EncodedMessage {
	chunkPayload := r.chunkPayload()
	if msg == nil || len(msg.Packets) == 0 || chunkPayload == protocol.DefaultMaxChunkPayload {
		return msg
	}

	packets, err := protocol.RefragmentHex(msg.Packets, uint8(msg.TxID), chunkPayload)
	if err != nil {
		log.Errorf("Cannot re-fragment %s txID=%d for a %d-byte payload chunk: %v; sending it as encoded",
			msg.MessageType, msg.TxID, chunkPayload, err)
		return msg
	}

	log.Debugf("Re-fragmented %s txID=%d from %d to %d fragment(s) for a %d-byte payload chunk",
		msg.MessageType, msg.TxID, len(msg.Packets), len(packets), chunkPayload)

	refragmented := *msg
	refragmented.Packets = packets
	return &refragmented
}

// sendWithFaults applies any armed fault to one outgoing message and then
// sends whatever is left to send.
func (r *Router) sendWithFaults(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage) error {
	msg = r.refragment(r.resign(msg))

	fault := r.matchResponseFault(charType, msg)
	if fault == nil {
		return r.sendAndRecord(charType, msg, -1, "", "")
	}

	switch fault.Kind {
	case faults.KindDropResponse:
		log.Warnf("Fault %d (drop_response): suppressing %s txID=%d; state changes were already applied",
			fault.ID, msg.MessageType, msg.TxID)
		r.recordSuppressed(charType, msg, fault.Kind, "response suppressed after state was applied")
		return nil

	case faults.KindDelayResponse:
		delay := fault.Delay()
		log.Warnf("Fault %d (delay_response): holding %s txID=%d for %v", fault.ID, msg.MessageType, msg.TxID, delay)
		time.Sleep(delay)
		return r.sendAndRecord(charType, msg, -1, fault.Kind, fmt.Sprintf("delayed %v", delay))

	case faults.KindErrorResponse:
		return r.sendErrorResponse(charType, msg, fault)

	case faults.KindDisconnect:
		return r.disconnectFault(charType, msg, fault)

	default:
		log.Warnf("Fault %d has kind %q, which the response path does not apply", fault.ID, fault.Kind)
		return r.sendAndRecord(charType, msg, -1, "", "")
	}
}

// matchResponseFault consumes the fault (if any) armed for this message.
func (r *Router) matchResponseFault(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage) *faults.Fault {
	if r.faultRegistry == nil {
		return nil
	}
	return r.faultRegistry.Match(faults.Target{
		Opcode:         msg.Opcode,
		Message:        msg.MessageType,
		Characteristic: charType.String(),
	},
		faults.KindDropResponse,
		faults.KindDelayResponse,
		faults.KindErrorResponse,
		faults.KindDisconnect,
	)
}

// sendErrorResponse replaces a response with a protocol ErrorResponse, via the
// ErrorResponseEncoder hook. With no encoder installed the fault degrades to a
// drop and says so, rather than pretending an error path was exercised.
func (r *Router) sendErrorResponse(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage, fault *faults.Fault) error {
	if ErrorResponseEncoder == nil {
		log.Warnf("Fault %d (error_response): no ErrorResponseEncoder installed; dropping %s txID=%d instead",
			fault.ID, msg.MessageType, msg.TxID)
		r.recordSuppressed(charType, msg, fault.Kind, "no ErrorResponseEncoder installed; response dropped instead")
		return nil
	}

	errMsg, err := ErrorResponseEncoder(msg.TxID, fault.ErrorCode, msg.Opcode, msg.MessageType)
	if err != nil {
		log.Errorf("Fault %d (error_response): encoder failed for %s: %v", fault.ID, msg.MessageType, err)
		r.recordSuppressed(charType, msg, fault.Kind, fmt.Sprintf("ErrorResponseEncoder failed: %v", err))
		return nil
	}

	// A real pump answers on the characteristic the message belongs to, not on
	// whatever characteristic the rejected request arrived on: ErrorResponse is
	// a CURRENT_STATUS message, and that is the only place the driver looks for
	// it (TandemPeripheralManager's CURRENT_STATUS/opcode-77 branch). An
	// encoder that names no characteristic keeps the request's.
	errCharType := charType
	if named, ok := bluetooth.CharacteristicTypeFromName(errMsg.Characteristic); ok {
		errCharType = named
	}

	log.Warnf("Fault %d (error_response): answering %s txID=%d with ErrorResponse code %d on %s",
		fault.ID, msg.MessageType, msg.TxID, fault.ErrorCode, errCharType)
	return r.sendAndRecord(errCharType, errMsg, -1, fault.Kind,
		fmt.Sprintf("ErrorResponse code %d in place of %s", fault.ErrorCode, msg.MessageType))
}

// disconnectFault drops the link either instead of answering at all, or
// partway through the response's fragments.
func (r *Router) disconnectFault(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage, fault *faults.Fault) error {
	if fault.After == faults.AfterPartialResponse {
		limit := fault.FragmentsSent
		log.Warnf("Fault %d (disconnect): sending %d of %d fragments of %s txID=%d, then dropping the link",
			fault.ID, limit, len(msg.Packets), msg.MessageType, msg.TxID)
		err := r.sendAndRecord(charType, msg, limit, fault.Kind,
			fmt.Sprintf("link dropped after %d of %d fragments", limit, len(msg.Packets)))
		r.ble.ShutdownConnection()
		return err
	}

	log.Warnf("Fault %d (disconnect): dropping the link instead of answering %s txID=%d",
		fault.ID, msg.MessageType, msg.TxID)
	r.recordSuppressed(charType, msg, fault.Kind, "link dropped instead of answering")
	r.ble.ShutdownConnection()
	return nil
}

// recordSuppressed logs a response that was encoded but never sent.
func (r *Router) recordSuppressed(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage, fault, note string) {
	if r.requestLog == nil {
		return
	}
	r.requestLog.RecordResponse(charType.String(), msg.MessageType, msg.Opcode, msg.TxID, msg.Packets, 0, fault, note)
}

// sendAndRecord sends up to limit fragments (-1 for all) and records the
// result in the request log.
func (r *Router) sendAndRecord(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage, limit int, fault, note string) error {
	sent, err := r.sendMessage(charType, msg, limit)
	if r.requestLog != nil {
		if err != nil {
			if note != "" {
				note += "; "
			}
			note += "send error: " + err.Error()
		}
		r.requestLog.RecordResponse(charType.String(), msg.MessageType, msg.Opcode, msg.TxID, msg.Packets, sent, fault, note)
	}
	return err
}

// sendMessage sends an encoded message on a characteristic, stopping after
// limit fragments when limit is non-negative. It returns how many fragments
// actually went out.
func (r *Router) sendMessage(charType bluetooth.CharacteristicType, msg *pumpx2.EncodedMessage, limit int) (int, error) {
	log.Infof("Sending %s on %s: txID=%d, %d packet(s)",
		msg.MessageType, charType, msg.TxID, len(msg.Packets))

	sent := 0
	for i, packetHex := range msg.Packets {
		if limit >= 0 && sent >= limit {
			break
		}

		packetData, err := hex.DecodeString(packetHex)
		if err != nil {
			return sent, fmt.Errorf("failed to decode packet %d: %w", i, err)
		}

		protocol.LogPacket("TX", charType, packetData)

		// Send via notification
		if err := r.ble.Notify(charType, packetData); err != nil {
			return sent, fmt.Errorf("failed to send packet %d: %w", i, err)
		}
		sent++

		log.Tracef("Sent packet %d/%d: %s", i+1, len(msg.Packets), packetHex)
	}

	return sent, nil
}

// SendNative notifies a message built by the Go encoder in pkg/protocol,
// bypassing the pumpX2 cliparser entirely. This is the escape hatch for
// messages cliparser cannot encode and for fault injection that needs to put
// specific bytes on a characteristic.
func (r *Router) SendNative(msg *protocol.NativeMessage) error {
	if msg == nil {
		return nil
	}

	fragments := r.refragmentNative(msg)

	log.Infof("Sending native %s on %s: txID=%d, %d packet(s)",
		msg.MessageType, msg.Characteristic, msg.TxID, len(fragments))

	for i, fragment := range fragments {
		protocol.LogPacket("TX", msg.Characteristic, fragment)
		if err := r.ble.Notify(msg.Characteristic, fragment); err != nil {
			return fmt.Errorf("failed to send %s fragment %d: %w", msg.MessageType, i, err)
		}
	}

	return nil
}

// refragmentNative re-frames a natively-encoded message for the link's ATT
// MTU, the same way refragment does for a cliparser-encoded one.
func (r *Router) refragmentNative(msg *protocol.NativeMessage) [][]byte {
	chunkPayload := r.chunkPayload()
	if chunkPayload == protocol.DefaultMaxChunkPayload || len(msg.Fragments) == 0 {
		return msg.Fragments
	}

	body, err := protocol.MessageBodyFromFragmentsHex(msg.PacketsHex())
	if err == nil {
		var fragments [][]byte
		if fragments, err = protocol.FragmentMessage(body, msg.TxID, chunkPayload); err == nil {
			log.Debugf("Re-fragmented native %s txID=%d from %d to %d fragment(s) for a %d-byte payload chunk",
				msg.MessageType, msg.TxID, len(msg.Fragments), len(fragments), chunkPayload)
			return fragments
		}
	}

	log.Errorf("Cannot re-fragment native %s txID=%d for a %d-byte payload chunk: %v; sending it as encoded",
		msg.MessageType, msg.TxID, chunkPayload, err)
	return msg.Fragments
}

// SendErrorResponse sends the ErrorResponse a real pump returns in place of the
// response to a request it rejects: opcode 77 with a two-byte cargo of the
// rejected request's opcode and a pumpX2 error code.
//
// charType is the characteristic to notify on; pass bluetooth.CharCurrentStatus
// for the normal case, which is where the driver looks for it. The response is
// left unsigned so it fits one fragment, matching how the driver reads it.
//
// Note that TandemKit only logs an ErrorResponse and does not release the
// pending write it answers, so injecting one makes the driver stall until its
// 30-second timeout rather than fail fast. That is a driver conformance gap,
// not a bug in this path.
func (r *Router) SendErrorResponse(charType bluetooth.CharacteristicType, txID, requestOpcode, errorCode uint8) error {
	msg, err := protocol.BuildErrorResponse(txID, requestOpcode, errorCode, protocol.ErrorResponseOptions{})
	if err != nil {
		return fmt.Errorf("failed to build ErrorResponse: %w", err)
	}
	msg.Characteristic = charType

	log.Warnf("Sending ErrorResponse on %s: txID=%d, requestOpcode=%d, errorCode=%d",
		charType, txID, requestOpcode, errorCode)

	return r.SendNative(msg)
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
		r.pumpState.StartBolusWithSource(
			bolusState.UnitsTotal, bolusState.BolusID, bolusState.SourceID, bolusState.TypeBitmask)
		r.pumpState.AppendHistory(state.HistoryEvent{
			TypeID: state.HistoryBolusActivated,
			Name:   "BolusActivated",
			Fields: map[string]interface{}{
				"bolusId":     bolusState.BolusID,
				"selectedIob": 0,
				"iob":         r.pumpState.GetIOB(),
				"bolusSize":   bolusState.UnitsTotal,
			},
		})
		if r.qeNotifier != nil {
			if err := r.qeNotifier.NotifyBolusStart(bolusState.BolusID, bolusState.UnitsTotal); err != nil {
				log.Warnf("Failed to notify bolus start: %v", err)
			}
		}
		return
	}
	currentBolus := *r.pumpState.Bolus
	r.pumpState.StopBolus()
	if !currentBolus.Active {
		return
	}

	// A canceled bolus is still a finished bolus: record it so the driver's
	// next LastBolusStatus query reports the same bolusId it commanded, along
	// with how much actually went in before the cancel.
	r.pumpState.RecordLastBolus(state.LastBolusRecord{
		BolusID:        currentBolus.BolusID,
		RequestedUnits: currentBolus.UnitsTotal,
		DeliveredUnits: currentBolus.UnitsDelivered,
		SourceID:       currentBolus.SourceID,
		TypeBitmask:    currentBolus.TypeBitmask,
		EndReasonID:    state.BolusEndReasonStopped,
	})

	if r.qeNotifier != nil {
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
	if basalState.TempBasalActive {
		// The record's own fields are the commanded percentage, the duration in
		// milliseconds and the temp rate id -- the same three the driver reads
		// back out of TempRateActivatedHistoryLog.
		r.pumpState.AppendHistory(state.HistoryEvent{
			TypeID: state.HistoryTempRateActivated,
			Name:   "TempRateActivated",
			Fields: map[string]interface{}{
				"percent":              basalState.TempBasalPercent,
				"durationMilliseconds": basalState.TempBasalEnd.Sub(basalState.TempBasalStart).Seconds() * 1000,
				"tempRateId":           basalState.TempRateID,
			},
		})
	}
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

	// Both records carry the reservoir's remaining whole units at the moment of
	// the change: on real pumps this field tracks
	// InsulinStatusResponse.currentInsulinAmount to the unit, and it used to be
	// emitted as an empty payload here.
	insulinAmount := int(r.pumpState.GetReservoirLevel())
	if suspended {
		r.pumpState.AppendHistory(state.HistoryEvent{
			TypeID: state.HistoryPumpingSuspended,
			Name:   "PumpingSuspended",
			Fields: map[string]interface{}{
				"preSuspendState": 106,
				"insulinAmount":   insulinAmount,
				"reasonId":        0, // user-aborted; a pump-raised suspend uses 1
				"rpaTimeout":      15,
			},
		})
	} else {
		r.pumpState.AppendHistory(state.HistoryEvent{
			TypeID: state.HistoryPumpingResumed,
			Name:   "PumpingResumed",
			Fields: map[string]interface{}{
				"preResumeState": 100,
				"insulinAmount":  insulinAmount,
			},
		})
	}
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

// ResetJPAKESession clears any in-progress JPAKE authenticator. Call this on
// BLE disconnect so a stale/broken authenticator (e.g. one whose pumpX2
// subprocess died mid-handshake) is never reused by the next connection.
func (r *Router) ResetJPAKESession() {
	r.jpakeManager.RemoveAll()
}

// GetStats returns router statistics
func (r *Router) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"registeredHandlers": len(r.handlers),
		"authenticated":      r.pumpState.IsAuthenticated,
	}
}
