package main

import (
	"encoding/hex"
	"flag"
	"time"

	"github.com/jwoglom/faketandem/pkg/api"
	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/config"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func main() {
	// if both verbose and quiet are chosen, e.g., -v -q, the verbose dominates
	var traceLevel = flag.Bool("v", false, "verbose off by default, TraceLevel")
	var infoLevel = flag.Bool("q", false, "quiet off by default, InfoLevel")
	var pumpX2Path = flag.String("pumpx2-path", "", "path to pumpX2 repository (required)")
	var pumpX2Mode = flag.String("pumpx2-mode", "gradle", "mode to run cliparser: 'gradle' or 'jar'")
	var gradleCmd = flag.String("gradle-cmd", "./gradlew", "gradle command to use")
	var javaCmd = flag.String("java-cmd", "java", "java command to use")

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

	log.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
		ForceColors:  true,
	})

	// Initialize configuration
	cfg, err := config.New(*pumpX2Path, *pumpX2Mode, *gradleCmd, *javaCmd, logLevel)
	if err != nil {
		log.Fatalf("Configuration error: %s", err)
	}

	log.Info("Starting Tandem Pump Emulator")
	log.Infof("pumpX2 repository: %s", cfg.PumpX2Path)
	log.Infof("pumpX2 mode: %s", cfg.PumpX2Mode)
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
	bridge, err := pumpx2.NewBridge(cfg.PumpX2Path, cfg.PumpX2Mode, cfg.GradleCmd, cfg.JavaCmd)
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
	log.Infof("Pump state initialized: serial=%s, model=%s, API version=%d",
		pumpState.GetSerialNumber(), pumpState.Model, pumpState.GetAPIVersion())
	log.Infof("Initial state: reservoir=%.1f units, battery=%d%%, basal rate=%.2f U/hr",
		pumpState.GetReservoirLevel(), pumpState.GetBatteryLevel(), pumpState.GetBasalRate())

	// Set pairing code in bridge
	bridge.SetPairingCode(pumpState.GetPairingCode())

	ble, err := bluetooth.New("hci0")
	if err != nil {
		log.Fatalf("Could not start BLE: %s", err)
	}

	// Create message router
	router := handler.NewRouter(bridge, pumpState, ble, txManager)
	log.Info("Message router initialized")

	// Create API server
	server := api.New(ble)

	// Set up write handler to log incoming data and notify websocket clients
	ble.SetWriteHandler(func(charType bluetooth.CharacteristicType, data []byte) {
		protocol.LogPacket("RX", charType, data)
		server.SendWriteEvent(charType, data)

		// Reassemble multi-packet messages
		message, isComplete, err := reassembler.AddPacket(charType, data)
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
		parsed, err := bridge.ParseMessage(int(charType), hex.EncodeToString(message))
		if err != nil {
			log.Errorf("Failed to parse message: %v", err)
			return
		}

		log.Infof("Parsed message: type=%s, txID=%d, opcode=%d",
			parsed.MessageType, parsed.TxID, parsed.Opcode)

		// Route to handler
		if err := router.RouteMessage(charType, parsed); err != nil {
			log.Errorf("Failed to route message: %v", err)
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
	server.SetCommandHandler(func(command string, params map[string]interface{}) {
		log.Infof("Received command from websocket: %s, params: %v", command, params)
		// TODO: Handle custom commands
	})

	log.Info("Bluetooth device initialized, waiting for connections...")
	log.Info("Starting API server on :8080")

	// Start API server (blocking)
	go server.Start()

	// Keep the program running
	for {
		time.Sleep(time.Hour)
	}
}
