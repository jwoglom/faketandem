package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/jwoglom/faketandem/pkg/api"
	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/config"
	"github.com/jwoglom/faketandem/pkg/faults"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/harness"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/reqlog"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// Transport names accepted by the -transport flag.
const (
	transportBle     = "ble"
	transportVirtual = "virtual"
)

func main() {
	// if both verbose and quiet are chosen, e.g., -v -q, the verbose dominates
	var traceLevel = flag.Bool("v", false, "verbose off by default, TraceLevel")
	var infoLevel = flag.Bool("q", false, "quiet off by default, InfoLevel")
	var messageCodec = flag.String("message-codec", config.MessageCodecNative, "how pump messages are parsed and encoded: 'native' (Go implementations of pumpX2's messages, no JVM), 'pumpx2' (pumpX2's cliparser) or 'compare' (answer in Go, also run cliparser on every message and log any difference)")
	var pumpX2Path = flag.String("pumpx2-path", "", "path to a pumpX2 checkout (or PUMPX2_PATH); used only by -message-codec pumpx2/compare and -jpake-mode pumpx2, and not needed with -pumpx2-jar-path")
	var pumpX2Mode = flag.String("pumpx2-mode", "gradle", "how to run cliparser from -pumpx2-path: 'gradle' or 'jar'")
	var pumpX2JarPath = flag.String("pumpx2-jar-path", "", "path to a prebuilt cliparser jar for -message-codec pumpx2/compare and -jpake-mode pumpx2; skips gradle entirely and implies -pumpx2-mode=jar")
	var jpakeMode = flag.String("jpake-mode", config.JPAKEModeGo, "JPAKE pairing: 'go' (native EC-JPAKE, a port of pumpX2's) or 'pumpx2' (pumpX2's jpake-server subprocess)")
	var jpakeLongTermKey = flag.String("jpake-long-term-key", "", "hex-encoded JPAKE long-term key to pre-seed, letting a previously-paired client quick-pair (reconnect via Jpake3SessionKeyRequest directly) without a fresh full pairing; also displayed/settable in the web UI once derived from a completed pairing")
	var gradleCmd = flag.String("gradle-cmd", "./gradlew", "gradle command to use")
	var javaCmd = flag.String("java-cmd", "java", "java command to use")
	var transportName = flag.String("transport", defaultTransport(), "transport to use: 'ble' (real BLE peripheral, Linux only) or 'virtual' (radio-free virtual GATT link over TCP)")
	var virtualAddr = flag.String("virtual-addr", bluetooth.DefaultVirtualAddr, "address the virtual GATT transport listens on")
	var virtualPeripheralID = flag.String("virtual-peripheral-id", "", "stable peripheral UUID reported by the virtual transport (default: derived deterministically from the pump serial number)")
	var virtualMTU = flag.Int("virtual-mtu", bluetooth.DefaultATTMTU, "virtual transport only: ATT MTU to report and fragment responses to. The default 23 is the spec minimum and gives the 20-byte notifications pumpX2 and the cliparser jar produce; a real Mobi over iOS negotiates a much larger one, so most responses arrive in a single notification. Changeable at runtime via PUT /api/transport.")
	var virtualAckAfterHandling = flag.Bool("virtual-ack-after-handling", false, "virtual transport only: run the write handler synchronously and ack the write afterwards, reproducing the Linux/gatt ordering instead of real-pump ordering")
	var apiAddr = flag.String("api-addr", api.DefaultAddr, "address the HTTP/WebSocket API server listens on")
	var uiDir = flag.String("ui-dir", "", "directory to serve the web UI from (default: the 'ui' directory beside the executable, else ./ui)")
	var pumpTimeZone = flag.String("pump-timezone", "", "IANA time zone whose local time the emulated pump's clock is set to at startup, e.g. 'America/New_York' (default: the host's local zone; 'UTC' disables the local-time convention). A Tandem pump holds a date and time with no zone attached and every consumer decodes its wire timestamps as local time; the offset is fixed at startup and does not follow DST. Changeable at runtime via PUT /api/clock, POST /api/state/time and ChangeTimeDateRequest.")
	var requestLogSize = flag.Int("request-log-size", reqlog.DefaultCapacity, "how many messages the integration-harness request log (GET /api/log) retains")

	flag.Parse()

	// Determine log level
	logLevel := "debug"
	if *traceLevel {
		log.SetLevel(log.TraceLevel)
		logLevel = "trace"
	} else if *infoLevel {
		log.SetLevel(log.InfoLevel)
		logLevel = "info"
	} else {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableQuote: true,
		ForceColors:  true,
	})

	// Initialize configuration
	cfg, err := config.New(config.Options{
		MessageCodec:        *messageCodec,
		PumpX2Path:          *pumpX2Path,
		PumpX2Mode:          *pumpX2Mode,
		PumpX2JarPath:       *pumpX2JarPath,
		JPAKEMode:           *jpakeMode,
		GradleCmd:           *gradleCmd,
		JavaCmd:             *javaCmd,
		LogLevel:            logLevel,
		JPAKELongTermKeyHex: *jpakeLongTermKey,
	})
	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	log.Info("Starting Tandem Pump Emulator")
	log.Info("Service UUID: ", bluetooth.PumpServiceUUID)
	log.Info("Characteristics:")
	log.Info("  CurrentStatus:     ", bluetooth.CurrentStatusCharUUID)
	log.Info("  QualifyingEvents:  ", bluetooth.QualifyingEventsCharUUID)
	log.Info("  HistoryLog:        ", bluetooth.HistoryLogCharUUID)
	log.Info("  Authorization:     ", bluetooth.AuthorizationCharUUID)
	log.Info("  Control:           ", bluetooth.ControlCharUUID)
	log.Info("  ControlStream:     ", bluetooth.ControlStreamCharUUID)

	bridge := newMessageBridge(cfg, *pumpX2Path != "" || *pumpX2JarPath != "")
	defer func() { _ = bridge.Close() }() // Cleanup after the emulator stops.
	if bridge.Codec() != pumpx2.CodecNative {
		log.Info("Warming cliparser before accepting pairing requests...")
		if err := bridge.WarmUp(); err != nil {
			log.Fatalf("Failed to warm cliparser: %s", err)
		}
	}

	// Initialize protocol components
	reassembler := protocol.NewReassembler(30 * time.Second)
	defer reassembler.Stop()

	txManager := protocol.NewTransactionManager(10 * time.Second)

	log.Debugf("Protocol components initialized: reassembler timeout=30s, transaction timeout=10s")

	// Initialize pump state
	pumpState := state.NewPumpState()
	log.Infof("Pump state initialized: serial=%s, model=%s, API version=%d.%d",
		pumpState.GetSerialNumber(), pumpState.Model, pumpState.GetAPIVersionMajor(), pumpState.GetAPIVersionMinor())
	log.Infof("Initial state: reservoir=%.1f units, battery=%d%%, basal rate=%.2f U/hr",
		pumpState.GetReservoirLevel(), pumpState.GetBatteryLevel(), pumpState.GetBasalRate())

	applyPumpTimeZone(pumpState, *pumpTimeZone)

	// Keep the bridge's copy of the pairing code in step with pump state,
	// whichever API changes it: the websocket setPairingCode command, the
	// harness's PUT/PATCH /api/state, or anything else that reaches
	// PumpState.SetPairingCode.
	bridge.SetPairingCode(pumpState.GetPairingCode())
	pumpState.OnPairingCodeChange(bridge.SetPairingCode)

	if len(cfg.JPAKELongTermKey) > 0 {
		pumpState.SetLongTermKey(cfg.JPAKELongTermKey)
		log.Infof("Seeded JPAKE long-term key from -jpake-long-term-key flag (%d bytes); quick-pair reconnects will be honored", len(cfg.JPAKELongTermKey))
	}

	// Start background simulator
	simulator := state.NewSimulator(pumpState, 1*time.Second)
	defer simulator.Stop()

	ble, err := newTransport(transportConfig{
		name:             *transportName,
		virtualAddr:      *virtualAddr,
		peripheralID:     *virtualPeripheralID,
		ackAfterHandling: *virtualAckAfterHandling,
		attMTU:           *virtualMTU,
		serialNumber:     pumpState.GetSerialNumber(),
	})
	if err != nil {
		log.Fatalf("Could not start transport: %s", err)
	}

	// Create message router
	router := handler.NewRouter(bridge, pumpState, ble, txManager, cfg.JPAKEMode, cfg.PumpX2Path, cfg.PumpX2Mode, cfg.GradleCmd, cfg.JavaCmd, cfg.PumpX2JarPath)
	log.Info("Message router initialized")
	defer router.ClosePairing()
	if err := router.PreparePairing(); err != nil {
		log.Fatalf("Failed to prepare JPAKE pairing: %s", err)
	}
	pumpState.OnPairingCodeChange(func(string) {
		router.ResetJPAKESession()
		go preparePairing(router)
	})

	// Integration-harness plumbing: the pump-side message record and the fault
	// injector. Both are inert until a harness uses them -- an empty registry
	// injects nothing, and the log only observes.
	requestLog := reqlog.New(*requestLogSize)
	requestLog.SetClock(pumpState.Now)
	faultRegistry := faults.NewRegistry()
	router.SetRequestLog(requestLog)
	router.SetFaultRegistry(faultRegistry)

	// Connect simulator with qualifying events notifier
	simulator.SetEventNotifier(router.GetQualifyingEventsNotifier())
	log.Info("Qualifying events notifier connected to simulator")

	// Start simulator after event notifier is connected
	simulator.Start()
	log.Info("Background simulator started (update interval: 1s)")

	// Create API server
	server := api.New(ble)
	server.SetUIDir(*uiDir)
	server.SetSettingsManager(router.GetSettingsManager())

	h := harness.New(harness.Options{
		PumpState: pumpState,
		Simulator: simulator,
		Transport: ble,
		Router:    router,
		Requests:  requestLog,
		Faults:    faultRegistry,
	})
	server.SetExtraRoutes(h.RegisterRoutes, h.Endpoints())

	configureConnectionHandlers(ble, server, router, pumpState, requestLog, reassembler)

	configureWriteHandler(ble, server, bridge, router, pumpState, reassembler)

	// Set up read handler
	ble.SetReadHandler(func(charType bluetooth.CharacteristicType) []byte {
		log.Debugf("Read request on %s", charType)
		// TODO: Return appropriate data based on characteristic
		return nil
	})

	// Set up custom command handler for websocket commands
	configureWebsocketCommands(server, ble, pumpState, router)

	log.Info("Transport initialized, waiting for connections...")
	log.Infof("Starting API server on %s", *apiAddr)

	// Start API server (blocking)
	go server.Start(*apiAddr)

	// Keep the program running
	for {
		time.Sleep(time.Hour)
	}
}

// newMessageBridge says which codec and JPAKE mode the emulator runs and
// starts the codec. pumpX2Given is whether a pumpX2 location was passed,
// which only the pumpX2 codecs and JPAKE mode use.
func newMessageBridge(cfg *config.Config, pumpX2Given bool) *pumpx2.Bridge {
	log.Infof("Message codec: %s", cfg.MessageCodec)
	log.Infof("JPAKE mode: %s", cfg.JPAKEMode)
	if cfg.UsesPumpX2() {
		log.Infof("pumpX2: repository %q, jar %q, mode %s", cfg.PumpX2Path, cfg.PumpX2JarPath, cfg.PumpX2Mode)
	} else if pumpX2Given {
		log.Warn("A pumpX2 path was given but nothing uses it; pass -message-codec pumpx2 (or compare) and/or -jpake-mode pumpx2 to run pumpX2")
	}
	bridge, err := pumpx2.NewBridgeWithCodec(cfg.MessageCodec, cfg.PumpX2Path, cfg.PumpX2Mode, cfg.GradleCmd, cfg.JavaCmd, cfg.PumpX2JarPath)
	if err != nil {
		log.Fatalf("Failed to initialize the %s message codec: %s", cfg.MessageCodec, err)
	}
	return bridge
}

// defaultTransport picks the transport appropriate for the build platform:
// real BLE on Linux (where the gatt peripheral implementation exists) and the
// virtual link everywhere else.
func defaultTransport() string {
	if runtime.GOOS == "linux" {
		return transportBle
	}
	return transportVirtual
}

// transportConfig is the transport half of the command line, grouped so
// newTransport does not take a half-dozen positional strings and bools.
type transportConfig struct {
	name             string
	virtualAddr      string
	peripheralID     string
	ackAfterHandling bool
	attMTU           int
	serialNumber     string
}

// newTransport constructs the selected transport.
func newTransport(cfg transportConfig) (bluetooth.Transport, error) {
	switch cfg.name {
	case transportBle:
		log.Info("Using BLE transport (adapter hci0)")
		return bluetooth.New("hci0")
	case transportVirtual:
		log.Infof("Using virtual GATT transport on %s (ATT MTU %d, %d-byte notifications)",
			cfg.virtualAddr, cfg.attMTU, bluetooth.MaxNotificationBytes(cfg.attMTU))
		return bluetooth.NewVirtual(bluetooth.VirtualOptions{
			Addr:             cfg.virtualAddr,
			PeripheralID:     cfg.peripheralID,
			SerialNumber:     cfg.serialNumber,
			AckAfterHandling: cfg.ackAfterHandling,
			ATTMTU:           cfg.attMTU,
		})
	default:
		return nil, fmt.Errorf("unknown transport %q (expected %q or %q)", cfg.name, transportBle, transportVirtual)
	}
}

// isQualifyingEventAck reports whether data is a central's acknowledgement of a
// QualifyingEvents notification (four zero bytes written to that
// characteristic), rather than a pump protocol message.
func isQualifyingEventAck(charType bluetooth.CharacteristicType, data []byte) bool {
	if charType != bluetooth.CharQualifyingEvents || len(data) != 4 {
		return false
	}
	for _, b := range data {
		if b != 0x00 {
			return false
		}
	}
	return true
}

func configureWriteHandler(ble bluetooth.Transport, server *api.Server, bridge *pumpx2.Bridge, router *handler.Router, pumpState *state.PumpState, reassembler *protocol.Reassembler) {
	// Set up write handler to log incoming data and notify websocket clients
	ble.SetWriteHandler(func(charType bluetooth.CharacteristicType, data []byte) {
		protocol.LogPacket("RX", charType, data)
		server.SendWriteEvent(charType, data)

		// A central acknowledges every QualifyingEvents notification by writing
		// four zero bytes back to that characteristic. It is not a pump message:
		// feeding it to the reassembler and then to the parser produces garbage
		// (QualifyingEvents has no pumpX2 characteristic enum, so the opcode is
		// guessed). Consume it here.
		if isQualifyingEventAck(charType, data) {
			log.Debugf("Consumed QualifyingEvents acknowledgement write: %s", hex.EncodeToString(data))
			return
		}

		// Reassemble multi-packet messages
		message, rawPacketsHex, isComplete, err := reassembler.AddPacket(charType, data)
		if err != nil {
			log.Errorf("Failed to add packet to reassembler: %v", err)
			return
		}

		if !isComplete {
			log.Trace("Waiting for more packets...")
			return
		}

		// We have a complete message, parse it
		log.Infof("Received complete message on %s: %s", charType, hex.EncodeToString(message))
		pairingRequest := charType == bluetooth.CharAuthorization
		if pairingRequest {
			server.SendPairingProgress("Pairing request received; processing authentication...", false)
		}

		// Parse the message the way pumpX2's cliparser does
		parsed, err := bridge.ParseMessage(charType, rawPacketsHex)
		if err != nil {
			log.Errorf("Failed to parse message: %v", err)
			if pairingRequest {
				server.SendPairingProgress(fmt.Sprintf("Could not parse pairing request: %v", err), true)
			}
			return
		}
		if !ble.IsConnected() {
			return // The central left while parsing; do not start a stale handshake.
		}

		log.Infof("Parsed message: type=%s, txID=%d, opcode=%d",
			parsed.MessageType, parsed.TxID, parsed.Opcode)

		// Route to handler
		if err := router.RouteMessage(charType, parsed); err != nil {
			log.Errorf("Failed to route message: %v", err)
			if pairingRequest {
				server.SendPairingProgress(fmt.Sprintf("Pairing failed at %s: %v", parsed.MessageType, err), true)
			}
			if errors.Is(err, handler.ErrJPAKEQuickPairRejected) {
				log.Warn("Dropping connection to force client back to full pairing (no cached long-term JPAKE key available for this quick-pair reconnect)")
				ble.ShutdownConnection()
			}
			if errors.Is(err, handler.ErrJPAKEHandshakeFailed) {
				// No response to this round is ever coming: a proof or key
				// confirmation did not verify, or pumpX2's jpake-server gave
				// up. Cut the link instead of leaving the client to sit out
				// its own 30-90 second pairing timeout; it reconnects and
				// pairs again from scratch.
				log.Warn("Dropping connection: this JPAKE pairing cannot complete. The client should pair again.")
				router.ResetJPAKESession()
				ble.ShutdownConnection()
			}
			return
		}
		if pairingRequest {
			message := fmt.Sprintf("%s completed; waiting for the app's next pairing request.", parsed.MessageType)
			if pumpState.Authenticated() {
				message = "Pairing completed successfully."
			}
			server.SendPairingProgress(message, false)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.Authenticated(), pumpState.GetLongTermKey())
		}
	})

}

func configureConnectionHandlers(ble bluetooth.Transport, server *api.Server, router *handler.Router, pumpState *state.PumpState, requestLog *reqlog.Log, reassembler *protocol.Reassembler) {
	ble.SetConnectionHandler(func(connected bool) {
		server.SendPumpState()
		if requestLog != nil {
			requestLog.RecordConnection(connected, "")
		}
		if connected {
			log.Info("BLE central connected; updated websocket clients.")
			server.SendPairingProgress("App connected; waiting for an authentication request.", false)
			return
		}
		log.Info("BLE central disconnected; updated websocket clients.")
		// Clear any in-progress JPAKE authenticator so a stale/broken one
		// is never reused by the next connection attempt.
		wasAuthenticated := pumpState.Authenticated()
		reassembler.Reset()
		router.ResetJPAKESession()
		// A real pump derives a fresh session key per connection, so a central
		// that reconnects must authenticate again before any auth-gated message
		// is served. Leaving IsAuthenticated set across a disconnect let a
		// reconnecting central skip authentication entirely, which would mask
		// reconnect-auth regressions in the driver under test.
		//
		// This clears the per-connection session key only. The cached JPAKE
		// long-term key (PumpState.LongTermKey) deliberately survives, so the
		// quick-reconnect flow -- rounds 3/4 against the cached secret -- still
		// works on the next connection.
		pumpState.ResetAuthentication()
		server.SendPairingState(pumpState.GetPairingCode(), false, pumpState.GetLongTermKey())
		if wasAuthenticated {
			server.SendPairingProgress("App disconnected. Authentication is required on reconnect.", false)
		} else {
			server.SendPairingProgress("App disconnected before authentication completed. Retry pairing with the configured PIN.", true)
		}
		go preparePairing(router)
	})
}

func preparePairing(router *handler.Router) {
	if err := router.PreparePairing(); err != nil {
		log.Errorf("Failed to prepare the next JPAKE pairing attempt: %v", err)
	}
}

func configureWebsocketCommands(server *api.Server, ble bluetooth.Transport, pumpState *state.PumpState, router *handler.Router) {
	server.SetCommandHandler(func(command string, params map[string]interface{}) {
		log.Infof("Received command from websocket: %s, params: %v", command, params)
		switch command {
		case "getPairingState":
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.Authenticated(), pumpState.GetLongTermKey())
			server.SendLatestPairingProgress()
		case "setPairingCode":
			pairingCode, _ := params["pairingCode"].(string)
			if !validPairingCode(pairingCode) {
				server.SendPairingProgress("Pairing code must be exactly 6 digits.", true)
				return
			}
			// The bridge is updated by the observer main() registered on
			// PumpState, so every route that sets the code gets it.
			if pairingCode != pumpState.GetPairingCode() {
				ble.ShutdownConnection()
				pumpState.SetPairingCode(pairingCode)
				pumpState.ResetAuthentication()
			}
			server.SendPairingProgress("Pairing code saved. Use the same PIN in the Tandem app; authentication completes when the app pairs.", false)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.Authenticated(), pumpState.GetLongTermKey())
		case "resetPairing":
			ble.ShutdownConnection()
			router.ResetJPAKESession()
			pumpState.ResetAuthentication()
			server.SendPairingProgress("Authentication reset. Start a new pairing attempt in the app.", false)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.Authenticated(), pumpState.GetLongTermKey())
		case "setLongTermKey":
			longTermKeyHex, _ := params["longTermKey"].(string)
			longTermKey, err := hex.DecodeString(longTermKeyHex)
			if longTermKeyHex == "" || err != nil {
				log.Warnf("Invalid long-term key from setLongTermKey command: %q", longTermKeyHex)
				return
			}
			pumpState.SetLongTermKey(longTermKey)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.Authenticated(), pumpState.GetLongTermKey())
		case "resetLongTermKey":
			pumpState.SetLongTermKey(nil)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.Authenticated(), pumpState.GetLongTermKey())
		case "disconnectPump":
			ble.ShutdownConnection()
			server.SendPumpState()
		default:
			log.Warnf("Unhandled websocket command: %s", command)
		}
	})
}

func validPairingCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

// applyPumpTimeZone puts the pump's clock in the zone named by -pump-timezone,
// or leaves it in the host's, and says which.
//
// A Tandem pump keeps local time with no zone attached: the pump-epoch seconds
// it puts on the wire count from 2008-01-01 00:00:00 as read on its own clock,
// and every consumer decodes them that way. Emitting UTC-based seconds instead
// lands every timestamp one UTC offset from the pump's own record.
func applyPumpTimeZone(pumpState *state.PumpState, zone string) {
	if zone != "" {
		loc, err := time.LoadLocation(zone)
		if err != nil {
			log.Fatalf("Invalid -pump-timezone %q: %s", zone, err)
		}
		pumpState.SetPumpTimeZone(loc)
	}
	log.Infof("Pump clock: set to local time in %s (UTC offset %+d s, fixed until the clock is set again); wire timestamps are seconds since 2008-01-01 as read on that clock",
		pumpState.GetPumpTimeZone(), pumpState.PumpTimeZoneOffsetSeconds())
}
