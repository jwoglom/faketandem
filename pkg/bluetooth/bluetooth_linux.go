//go:build linux

package bluetooth

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/linux/cmd"
	log "github.com/sirupsen/logrus"
)

// Ble represents the Bluetooth Low Energy device
type Ble struct {
	device  *gatt.Device
	central *gatt.Central

	// Notifiers for each characteristic
	notifiers    map[CharacteristicType]gatt.Notifier
	notifiersMtx sync.Mutex

	// Data storage for each characteristic (for reads)
	charData    map[CharacteristicType][]byte
	charDataMtx sync.RWMutex

	// Handlers
	writeHandler WriteHandler
	readHandler  ReadHandler
}

// DefaultServerOptions contains the default options for the BLE server on Linux
var DefaultServerOptions = []gatt.Option{
	gatt.LnxMaxConnections(1),
	gatt.LnxDeviceID(-1, true),
	gatt.LnxSetAdvertisingParameters(&cmd.LESetAdvertisingParameters{
		AdvertisingIntervalMin: 0x00f4,
		AdvertisingIntervalMax: 0x00f4,
		AdvertisingChannelMap:  0x7,
	}),
}

// New creates a new BLE device with the Tandem pump service
func New(adapterID string) (*Ble, error) {
	d, err := gatt.NewDevice(DefaultServerOptions...)
	if err != nil {
		log.Fatalf("pkg bluetooth; failed to open device, err: %s", err)
		return nil, err
	}

	b := &Ble{
		device:    &d,
		notifiers: make(map[CharacteristicType]gatt.Notifier),
		charData:  make(map[CharacteristicType][]byte),
	}

	d.Handle(
		gatt.CentralConnected(func(c gatt.Central) {
			fmt.Println("pkg bluetooth; ** New connection from:", c.ID())
			b.central = &c
		}),
		gatt.CentralDisconnected(func(c gatt.Central) {
			log.Tracef("pkg bluetooth; ** disconnect: %s", c.ID())
			b.central = nil
		}),
	)

	// Handler for when the device is powered on
	onStateChanged := func(d gatt.Device, s gatt.State) {
		fmt.Printf("Bluetooth state: %s\n", s)
		switch s {
		case gatt.StatePoweredOn:
			b.setupService(d)
		default:
		}
	}

	err = d.Init(onStateChanged)
	if err != nil {
		log.Fatalf("pkg bluetooth; could not init bluetooth: %s", err)
		return nil, err
	}

	return b, nil
}

// setupService creates the pump service and all characteristics
func (b *Ble) setupService(d gatt.Device) {
	serviceUUID := gatt.MustParseUUID(PumpServiceUUID)
	s := gatt.NewService(serviceUUID)

	// Add all characteristics
	b.addCharacteristic(s, CurrentStatusCharUUID, CharCurrentStatus)
	b.addCharacteristic(s, QualifyingEventsCharUUID, CharQualifyingEvents)
	b.addCharacteristic(s, HistoryLogCharUUID, CharHistoryLog)
	b.addCharacteristic(s, AuthorizationCharUUID, CharAuthorization)
	b.addCharacteristic(s, ControlCharUUID, CharControl)
	b.addCharacteristic(s, ControlStreamCharUUID, CharControlStream)

	err := d.AddService(s)
	if err != nil {
		log.Fatalf("pkg bluetooth; could not add service: %s", err)
	}

	// Advertise the service
	err = d.AdvertiseNameAndServices("Tandem Mobi 123", []gatt.UUID{serviceUUID})
	if err != nil {
		log.Fatalf("pkg bluetooth; could not advertise: %s", err)
	}

	log.Info("pkg bluetooth; Pump service is now advertising")
	log.Info("pkg bluetooth; Service UUID:", PumpServiceUUID)
	log.Info("pkg bluetooth; Ready for connections")
}

// addCharacteristic adds a characteristic to the service with read, write, and notify capabilities
func (b *Ble) addCharacteristic(s *gatt.Service, uuidStr string, charType CharacteristicType) {
	charUUID := gatt.MustParseUUID(uuidStr)
	char := s.AddCharacteristic(charUUID)

	// Handle writes
	char.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		log.Tracef("pkg bluetooth; received write on %s: %s", charType, hex.EncodeToString(data))

		// Make a copy of the data
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		if b.writeHandler != nil {
			b.writeHandler(charType, dataCopy)
		}
		return 0
	})

	// Handle reads
	char.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		b.charDataMtx.RLock()
		data := b.charData[charType]
		b.charDataMtx.RUnlock()

		if b.readHandler != nil {
			data = b.readHandler(charType)
		}

		if data == nil {
			data = []byte{}
		}

		log.Tracef("pkg bluetooth; read request on %s, responding with: %s", charType, hex.EncodeToString(data))
		if _, err := rsp.Write(data); err != nil {
			log.Warnf("Failed to write BLE response: %v", err)
		}
	})

	// Handle notifications
	char.HandleNotifyFunc(func(r gatt.Request, n gatt.Notifier) {
		b.notifiersMtx.Lock()
		b.notifiers[charType] = n
		b.notifiersMtx.Unlock()
		log.Infof("pkg bluetooth; notifications enabled for %s from %s", charType, r.Central.ID())
	})
}

// SetWriteHandler sets the callback for when data is written to any characteristic
func (b *Ble) SetWriteHandler(handler WriteHandler) {
	b.writeHandler = handler
}

// SetReadHandler sets the callback for when data is read from any characteristic
func (b *Ble) SetReadHandler(handler ReadHandler) {
	b.readHandler = handler
}

// SetCharacteristicData sets the data that will be returned when a characteristic is read
func (b *Ble) SetCharacteristicData(charType CharacteristicType, data []byte) {
	b.charDataMtx.Lock()
	defer b.charDataMtx.Unlock()
	b.charData[charType] = data
}

// Notify sends a notification on the specified characteristic
func (b *Ble) Notify(charType CharacteristicType, data []byte) error {
	b.notifiersMtx.Lock()
	notifier, exists := b.notifiers[charType]
	b.notifiersMtx.Unlock()

	if !exists || notifier == nil {
		return fmt.Errorf("no notifier registered for %s", charType)
	}

	if notifier.Done() {
		return fmt.Errorf("notifier for %s is closed", charType)
	}

	log.Tracef("pkg bluetooth; sending notification on %s: %s", charType, hex.EncodeToString(data))
	_, err := notifier.Write(data)
	return err
}

// IsConnected returns true if a central device is connected
func (b *Ble) IsConnected() bool {
	return b.central != nil
}

// ShutdownConnection closes the connection with the central device
func (b *Ble) ShutdownConnection() {
	if b.central != nil {
		if err := (*b.central).Close(); err != nil {
			log.Debugf("Error closing central connection: %v", err)
		}
	}
}
