package bluetooth

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/linux/cmd"
	log "github.com/sirupsen/logrus"
)

// Service UUID for the Tandem pump
const (
	PumpServiceUUID = "0000fdfb-0000-1000-8000-00805f9b34fb"
)

// Characteristic UUIDs
const (
	CurrentStatusCharUUID      = "7B83FFF6-9F77-4E5C-8064-AAE2C24838B9"
	QualifyingEventsCharUUID   = "7B83FFF7-9F77-4E5C-8064-AAE2C24838B9"
	HistoryLogCharUUID         = "7B83FFF8-9F77-4E5C-8064-AAE2C24838B9"
	AuthorizationCharUUID      = "7B83FFF9-9F77-4E5C-8064-AAE2C24838B9"
	ControlCharUUID            = "7B83FFFC-9F77-4E5C-8064-AAE2C24838B9"
	ControlStreamCharUUID      = "7B83FFFD-9F77-4E5C-8064-AAE2C24838B9"
)

// CharacteristicType identifies which characteristic received data
type CharacteristicType int

const (
	CharCurrentStatus CharacteristicType = iota
	CharQualifyingEvents
	CharHistoryLog
	CharAuthorization
	CharControl
	CharControlStream
)

func (c CharacteristicType) String() string {
	switch c {
	case CharCurrentStatus:
		return "CurrentStatus"
	case CharQualifyingEvents:
		return "QualifyingEvents"
	case CharHistoryLog:
		return "HistoryLog"
	case CharAuthorization:
		return "Authorization"
	case CharControl:
		return "Control"
	case CharControlStream:
		return "ControlStream"
	default:
		return "Unknown"
	}
}

// WriteHandler is called when data is written to a characteristic
type WriteHandler func(charType CharacteristicType, data []byte)

// ReadHandler is called when data is read from a characteristic
type ReadHandler func(charType CharacteristicType) []byte

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
	err = d.AdvertiseNameAndServices("TandemPump", []gatt.UUID{serviceUUID})
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
		rsp.Write(data)
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
		(*b.central).Close()
	}
}
