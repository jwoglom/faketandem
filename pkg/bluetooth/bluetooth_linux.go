//go:build linux

package bluetooth

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/linux/cmd"
	log "github.com/sirupsen/logrus"
)

const (
	advTypeSomeUUID16 = 0x02
	advTypeTxPower    = 0x0A
	pumpName          = "Tandem Mobi 123"
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

	writeNotifyChars map[CharacteristicType]*gatt.Characteristic
	notifyOnlyChars  map[CharacteristicType]*gatt.Characteristic

	// Handlers
	writeHandler      WriteHandler
	readHandler       ReadHandler
	connectionHandler ConnectionHandler

	// Pairing state
	pairingState    PairingState
	pairingStateMtx sync.RWMutex
	pumpNameForAdv  string
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
		pairingState:  PairingStateNotDiscoverable,
		writeNotifyChars: make(map[CharacteristicType]*gatt.Characteristic),
		notifyOnlyChars:  make(map[CharacteristicType]*gatt.Characteristic),
	}

	d.Handle(
		gatt.CentralConnected(func(c gatt.Central) {
			fmt.Println("pkg bluetooth; ** New connection from:", c.ID())
			
			// Reject connection if not discoverable
			b.pairingStateMtx.RLock()
			state := b.pairingState
			b.pairingStateMtx.RUnlock()
			
			if state == PairingStateNotDiscoverable {
				log.Warnf("pkg bluetooth; rejecting connection from %s - not discoverable", c.ID())
				if err := c.Close(); err != nil {
					log.Debugf("Error closing rejected connection: %v", err)
				}
				return
			}
			
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
	b.pumpNameForAdv = pumpName

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

	err = b.advertisePump(d, pumpName)
	if err != nil {
		log.Fatalf("pkg bluetooth; could not advertise: %s", err)
	}

	log.Info("pkg bluetooth; Pump service is now advertising")
	log.Info("pkg bluetooth; Service UUID:", PumpServiceUUID)
	log.Info("pkg bluetooth; Ready for connections (discoverable: false)")
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

	b.addReadWriteCharacteristic(s, DeviceNameCharUUID, []byte(pumpName))
	b.addReadOnlyCharacteristic(s, AppearanceCharUUID, []byte{0x00, 0x00})
	b.addReadOnlyCharacteristic(s, PeripheralPreferredConnectionParametersCharUUID, []byte{0x18, 0x00, 0x28, 0x00, 0x00, 0x00, 0xf4, 0x01})
	b.addReadOnlyCharacteristic(s, CentralAddressResolutionCharUUID, []byte{0x01})

	b.addService(d, s, "Generic Access")
}

func (b *Ble) addDeviceInformationService(d gatt.Device) {
	serviceUUID := gatt.MustParseUUID(DeviceInformationServiceUUID)
	s := gatt.NewService(serviceUUID)

	b.addReadOnlyCharacteristic(s, ManufacturerNameStringCharUUID, []byte("Tandem Diabetes Care"))
	b.addReadOnlyCharacteristic(s, ModelNumberStringCharUUID, []byte("X2")) // Always "X2" even for Mobi
	b.addReadOnlyCharacteristic(s, SerialNumberStringCharUUID, []byte("bi 123"))
	b.addReadOnlyCharacteristic(s, SoftwareRevisionStringCharUUID, []byte("3553172181"))

	b.addService(d, s, "Device Information")
}

func (b *Ble) addUnknownServiceFDFA(d gatt.Device) {
	serviceUUID := gatt.MustParseUUID(UnknownServiceFDFAUUID)
	s := gatt.NewService(serviceUUID)

	b.addUnknownWriteNotifyCharacteristic(s, UnknownCharFFE8UUID)
	b.addUnknownWriteOnlyCharacteristic(s, UnknownCharFFE7UUID)

	b.addService(d, s, "Unknown FDFA")
}

func (b *Ble) advertisePump(d gatt.Device, name string) error {
	b.pairingStateMtx.RLock()
	state := b.pairingState
	b.pairingStateMtx.RUnlock()

	advPacket := &gatt.AdvPacket{}
	
	// Set flags based on discoverable state
	if state == PairingStateNotDiscoverable {
		advPacket.AppendFlags(0x04) // BR/EDR Not Supported (not discoverable)
	} else {
		advPacket.AppendFlags(0x06) // LE General Discoverable + BR/EDR Not Supported
	}
	
	advPacket.AppendField(advTypeSomeUUID16, uint16ToBytes(0xFDFB))
	advPacket.AppendField(advTypeTxPower, []byte{0x04})
	
	// Set manufacturer data based on pairing state
	var lastByte byte
	switch state {
	case PairingStateNotDiscoverable:
		lastByte = 0x10 // Not discoverable
	case PairingStateDiscoverableOnly:
		lastByte = 0x10 // Discoverable but no pairing step
	case PairingStatePairStep1:
		lastByte = 0x11 // Discoverable with PairStep1
	case PairingStatePairStep2:
		lastByte = 0x12 // Discoverable with PairStep2
	default:
		lastByte = 0x10
	}
	mfgData := []byte{0x00, 0x01, lastByte}
	advPacket.AppendManufacturerData(0x059D, mfgData)

	scanPacket := &gatt.AdvPacket{}
	scanPacket.AppendName(name)

	advData := &cmd.LESetAdvertisingData{
		AdvertisingDataLength: uint8(advPacket.Len()),
		AdvertisingData:       advPacket.Bytes(),
	}
	scanData := &cmd.LESetScanResponseData{
		ScanResponseDataLength: uint8(scanPacket.Len()),
		ScanResponseData:       scanPacket.Bytes(),
	}

	if err := d.Option(
		gatt.LnxSetAdvertisingData(advData),
		gatt.LnxSetScanResponseData(scanData),
	); err != nil {
		return err
	}

	return d.Option(gatt.LnxSetAdvertisingEnable(true))
}

func (b *Ble) updateAdvertising(d gatt.Device, name string) error {
	// Disable advertising before updating data
	if err := d.Option(gatt.LnxSetAdvertisingEnable(false)); err != nil {
		return fmt.Errorf("failed to disable advertising: %w", err)
	}

	// Update the advertising data
	if err := b.advertisePump(d, name); err != nil {
		return err
	}

	return nil
}

func uint16ToBytes(value uint16) []byte {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, value)
	return bytes
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
	b.writeNotifyChars[charType] = char
	b.bindWriteNotifyHandlers(char, charType)
}

func (b *Ble) addNotifyOnlyCharacteristic(s *gatt.Service, uuidStr string, charType CharacteristicType) {
	charUUID := gatt.MustParseUUID(uuidStr)
	char := s.AddCharacteristic(charUUID)
	b.notifyOnlyChars[charType] = char
	b.bindNotifyHandlers(char, charType)
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

func (b *Ble) bindWriteNotifyHandlers(char *gatt.Characteristic, charType CharacteristicType) {
	char.HandleWriteFunc(func(r gatt.Request, data []byte) (status byte) {
		log.Tracef("pkg bluetooth; received write on %s: %s", charType, hex.EncodeToString(data))

		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		if b.writeHandler != nil {
			b.writeHandler(charType, dataCopy)
		}
		return 0
	})

	b.bindNotifyHandlers(char, charType)
}

func (b *Ble) bindNotifyHandlers(char *gatt.Characteristic, charType CharacteristicType) {
	char.HandleNotifyFunc(func(r gatt.Request, n gatt.Notifier) {
		b.notifiersMtx.Lock()
		b.notifiers[charType] = n
		b.notifiersMtx.Unlock()
		log.Infof("pkg bluetooth; notifications enabled for %s from %s", charType, r.Central.ID())
	})
}

func (b *Ble) reenableCharacteristicHandlers() {
	for charType, char := range b.writeNotifyChars {
		if char == nil {
			continue
		}
		b.bindWriteNotifyHandlers(char, charType)
	}

	for charType, char := range b.notifyOnlyChars {
		if char == nil {
			continue
		}
		b.bindNotifyHandlers(char, charType)
	}
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

// SetPairingState sets the pairing/discoverable state
func (b *Ble) SetPairingState(state PairingState) error {
	b.pairingStateMtx.Lock()
	b.pairingState = state
	b.pairingStateMtx.Unlock()

	if b.device == nil {
		return fmt.Errorf("device not initialized")
	}

	// If setting to not discoverable, disconnect any existing connection
	if state == PairingStateNotDiscoverable && b.central != nil {
		log.Info("pkg bluetooth; disconnecting existing connection due to non-discoverable mode")
		b.ShutdownConnection()
	}

	// Update the advertising data (disables, updates, re-enables)
	if err := b.updateAdvertising(*b.device, b.pumpNameForAdv); err != nil {
		return fmt.Errorf("failed to update advertising: %w", err)
	}

	b.reenableCharacteristicHandlers()

	log.Infof("Pairing state set to: %v", state)
	return nil
}

// GetPairingState returns the current pairing state
func (b *Ble) GetPairingState() PairingState {
	b.pairingStateMtx.RLock()
	defer b.pairingStateMtx.RUnlock()
	return b.pairingState
}
