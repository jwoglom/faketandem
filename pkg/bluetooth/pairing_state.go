package bluetooth

// PairingState represents the current pairing/discoverable state
type PairingState string

const (
	// PairingStateNotDiscoverable - not discoverable, manufacturer data 0x10
	PairingStateNotDiscoverable PairingState = "NotDiscoverable"
	// PairingStateDiscoverableOnly - discoverable, manufacturer data 0x10
	PairingStateDiscoverableOnly PairingState = "DiscoverableOnly"
	// PairingStatePairStep1 - discoverable, manufacturer data 0x11
	PairingStatePairStep1 PairingState = "PairStep1"
	// PairingStatePairStep2 - discoverable, manufacturer data 0x12
	PairingStatePairStep2 PairingState = "PairStep2"
)
