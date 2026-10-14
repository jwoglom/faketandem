package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// mobiSoftwareInfo is what a Mobi's answer parses to: the same software ID
// and part number for the app and bootloader, at revision 0.
func mobiSoftwareInfo(mcuType currentstatus.MCUType, softwareID string, partNumber int64, cargo []byte) currentstatus.CommonSoftwareInfoResponse {
	mcuTypeID := mcuType.ID()
	var revision int64
	m := currentstatus.CommonSoftwareInfoResponse{
		MCUTypeID:                    &mcuTypeID,
		AppSoftwareVersion:           softwareID,
		AppSoftwarePartNumber:        partNumber,
		BootloaderVersion:            softwareID,
		BootloaderPartNumber:         partNumber,
		BootloaderPartRevisionNumber: &revision,
	}
	m.SetCargo(cargo)
	return m
}

func TestCommonSoftwareInfoResponse(t *testing.T) {
	// A Tandem Mobi's answer for its ARM MCU on Control-IQ 7.7.0.1.
	cargo, _ := hex.DecodeString("006663383265343935646465383266636600167a0f00000000006663383265343935646465383266636600167a0f0000000000")
	for _, tc := range []struct {
		name string
		hex  string
		txID int
	}{
		{"Mobi 7.7", "002f8f2f33006663383265343935646465383266636600167a0f00000000006663383265343935646465383266636600167a0f00000000000457", 47},
		{"Mobi API 3.6", "00398f3933006663383265343935646465383266636600167a0f00000000006663383265343935646465383266636600167a0f0000000000a739", 57},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := currentstatus.NewCommonSoftwareInfoResponseFromCargo(cargo)
			if err != nil {
				t.Fatal(err)
			}
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 4, messages.CurrentStatus, expected).(*currentstatus.CommonSoftwareInfoResponse)
			if want := mobiSoftwareInfo(currentstatus.MCUTypeARM, "fc82e495dde82fcf", 1014294, cargo); !reflect.DeepEqual(*parsed, want) {
				t.Errorf("parsed %s, want %s", messages.ToString(parsed), messages.ToString(&want))
			}
			if mcuType := parsed.MCUType(); mcuType == nil || *mcuType != currentstatus.MCUTypeARM {
				t.Errorf("MCU type %v, want ARM", mcuType)
			}
		})
	}
}

func TestCommonSoftwareInfoResponseBuildsTheMobiLayout(t *testing.T) {
	for _, tc := range []struct {
		name       string
		hex        string
		mcuType    currentstatus.MCUType
		softwareID string
		partNumber int64
	}{
		{"Mobi 7.9.0.2, BLE", "016433633931616435346331326232323800b3870f00000000006433633931616435346331326232323800b3870f0000000000",
			currentstatus.MCUTypeBLE, "d3c91ad54c12b228", 1017779},
		{"Mobi 7.9.0.1, ARM", "0034323064643130663137343365623765000b830f000000000034323064643130663137343365623765000b830f0000000000",
			currentstatus.MCUTypeARM, "420dd10f1743eb7e", 1016587},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := hex.DecodeString(tc.hex)
			parsed, err := currentstatus.NewCommonSoftwareInfoResponseFromCargo(raw)
			if err != nil {
				t.Fatal(err)
			}
			if want := mobiSoftwareInfo(tc.mcuType, tc.softwareID, tc.partNumber, raw); !reflect.DeepEqual(*parsed, want) {
				t.Errorf("parsed %s, want %s", messages.ToString(parsed), messages.ToString(&want))
			}
			built, err := currentstatus.NewMobiCommonSoftwareInfoResponse(tc.mcuType.ID(), tc.softwareID, tc.partNumber, 0, tc.softwareID, tc.partNumber, 0)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(built.Cargo(), raw) {
				t.Errorf("built %x, want %x", built.Cargo(), raw)
			}
		})
	}
}

func TestCommonSoftwareInfoResponseX2LayoutRoundTrips(t *testing.T) {
	built, err := currentstatus.NewCommonSoftwareInfoResponse("0123456789abcdef", 1000001, 2, 3, "fedcba9876543210", 1000004, 5, 6)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := currentstatus.NewCommonSoftwareInfoResponseFromCargo(built.Cargo())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, built) {
		t.Errorf("parsed %s, want %s", messages.ToString(parsed), messages.ToString(built))
	}
}

func TestCommonSoftwareInfoResponseRefusesAVersionTooLongForItsField(t *testing.T) {
	if _, err := currentstatus.NewMobiCommonSoftwareInfoResponse(0, "0123456789abcdef01", 1, 0, "", 1, 0); err == nil {
		t.Error("an 18-byte version was put in a 17-byte field")
	}
}

func TestCommonSoftwareInfoResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CommonSoftwareInfoResponse")
}
