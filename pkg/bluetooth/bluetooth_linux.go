//go:build linux

package bluetooth

import (
	"encoding/hex"
	"fmt"
	"strings"
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

	extraCharData    map[string][]byte
	extraCharDataMtx sync.RWMutex

	// Handlers
	writeHandler      WriteHandler
	readHandler       ReadHandler
	connectionHandler ConnectionHandler
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
		device:        &d,
		notifiers:     make(map[CharacteristicType]gatt.Notifier),
		charData:      make(map[CharacteristicType][]byte),
		extraCharData: make(map[string][]byte),
	}

	d.Handle(
		gatt.CentralConnected(func(c gatt.Central) {
			fmt.Println("pkg bluetooth; ** New connection from:", c.ID())
			b.central = &c
			if b.connectionHandler != nil {
				b.connectionHandler(true)
			}
		}),
		gatt.CentralDisconnected(func(c gatt.Central) {
			log.Tracef("pkg bluetooth; ** disconnect: %s", c.ID())
			b.central = nil
			if b.connectionHandler != nil {
				b.connectionHandler(false)
			}
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
	b.addGenericAttributeService(d)
	b.addGenericAccessService(d)
	b.addDeviceInformationService(d)
	b.addUnknownServiceFDFA(d)

	serviceUUID := gatt.MustParseUUID(PumpServiceUUID)
	s := gatt.NewService(serviceUUID)

	// Add all characteristics
	b.addWriteNotifyCharacteristic(s, CurrentStatusCharUUID, CharCurrentStatus)
	b.addWriteNotifyCharacteristic(s, QualifyingEventsCharUUID, CharQualifyingEvents)
	b.addNotifyOnlyCharacteristic(s, HistoryLogCharUUID, CharHistoryLog)
	b.addWriteNotifyCharacteristic(s, AuthorizationCharUUID, CharAuthorization)
	b.addWriteNotifyCharacteristic(s, ControlCharUUID, CharControl)
	b.addWriteNotifyCharacteristic(s, ControlStreamCharUUID, CharControlStream)

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

func (b *Ble) addGenericAttributeService(d gatt.Device) {
	serviceUUID := gatt.MustParseUUID(GenericAttributeServiceUUID)
	s := gatt.NewService(serviceUUID)
	char := s.AddCharacteristic(gatt.MustParseUUID(ServiceChangedCharUUID))
	char.HandleNotifyFunc(func(r gatt.Request, n gatt.Notifier) {
		log.Infof("pkg bluetooth; service changed indications enabled from %s", r.Central.ID())
	})
	b.addService(d, s, "Generic Attribute")
}

func (b *Ble) addGenericAccessService(d gatt.Device) {
	serviceUUID := gatt.MustParseUUID(GenericAccessServiceUUID)
	s := gatt.NewService(serviceUUID)

	b.addReadWriteCharacteristic(s, DeviceNameCharUUID, []byte("Tandem Mobi 123"))
	b.addReadOnlyCharacteristic(s, AppearanceCharUUID, []byte{0x00, 0x00})
	b.addReadOnlyCharacteristic(s, PeripheralPreferredConnectionParametersCharUUID, []byte{0x18, 0x00, 0x28, 0x00, 0x00, 0x00, 0xf4, 0x01})
	b.addReadOnlyCharacteristic(s, CentralAddressResolutionCharUUID, []byte{0x01})

	b.addService(d, s, "Generic Access")
}

func (b *Ble) addDeviceInformationService(d gatt.Device) {
	serviceUUID := gatt.MustParseUUID(DeviceInformationServiceUUID)
	s := gatt.NewService(serviceUUID)

	b.addReadOnlyCharacteristic(s, ManufacturerNameStringCharUUID, []byte("Tandem Diabetes Care"))
	b.addReadOnlyCharacteristic(s, ModelNumberStringCharUUID, []byte("Mobi"))
	b.addReadOnlyCharacteristic(s, SerialNumberStringCharUUID, []byte("11223344"))
	b.addReadOnlyCharacteristic(s, SoftwareRevisionStringCharUUID, []byte("1.0.0"))

	b.addService(d, s, "Device Information")
}

func (b *Ble) addUnknownServiceFDFA(d gatt.Device) {
	serviceUUID := gatt.MustParseUUID(UnknownServiceFDFAUUID)
	s := gatt.NewService(serviceUUID)

	b.addUnknownWriteNotifyCharacteristic(s, UnknownCharFFE8UUID)
	b.addUnknownWriteOnlyCharacteristic(s, UnknownCharFFE7UUID)

	b.addService(d, s, "Unknown FDFA")
}


func (b *Ble) addService(d gatt.Device, s *gatt.Service, name string) {
	if err := d.AddService(s); err != nil {
		log.Fatalf("pkg bluetooth; could not add %s service: %s", name, err)
	}
}

// addCharacteristic adds a characteristic to the service with read, write, and notify capabilities

func (b *Ble) addWriteNotifyCharacteristic(s *gatt.Service, uuidStr string, charType CharacteristicType) {
	charUUID := gatt.MustParseUUID(uuidStr)
	char := s.AddCharacteristic(charUUID)

	char.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		log.Tracef("pkg bluetooth; received write on %s: %s", charType, hex.EncodeToString(data))

		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		if b.writeHandler != nil {
			b.writeHandler(charType, dataCopy)
		}
		return 0
	})

	char.HandleNotifyFunc(func(r gatt.Request, n gatt.Notifier) {
		b.notifiersMtx.Lock()
		b.notifiers[charType] = n
		b.notifiersMtx.Unlock()
		log.Infof("pkg bluetooth; notifications enabled for %s from %s", charType, r.Central.ID())
	})
}

func (b *Ble) addNotifyOnlyCharacteristic(s *gatt.Service, uuidStr string, charType CharacteristicType) {
	charUUID := gatt.MustParseUUID(uuidStr)
	char := s.AddCharacteristic(charUUID)
	char.HandleNotifyFunc(func(r gatt.Request, n gatt.Notifier) {
		b.notifiersMtx.Lock()
		b.notifiers[charType] = n
		b.notifiersMtx.Unlock()
		log.Infof("pkg bluetooth; notifications enabled for %s from %s", charType, r.Central.ID())
	})
}

func (b *Ble) addUnknownWriteNotifyCharacteristic(s *gatt.Service, uuidStr string) {
	charUUID := gatt.MustParseUUID(uuidStr)
	char := s.AddCharacteristic(charUUID)
	char.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		log.Tracef("pkg bluetooth; received write on %s: %s", uuidStr, hex.EncodeToString(data))
		return 0
	})
	char.HandleNotifyFunc(func(r gatt.Request, n gatt.Notifier) {
		log.Infof("pkg bluetooth; notifications enabled for %s from %s", uuidStr, r.Central.ID())
	})
}

func (b *Ble) addUnknownWriteOnlyCharacteristic(s *gatt.Service, uuidStr string) {
	charUUID := gatt.MustParseUUID(uuidStr)
	char := s.AddCharacteristic(charUUID)
	char.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		log.Tracef("pkg bluetooth; received write on %s: %s", uuidStr, hex.EncodeToString(data))
		return 0
	})
}

func (b *Ble) addReadOnlyCharacteristic(s *gatt.Service, uuidStr string, initialValue []byte) {
	if initialValue != nil {
		b.setExtraCharacteristicData(uuidStr, initialValue)
	}
	char := s.AddCharacteristic(gatt.MustParseUUID(uuidStr))
	char.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		data := b.getExtraCharacteristicData(uuidStr)
		if data == nil {
			data = []byte{}
		}
		log.Tracef("pkg bluetooth; read request on %s, responding with: %s", uuidStr, hex.EncodeToString(data))
		if _, err := rsp.Write(data); err != nil {
			log.Warnf("Failed to write BLE response: %v", err)
		}
	})
}

func (b *Ble) addReadWriteCharacteristic(s *gatt.Service, uuidStr string, initialValue []byte) {
	if initialValue != nil {
		b.setExtraCharacteristicData(uuidStr, initialValue)
	}
	char := s.AddCharacteristic(gatt.MustParseUUID(uuidStr))
	char.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		b.setExtraCharacteristicData(uuidStr, dataCopy)
		log.Tracef("pkg bluetooth; received write on %s: %s", uuidStr, hex.EncodeToString(dataCopy))
		return 0
	})
	char.HandleReadFunc(func(rsp gatt.ResponseWriter, req *gatt.ReadRequest) {
		data := b.getExtraCharacteristicData(uuidStr)
		if data == nil {
			data = []byte{}
		}
		log.Tracef("pkg bluetooth; read request on %s, responding with: %s", uuidStr, hex.EncodeToString(data))
		if _, err := rsp.Write(data); err != nil {
			log.Warnf("Failed to write BLE response: %v", err)
		}
	})
}


func (b *Ble) setExtraCharacteristicData(uuidStr string, data []byte) {
	if data == nil {
		return
	}
	key := strings.ToLower(uuidStr)
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	b.extraCharDataMtx.Lock()
	b.extraCharData[key] = dataCopy
	b.extraCharDataMtx.Unlock()
}

func (b *Ble) getExtraCharacteristicData(uuidStr string) []byte {
	key := strings.ToLower(uuidStr)
	b.extraCharDataMtx.RLock()
	data := b.extraCharData[key]
	b.extraCharDataMtx.RUnlock()
	if data == nil {
		return nil
	}
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	return dataCopy
}

// SetWriteHandler sets the callback for when data is written to any characteristic
func (b *Ble) SetWriteHandler(handler WriteHandler) {
	b.writeHandler = handler
}

// SetReadHandler sets the callback for when data is read from any characteristic
func (b *Ble) SetReadHandler(handler ReadHandler) {
	b.readHandler = handler
}

// SetConnectionHandler sets the callback for when a central connects or disconnects
func (b *Ble) SetConnectionHandler(handler ConnectionHandler) {
	b.connectionHandler = handler
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
