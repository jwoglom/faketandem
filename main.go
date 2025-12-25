package main

import (
	"encoding/hex"
	"flag"
	"time"

	"github.com/avereha/pod/pkg/bluetooth"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func main() {
	// if both verbose and quiet are chosen, e.g., -v -q, the verbose dominates
	var traceLevel = flag.Bool("v", false, "verbose off by default, TraceLevel")
	var infoLevel = flag.Bool("q", false, "quiet off by default, InfoLevel")

	flag.Parse()

	if *traceLevel {
		log.SetLevel(log.TraceLevel)
	} else if *infoLevel {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
		ForceColors:  true,
	})

	log.Info("Starting Tandem Pump Emulator")
	log.Info("Service UUID: ", bluetooth.PumpServiceUUID)
	log.Info("Characteristics:")
	log.Info("  CurrentStatus:     ", bluetooth.CurrentStatusCharUUID)
	log.Info("  QualifyingEvents:  ", bluetooth.QualifyingEventsCharUUID)
	log.Info("  HistoryLog:        ", bluetooth.HistoryLogCharUUID)
	log.Info("  Authorization:     ", bluetooth.AuthorizationCharUUID)
	log.Info("  Control:           ", bluetooth.ControlCharUUID)
	log.Info("  ControlStream:     ", bluetooth.ControlStreamCharUUID)

	ble, err := bluetooth.New("hci0")
	if err != nil {
		log.Fatalf("Could not start BLE: %s", err)
	}

	// Set up write handler to log incoming data
	ble.SetWriteHandler(func(charType bluetooth.CharacteristicType, data []byte) {
		log.Infof("Received write on %s: %s", charType, hex.EncodeToString(data))
		// TODO: Add your response logic here
	})

	// Set up read handler
	ble.SetReadHandler(func(charType bluetooth.CharacteristicType) []byte {
		log.Debugf("Read request on %s", charType)
		// TODO: Return appropriate data based on characteristic
		return nil
	})

	log.Info("Bluetooth device initialized, waiting for connections...")

	// Keep the program running
	for {
		time.Sleep(time.Hour)
	}
}
