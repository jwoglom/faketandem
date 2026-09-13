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
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
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
	var pumpX2Path = flag.String("pumpx2-path", "", "path to pumpX2 repository (required unless -pumpx2-jar-path is set)")
	var pumpX2Mode = flag.String("pumpx2-mode", "gradle", "mode to run cliparser: 'gradle' or 'jar'")
	var pumpX2JarPath = flag.String("pumpx2-jar-path", "", "path to a prebuilt cliparser jar; skips gradle entirely and implies -pumpx2-mode=jar")
	var jpakeMode = flag.String("jpake-mode", "pumpx2", "JPAKE mode: 'pumpx2' (real EC-JPAKE via pumpX2's jpake-server, required for real hardware/apps) or 'go' (simplified, cryptographically incompatible with real devices)")
	var jpakeLongTermKey = flag.String("jpake-long-term-key", "", "hex-encoded JPAKE long-term key to pre-seed, letting a previously-paired client quick-pair (reconnect via Jpake3SessionKeyRequest directly) without a fresh full pairing; also displayed/settable in the web UI once derived from a completed pairing")
	var gradleCmd = flag.String("gradle-cmd", "./gradlew", "gradle command to use")
	var javaCmd = flag.String("java-cmd", "java", "java command to use")
	var transportName = flag.String("transport", defaultTransport(), "transport to use: 'ble' (real BLE peripheral, Linux only) or 'virtual' (radio-free virtual GATT link over TCP)")
	var virtualAddr = flag.String("virtual-addr", bluetooth.DefaultVirtualAddr, "address the virtual GATT transport listens on")
	var virtualPeripheralID = flag.String("virtual-peripheral-id", "", "stable peripheral UUID reported by the virtual transport (default: derived deterministically from the pump serial number)")
	var virtualAckAfterHandling = flag.Bool("virtual-ack-after-handling", false, "virtual transport only: run the write handler synchronously and ack the write afterwards, reproducing the Linux/gatt ordering instead of real-pump ordering")
	var apiAddr = flag.String("api-addr", api.DefaultAddr, "address the HTTP/WebSocket API server listens on")
	var uiDir = flag.String("ui-dir", "", "directory to serve the web UI from (default: the 'ui' directory beside the executable, else ./ui)")

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
	cfg, err := config.New(*pumpX2Path, *pumpX2Mode, *jpakeMode, *gradleCmd, *javaCmd, logLevel, *pumpX2JarPath, *jpakeLongTermKey)
	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	log.Info("Starting Tandem Pump Emulator")
	log.Infof("pumpX2 repository: %s", cfg.PumpX2Path)
	log.Infof("pumpX2 mode: %s", cfg.PumpX2Mode)
	log.Infof("JPAKE mode: %s", cfg.JPAKEMode)
	log.Info("Service UUID: ", bluetooth.PumpServiceUUID)
	log.Info("Characteristics:")
	log.Info("  CurrentStatus:     ", bluetooth.CurrentStatusCharUUID)
	log.Info("  QualifyingEvents:  ", bluetooth.QualifyingEventsCharUUID)
	log.Info("  HistoryLog:        ", bluetooth.HistoryLogCharUUID)
	log.Info("  Authorization:     ", bluetooth.AuthorizationCharUUID)
	log.Info("  Control:           ", bluetooth.ControlCharUUID)
	log.Info("  ControlStream:     ", bluetooth.ControlStreamCharUUID)

	// Initialize pumpX2 bridge
	log.Info("Initializing pumpX2 bridge...")
	bridge, err := pumpx2.NewBridge(cfg.PumpX2Path, cfg.PumpX2Mode, cfg.GradleCmd, cfg.JavaCmd, cfg.PumpX2JarPath)
	if err != nil {
		log.Fatalf("Failed to initialize pumpX2 bridge: %s", err)
	}
	log.Info("pumpX2 bridge initialized successfully")

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

	// Set pairing code in bridge
	bridge.SetPairingCode(pumpState.GetPairingCode())

	if len(cfg.JPAKELongTermKey) > 0 {
		pumpState.SetLongTermKey(cfg.JPAKELongTermKey)
		log.Infof("Seeded JPAKE long-term key from -jpake-long-term-key flag (%d bytes); quick-pair reconnects will be honored", len(cfg.JPAKELongTermKey))
	}

	// Start background simulator
	simulator := state.NewSimulator(pumpState, 1*time.Second)
	defer simulator.Stop()

	ble, err := newTransport(*transportName, *virtualAddr, *virtualPeripheralID, *virtualAckAfterHandling, pumpState.GetSerialNumber())
	if err != nil {
		log.Fatalf("Could not start transport: %s", err)
	}

	// Create message router
	router := handler.NewRouter(bridge, pumpState, ble, txManager, cfg.JPAKEMode, cfg.PumpX2Path, cfg.PumpX2Mode, cfg.GradleCmd, cfg.JavaCmd, cfg.PumpX2JarPath)
	log.Info("Message router initialized")

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
	configureConnectionHandlers(ble, server, router)

	// Set up write handler to log incoming data and notify websocket clients
	ble.SetWriteHandler(func(charType bluetooth.CharacteristicType, data []byte) {
		protocol.LogPacket("RX", charType, data)
		server.SendWriteEvent(charType, data)

		// A central acknowledges every QualifyingEvents notification by writing
		// four zero bytes back to that characteristic. It is not a pump message:
		// feeding it to the reassembler and then to cliparser produces garbage
		// (QualifyingEvents has no pumpX2 characteristic enum, so the opcode is
		// guessed) and burns two JVM spawns per qualifying event. Consume it here.
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

		// Parse the message using pumpX2 bridge
		parsed, err := bridge.ParseMessage(charType, rawPacketsHex)
		if err != nil {
			log.Errorf("Failed to parse message: %v", err)
			return
		}

		log.Infof("Parsed message: type=%s, txID=%d, opcode=%d",
			parsed.MessageType, parsed.TxID, parsed.Opcode)

		// Route to handler
		if err := router.RouteMessage(charType, parsed); err != nil {
			log.Errorf("Failed to route message: %v", err)
			if errors.Is(err, handler.ErrJPAKEQuickPairRejected) {
				log.Warn("Dropping connection to force client back to full pairing (no cached long-term JPAKE key available for this quick-pair reconnect)")
				ble.ShutdownConnection()
			}
			return
		}
	})

	// Set up read handler
	ble.SetReadHandler(func(charType bluetooth.CharacteristicType) []byte {
		log.Debugf("Read request on %s", charType)
		// TODO: Return appropriate data based on characteristic
		return nil
	})

	// Set up custom command handler for websocket commands
	configureWebsocketCommands(server, ble, bridge, pumpState)

	log.Info("Transport initialized, waiting for connections...")
	log.Infof("Starting API server on %s", *apiAddr)

	// Start API server (blocking)
	go server.Start(*apiAddr)

	// Keep the program running
	for {
		time.Sleep(time.Hour)
	}
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

// newTransport constructs the selected transport.
func newTransport(name, virtualAddr, virtualPeripheralID string, virtualAckAfterHandling bool, serialNumber string) (bluetooth.Transport, error) {
	switch name {
	case transportBle:
		log.Info("Using BLE transport (adapter hci0)")
		return bluetooth.New("hci0")
	case transportVirtual:
		log.Infof("Using virtual GATT transport on %s", virtualAddr)
		return bluetooth.NewVirtual(bluetooth.VirtualOptions{
			Addr:             virtualAddr,
			PeripheralID:     virtualPeripheralID,
			SerialNumber:     serialNumber,
			AckAfterHandling: virtualAckAfterHandling,
		})
	default:
		return nil, fmt.Errorf("unknown transport %q (expected %q or %q)", name, transportBle, transportVirtual)
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

func configureConnectionHandlers(ble bluetooth.Transport, server *api.Server, router *handler.Router) {
	ble.SetConnectionHandler(func(connected bool) {
		server.SendPumpState()
		if connected {
			log.Info("BLE central connected; updated websocket clients.")
			return
		}
		log.Info("BLE central disconnected; updated websocket clients.")
		// Clear any in-progress JPAKE authenticator so a stale/broken one
		// (e.g. a pumpX2 subprocess that died mid-handshake) is never reused
		// by the next connection attempt.
		router.ResetJPAKESession()
	})
}

func configureWebsocketCommands(server *api.Server, ble bluetooth.Transport, bridge *pumpx2.Bridge, pumpState *state.PumpState) {
	server.SetCommandHandler(func(command string, params map[string]interface{}) {
		log.Infof("Received command from websocket: %s, params: %v", command, params)
		switch command {
		case "getPairingState":
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.IsAuthenticated, pumpState.GetLongTermKey())
		case "setPairingCode":
			pairingCode, _ := params["pairingCode"].(string)
			if pairingCode == "" {
				log.Warn("Pairing code missing from setPairingCode command")
				return
			}
			pumpState.SetPairingCode(pairingCode)
			pumpState.ResetAuthentication()
			bridge.SetPairingCode(pairingCode)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.IsAuthenticated, pumpState.GetLongTermKey())
		case "resetPairing":
			pumpState.ResetAuthentication()
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.IsAuthenticated, pumpState.GetLongTermKey())
		case "setLongTermKey":
			longTermKeyHex, _ := params["longTermKey"].(string)
			longTermKey, err := hex.DecodeString(longTermKeyHex)
			if longTermKeyHex == "" || err != nil {
				log.Warnf("Invalid long-term key from setLongTermKey command: %q", longTermKeyHex)
				return
			}
			pumpState.SetLongTermKey(longTermKey)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.IsAuthenticated, pumpState.GetLongTermKey())
		case "resetLongTermKey":
			pumpState.SetLongTermKey(nil)
			server.SendPairingState(pumpState.GetPairingCode(), pumpState.IsAuthenticated, pumpState.GetLongTermKey())
		case "disconnectPump":
			ble.ShutdownConnection()
			server.SendPumpState()
		default:
			log.Warnf("Unhandled websocket command: %s", command)
		}
	})
}
