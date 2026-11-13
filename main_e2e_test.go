package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
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
	apiURL string
	// events are the qualifying-event bitmasks notified so far, set aside by
	// exchange while it waits for a reply.
	events []uint32
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
	return &e2ePump{t: t, client: client, codec: pumpx2.NewNativeBridge(), apiURL: apiURL}
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
		got, ok := p.nextNotification(uuid, 5*time.Second)
		if !ok {
			p.t.Fatalf("no reply to %s (have %v)", name, fragments)
		}
		fragments = append(fragments, got)
		if b, _ := hex.DecodeString(got); len(b) > 0 && b[0]&0x0f == 0 {
			break
		}
	}
	reply, err := p.codec.ParseMessage(char, fragments)
	if err != nil || !reply.IsValid {
		p.t.Fatalf("reply to %s does not parse: %v %v", name, fragments, err)
	}
	return reply
}

// nextNotification reads the next notification on uuid, setting qualifying
// events aside in p.events.
func (p *e2ePump) nextNotification(uuid string, wait time.Duration) (string, bool) {
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		p.client.SetTimeout(time.Until(deadline))
		msg, err := p.client.TryRecv()
		p.client.SetTimeout(virtualtest.DefaultTimeout)
		if err != nil {
			return "", false
		}
		if msg.Type != "notify" {
			continue
		}
		if strings.EqualFold(msg.Characteristic, bluetooth.QualifyingEventsCharUUID) {
			if b, err := hex.DecodeString(msg.Value); err == nil && len(b) == 4 {
				p.events = append(p.events, uint32(b[0])|uint32(b[1])<<8|uint32(b[2])<<16|uint32(b[3])<<24)
			}
			continue
		}
		if strings.EqualFold(msg.Characteristic, uuid) {
			return msg.Value, true
		}
	}
	return "", false
}

// pair runs EC-JPAKE with the pairing code 123456 and returns the derived
// secret.
func (p *e2ePump) pair() []byte {
	t := p.t
	t.Helper()
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

	return secret
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
	p.pair()

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

// TestFillTubingButtonStreamsToTheApp holds the pump's button through the HTTP
// API in fill tubing mode and reads the press and the release off
// CONTROL_STREAM, signed with the session key.
func TestFillTubingButtonStreamsToTheApp(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the emulator as a separate process")
	}
	p := startWithoutPumpX2(t)
	secret := p.pair()
	// The session key is the derived secret's hex, as ASCII, which is how pumpX2 and TandemKit key the HMAC.
	sessionKey := []byte(hex.EncodeToString(secret))
	p.client.Subscribe(bluetooth.ControlStreamCharUUID, true)

	if code := p.post("/api/state/filltubing/button", `{"down":true}`); code != http.StatusConflict {
		t.Fatalf("pressing before fill tubing mode answered %d, want 409", code)
	}

	enter := p.exchange(bluetooth.CharControl, bluetooth.ControlCharUUID, "EnterFillTubingModeRequest", nil)
	if status, _ := enter.Cargo["status"].(int); status != 0 {
		t.Fatalf("EnterFillTubingModeRequest answered %v", enter.Cargo)
	}
	enterTxID := byte(p.txID)

	if code := p.post("/api/state/filltubing/button", `{"hold_seconds":0.3}`); code != http.StatusOK {
		t.Fatalf("holding the button answered %d", code)
	}

	states := []int32{p.readFillTubingButtonState(sessionKey, enterTxID), p.readFillTubingButtonState(sessionKey, enterTxID)}
	if states[0] != 1 || states[1] != 0 {
		t.Errorf("button states %v, want a press then a release", states)
	}
}

// readFillTubingButtonState reads the next CONTROL_STREAM message, which must be
// a FillTubingStateStreamResponse under txID signed with sessionKey.
func (p *e2ePump) readFillTubingButtonState(sessionKey []byte, txID byte) int32 {
	t := p.t
	t.Helper()
	payload := p.readStream(sessionKey, protocol.OpcodeFillTubingStateStreamResponse, txID)
	response, err := controlstream.NewFillTubingStateStreamResponseFromCargo(payload)
	if err != nil {
		t.Fatal(err)
	}
	return response.ButtonState
}

// readStream reads the next CONTROL_STREAM message, which must be opcode under
// txID signed with sessionKey, and returns its payload, signature and all.
func (p *e2ePump) readStream(sessionKey []byte, opcode, txID byte) []byte {
	t := p.t
	t.Helper()
	var body []byte
	for {
		got := p.client.CollectNotifications(1, 5*time.Second)
		if len(got) == 0 {
			t.Fatalf("no stream message with opcode %#x arrived", opcode)
		}
		fragment, _ := hex.DecodeString(got[0])
		body = append(body, fragment[2:]...)
		if fragment[0] == 0 {
			break
		}
	}
	if _, ok, err := protocol.VerifySignedMessage(body, sessionKey); err != nil || !ok {
		t.Errorf("stream message %x is not signed with the session key (%v)", body, err)
	}
	parsed, err := protocol.ParseMessageBody(body)
	if err != nil || parsed.Opcode != opcode || parsed.TxID != txID {
		t.Fatalf("stream message %x (%v), want opcode %#x under txID %d", body, err, opcode, txID)
	}
	return parsed.Payload
}

// readStreamCargo is readStream without the signature.
func (p *e2ePump) readStreamCargo(sessionKey []byte, opcode, txID byte) string {
	payload := p.readStream(sessionKey, opcode, txID)
	return hex.EncodeToString(payload[:len(payload)-protocol.SignedTrailerLength])
}

// TestCartridgeChangeRunsEndToEnd changes the cartridge, fills the tubing and
// fills the cannula as TandemKit's wizards do, reading each procedure's stream
// off CONTROL_STREAM, signed with the session key, under its request's txID.
func TestCartridgeChangeRunsEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the emulator as a separate process")
	}
	p := startWithoutPumpX2(t)
	sessionKey := []byte(hex.EncodeToString(p.pair()))
	p.client.Subscribe(bluetooth.ControlStreamCharUUID, true)
	if code := p.request(http.MethodPut, "/api/state", `{"suspended":true,"reservoir_units":20,"cartridge_ready_seconds":0.2,
		"cartridge_detect_seconds":0.2,"bolus_rate_units_per_second":1}`); code != http.StatusOK {
		t.Fatalf("staging answered %d", code)
	}

	p.changeCartridge(sessionKey)
	p.fillTubing(sessionKey)

	cannula := p.control("FillCannulaRequest", map[string]interface{}{"primeSizeMilliUnits": 300})
	fillState := protocol.OpcodeFillCannulaStateStreamResponse
	if got := []string{p.readStreamCargo(sessionKey, fillState, cannula), p.readStreamCargo(sessionKey, fillState, cannula)}; got[0] != "00" || got[1] != "02" {
		t.Errorf("fill cannula states %v, want filling then cannulaFilled", got)
	}

	p.checkCartridgeChanged()
}

// control sends a request on CONTROL, which must be accepted, and returns its txID.
func (p *e2ePump) control(name string, params map[string]interface{}) byte {
	p.t.Helper()
	reply := p.exchange(bluetooth.CharControl, bluetooth.ControlCharUUID, name, params)
	if status, _ := reply.Cargo["status"].(int); status != 0 {
		p.t.Fatalf("%s answered %v", name, reply.Cargo)
	}
	return byte(p.txID)
}

func (p *e2ePump) changeCartridge(sessionKey []byte) {
	t := p.t
	t.Helper()
	enter := p.control("EnterChangeCartridgeModeRequest", nil)
	changeState := protocol.OpcodeEnterChangeCartridgeModeStateStreamResponse
	if got := []string{p.readStreamCargo(sessionKey, changeState, enter), p.readStreamCargo(sessionKey, changeState, enter)}; got[0] != "00" || got[1] != "02" {
		t.Errorf("change cartridge states %v, want retracting then readyToChange", got)
	}
	if code := p.request(http.MethodPost, "/api/state/cartridge/insert", `{"units":250}`); code != http.StatusOK {
		t.Fatalf("inserting the cartridge answered %d", code)
	}

	exit := p.control("ExitChangeCartridgeModeRequest", nil)
	for _, percent := range []int32{0, 20, 40, 60, 80, 100} {
		payload := p.readStream(sessionKey, protocol.OpcodeDetectingCartridgeStateStreamResponse, exit)
		response, err := controlstream.NewDetectingCartridgeStateStreamResponseFromCargo(payload)
		if err != nil || response.PercentComplete != percent {
			t.Fatalf("cartridge check %v (%v), want %d%%", response, err, percent)
		}
	}
	load := p.exchange(bluetooth.CharCurrentStatus, bluetooth.CurrentStatusCharUUID, "LoadStatusRequest", nil)
	if load.Cargo["isLoadingActive"] != true || load.Cargo["loadStateId"] != 1 {
		t.Errorf("LoadStatusResponse %v, want LOAD_CARTRIDGE active", load.Cargo)
	}
}

func (p *e2ePump) fillTubing(sessionKey []byte) {
	t := p.t
	t.Helper()
	enter := p.control("EnterFillTubingModeRequest", nil)
	if code := p.request(http.MethodPost, "/api/state/filltubing/button", `{"hold_seconds":0.3}`); code != http.StatusOK {
		t.Fatalf("holding the button answered %d", code)
	}
	p.readFillTubingButtonState(sessionKey, enter)
	p.readFillTubingButtonState(sessionKey, enter)
	exit := p.control("ExitFillTubingModeRequest", nil)
	if got := p.readStreamCargo(sessionKey, protocol.OpcodeExitFillTubingModeStateStreamResponse, exit); got != "01" {
		t.Errorf("exit fill tubing state %s, want TUBING_FILLED", got)
	}
}

// checkCartridgeChanged reads the pump's own account of the change.
func (p *e2ePump) checkCartridgeChanged() {
	t := p.t
	t.Helper()
	var snapshot struct {
		Insulin struct {
			ReservoirUnits float64 `json:"reservoir_units"`
		} `json:"insulin"`
		Cartridge struct {
			LoadState string `json:"load_state"`
		} `json:"cartridge"`
		History struct {
			Entries []struct {
				Type string `json:"type"`
			} `json:"entries"`
		} `json:"history"`
	}
	p.getJSON("/api/state", &snapshot)
	if units := snapshot.Insulin.ReservoirUnits; units >= 249.7 || units < 240 {
		t.Errorf("reservoir %v U, want the 250 U cartridge less the tubing and 0.3 U cannula fills", units)
	}
	if snapshot.Cartridge.LoadState != "none" {
		t.Errorf("load state %q after the cannula fill, want none", snapshot.Cartridge.LoadState)
	}
	var records []string
	for _, e := range snapshot.History.Entries {
		switch e.Type {
		case "CartridgeFilled", "TubingFilled", "CannulaFilled":
			records = append(records, e.Type)
		}
	}
	if want := "CartridgeFilled,TubingFilled,CannulaFilled"; strings.Join(records, ",") != want {
		t.Errorf("records %v, want %s", records, want)
	}
}

func (p *e2ePump) request(method, path, body string) int {
	p.t.Helper()
	req, err := http.NewRequest(method, p.apiURL+path, strings.NewReader(body))
	if err != nil {
		p.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		p.t.Fatal(err)
	}
	_ = resp.Body.Close() // Safe to ignore: the status is all that is read.
	return resp.StatusCode
}

func (p *e2ePump) getJSON(path string, v interface{}) {
	p.t.Helper()
	resp, err := http.Get(p.apiURL + path)
	if err != nil {
		p.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }() // Safe to ignore: the body has been read.
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		p.t.Fatal(err)
	}
}

func (p *e2ePump) post(path, body string) int {
	p.t.Helper()
	resp, err := http.Post(p.apiURL+path, "application/json", strings.NewReader(body))
	if err != nil {
		p.t.Fatal(err)
	}
	_ = resp.Body.Close() // Safe to ignore: the status is all that is read.
	return resp.StatusCode
}
