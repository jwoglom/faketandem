package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/bluetooth/virtualtest"
	"github.com/jwoglom/faketandem/pkg/jpake"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// The test binary doubles as faketandem: with FAKETANDEM_E2E_ARGS set it runs
// main() with those flags instead of the tests.
func TestMain(m *testing.M) {
	if args := os.Getenv("FAKETANDEM_E2E_ARGS"); args != "" {
		os.Args = append([]string{os.Args[0]}, strings.Fields(args)...)
		main()
		return
	}
	os.Exit(m.Run())
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }() // Safe to ignore: only the port number is wanted.
	return l.Addr().(*net.TCPAddr).Port
}

// e2ePump is the emulator, run as its own process, and a central on its
// virtual link.
type e2ePump struct {
	t      *testing.T
	client *virtualtest.Client
	codec  *pumpx2.Bridge
	txID   int
}

// startWithoutPumpX2 runs the emulator in its default configuration with an
// empty PATH, so nothing it does can reach a JVM.
func startWithoutPumpX2(t *testing.T) *e2ePump {
	t.Helper()
	gattAddr := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	apiURL := fmt.Sprintf("http://127.0.0.1:%d", freePort(t))

	cmd := exec.Command(os.Args[0]) //nolint:gosec // re-executes this test binary
	cmd.Env = []string{
		"PATH=",
		"FAKETANDEM_E2E_ARGS=-transport virtual -virtual-addr " + gattAddr +
			" -api-addr " + strings.TrimPrefix(apiURL, "http://") + " -pump-timezone UTC -q",
	}
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill() // Safe to ignore: it may have exited already.
		_ = cmd.Wait()         // Safe to ignore: reaps the process.
		if t.Failed() {
			t.Logf("faketandem output:\n%s", logs.String())
		}
	})

	deadline := time.Now().Add(20 * time.Second)
	for {
		resp, err := http.Post(apiURL+"/api/bluetooth/pairingstate", "application/json",
			strings.NewReader(`{"pairingState":"PairStep1"}`))
		if err == nil {
			_ = resp.Body.Close() // Safe to ignore: the status is all that is read.
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("faketandem did not come up: %v\n%s", err, logs.String())
		}
		time.Sleep(100 * time.Millisecond)
	}

	client := virtualtest.Dial(t, gattAddr)
	client.Attach()
	for _, uuid := range []string{bluetooth.AuthorizationCharUUID, bluetooth.CurrentStatusCharUUID, bluetooth.ControlCharUUID} {
		client.Subscribe(uuid, true)
	}
	return &e2ePump{t: t, client: client, codec: pumpx2.NewNativeBridge()}
}

// exchange writes one request as a phone would and reads the reply.
func (p *e2ePump) exchange(char bluetooth.CharacteristicType, uuid, name string, params map[string]interface{}) *pumpx2.ParsedMessage {
	p.t.Helper()
	p.txID++
	request, err := p.codec.EncodeMessage(p.txID, name, params)
	if err != nil {
		p.t.Fatalf("encode %s: %v", name, err)
	}
	for _, fragment := range request.Packets {
		p.client.Write(uuid, fragment)
	}
	var fragments []string
	for {
		got := p.client.CollectNotifications(1, 5*time.Second)
		if len(got) == 0 {
			p.t.Fatalf("no reply to %s (have %v)", name, fragments)
		}
		fragments = append(fragments, got[0])
		if b, _ := hex.DecodeString(got[0]); len(b) > 0 && b[0]&0x0f == 0 {
			break
		}
	}
	reply, err := p.codec.ParseMessage(char, fragments)
	if err != nil || !reply.IsValid {
		p.t.Fatalf("reply to %s does not parse: %v %v", name, fragments, err)
	}
	return reply
}

func cargoBytes(t *testing.T, m *pumpx2.ParsedMessage, key string) []byte {
	t.Helper()
	s, _ := m.Cargo[key].(string)
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("%s.%s is not hex: %v", m.MessageType, key, err)
	}
	return b
}

// TestDefaultConfigurationPairsAndAnswersWithoutPumpX2 pairs over EC-JPAKE
// and reads the pump with no pumpX2 checkout, jar or JVM anywhere.
func TestDefaultConfigurationPairsAndAnswersWithoutPumpX2(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the emulator as a separate process")
	}
	p := startWithoutPumpX2(t)
	auth := bluetooth.CharAuthorization
	authUUID := bluetooth.AuthorizationCharUUID

	client := jpake.New(jpake.Client, jpake.PairingCodeToBytes("123456"), nil)
	round1, err := client.Round1()
	if err != nil {
		t.Fatal(err)
	}
	resp1a := p.exchange(auth, authUUID, "Jpake1aRequest", map[string]interface{}{"appInstanceId": 0, "centralChallenge": hex.EncodeToString(round1[:165])})
	resp1b := p.exchange(auth, authUUID, "Jpake1bRequest", map[string]interface{}{"appInstanceId": 0, "centralChallenge": hex.EncodeToString(round1[165:])})
	serverRound1 := append(cargoBytes(t, resp1a, "centralChallengeHash"), cargoBytes(t, resp1b, "centralChallengeHash")...)
	if _, err := client.ReadRound1(serverRound1); err != nil {
		t.Fatalf("the pump's round 1 does not verify: %v", err)
	}
	round2, err := client.Round2()
	if err != nil {
		t.Fatal(err)
	}
	resp2 := p.exchange(auth, authUUID, "Jpake2Request", map[string]interface{}{"appInstanceId": 0, "centralChallenge": hex.EncodeToString(round2)})
	if _, err := client.ReadRound2(cargoBytes(t, resp2, "centralChallengeHash")); err != nil {
		t.Fatalf("the pump's round 2 does not verify: %v", err)
	}
	secret, err := client.DeriveSecret()
	if err != nil {
		t.Fatal(err)
	}
	resp3 := p.exchange(auth, authUUID, "Jpake3SessionKeyRequest", map[string]interface{}{"cargo": "0000"})
	serverNonce := cargoBytes(t, resp3, "deviceKeyNonce")
	clientNonce := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	resp4 := p.exchange(auth, authUUID, "Jpake4KeyConfirmationRequest", map[string]interface{}{
		"appInstanceId": 0,
		"nonce":         hex.EncodeToString(clientNonce),
		"reserved":      hex.EncodeToString(make([]byte, 8)),
		"hashDigest":    hex.EncodeToString(jpake.KeyConfirmation(serverNonce, secret, clientNonce)),
	})
	if !bytes.Equal(cargoBytes(t, resp4, "hashDigest"), jpake.KeyConfirmation(serverNonce, secret, cargoBytes(t, resp4, "nonce"))) {
		t.Fatal("the pump's key confirmation does not verify")
	}

	status := bluetooth.CharCurrentStatus
	statusUUID := bluetooth.CurrentStatusCharUUID
	if reply := p.exchange(status, statusUUID, "ApiVersionRequest", nil); reply.MessageType != "ApiVersionResponse" {
		t.Errorf("ApiVersionRequest answered with %s", reply.MessageType)
	}
	basal := p.exchange(status, statusUUID, "CurrentBasalStatusRequest", nil)
	if basal.MessageType != "CurrentBasalStatusResponse" {
		t.Fatalf("CurrentBasalStatusRequest answered with %s", basal.MessageType)
	}
	if rate, _ := basal.Cargo["profileBasalRate"].(int); rate <= 0 {
		t.Errorf("profileBasalRate %v, want the emulator's non-zero profile rate", basal.Cargo["profileBasalRate"])
	}
	if reply := p.exchange(status, statusUUID, "TimeSinceResetRequest", nil); reply.MessageType != "TimeSinceResetResponse" {
		t.Errorf("TimeSinceResetRequest answered with %s", reply.MessageType)
	}
}
