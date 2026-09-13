# faketandem

Fake Tandem pump implementation, based on the 'pod' OmniPod simulator, for Raspberry Pi 3+.

## Virtual transport (no radio)

The emulator normally acts as a real BLE peripheral, which needs a Bluetooth
adapter and the Linux-only GATT stack. The `virtual` transport replaces the
radio with a TCP link that carries GATT semantics — service discovery, reads,
writes, subscriptions and notifications — as newline-delimited JSON. A client
with its own BLE stack can be pointed at it one layer below CoreBluetooth, and
the emulator runs unchanged on a Mac or in CI.

`-transport` defaults to `ble` on Linux and `virtual` everywhere else. The wire
protocol is documented in [docs/virtual-gatt-protocol.md](docs/virtual-gatt-protocol.md).

Fetch the pumpX2 cliparser jar (into `third_party/`) and run:

```bash
make jar
go run . -transport virtual \
  -pumpx2-jar-path third_party/pumpx2-cliparser-1.9.1.jar \
  -q
```

The virtual link then listens on `127.0.0.1:8081` and the web UI and HTTP API on
`http://127.0.0.1:8080/ui/`.

Like a real pump, the emulator starts in `NotDiscoverable` and refuses
connections until the pairing state is set:

```bash
curl -X POST http://127.0.0.1:8080/api/bluetooth/pairingstate \
  -d '{"pairingState":"PairStep1"}'
```

Flags:

| Flag | Default | Purpose |
|---|---|---|
| `-transport` | `ble` on Linux, `virtual` elsewhere | Which link to use. |
| `-virtual-addr` | `127.0.0.1:8081` | Address the virtual link listens on. |
| `-virtual-peripheral-id` | derived from the pump serial number | Stable peripheral UUID reported in `hello`. |
| `-virtual-ack-after-handling` | `false` | Acknowledge a write only after its handler has run, reproducing the Linux/gatt ordering instead of real-pump ordering (ack first). |
| `-api-addr` | `:8080` | Address of the HTTP/WebSocket API and web UI. |
| `-ui-dir` | `ui` beside the executable, else `./ui` | Where the web UI is served from. |
