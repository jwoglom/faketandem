package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake1aResponse(t *testing.T) {
	for _, tc := range []struct {
		name          string
		cargo         string
		hex           string
		appInstanceID int32
	}{
		{
			name:  "capture",
			cargo: "000041048139ce7f5012e2c32c8be4a3eb4511f9bd1c471bed0f1ccf623a2a0399e4f7de35e00c2ae0d8b42d173183ed624b276caf83bb68ce665f1acab03b758056dbca41047a0e04939c0089e44de6268e0018c390c5fb1d4d832a52fd67dcd003d31fd576ff3eb7a838d0b389a0d2544fc740119aced931ac6385ab8ca620e0756d17f5fb20b570ea8e5460cc45b1b733c2edb2bc32a206f1aab956da044e01ba1be6d09913",
			hex:   "00002100a7000041048139ce7f5012e2c32c8be4a3eb4511f9bd1c471bed0f1ccf623a2a0399e4f7de35e00c2ae0d8b42d173183ed624b276caf83bb68ce665f1acab03b758056dbca41047a0e04939c0089e44de6268e0018c390c5fb1d4d832a52fd67dcd003d31fd576ff3eb7a838d0b389a0d2544fc740119aced931ac6385ab8ca620e0756d17f5fb20b570ea8e5460cc45b1b733c2edb2bc32a206f1aab956da044e01ba1be6d099132562",
		},
		{
			name:          "app instance 1",
			cargo:         "0100410441e099e06fda4ba1ea6b6e727e4790fee1303018a97a8a1eed5093f2dab4cbced4ac1f392fbff14a89cc7b8ee06fdadd09d4b67222e1af69612ae3b727679ae94104c9dd154dd328e468cda64f39d6c5e5dfc6645fa60883cee41ca536d22fd6f5ddf12ed045c8eff2edf59f13249268cf403e19585711e8393872954632e896fbb820a8c41909ae5120fa17fc0716ca65dc5fb440e94b78ce9c82c997a0fc4f855d41",
			hex:           "00002100a70100410441e099e06fda4ba1ea6b6e727e4790fee1303018a97a8a1eed5093f2dab4cbced4ac1f392fbff14a89cc7b8ee06fdadd09d4b67222e1af69612ae3b727679ae94104c9dd154dd328e468cda64f39d6c5e5dfc6645fa60883cee41ca536d22fd6f5ddf12ed045c8eff2edf59f13249268cf403e19585711e8393872954632e896fbb820a8c41909ae5120fa17fc0716ca65dc5fb440e94b78ce9c82c997a0fc4f855d416119",
			appInstanceID: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cargo := decodeHex(t, tc.cargo)
			expected, err := authentication.NewJpake1aResponseFromCargo(cargo)
			if err != nil {
				t.Fatal(err)
			}
			parsed := parity.TestMessage(t, tc.hex, 0, 10, messages.Authorization, expected).(*authentication.Jpake1aResponse)
			if parsed.AppInstanceID != tc.appInstanceID || !bytes.Equal(parsed.CentralChallengeHash, cargo[2:]) {
				t.Errorf("app instance ID %d, round %x", parsed.AppInstanceID, parsed.CentralChallengeHash)
			}
		})
	}
}

func TestJpake1aResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake1aResponse")
}
