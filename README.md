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
| `-request-log-size` | `4096` | How many messages the harness request log (`GET /api/log`) retains. |

## Integration harness API

An automated conformance suite driving a driver against this emulator needs
three things the protocol itself cannot give it: the pump's own record of the
exchange, the ability to make the pump do something the driver did not command,
and the ability to make a message fail in a chosen way. These endpoints provide
them. **Everything here is inert until used** — with nothing armed and no clock
set, the emulator behaves exactly as it did before.

All examples assume the API on `http://127.0.0.1:8080`.

### Clock

Pump time is driven by a controllable clock. In `manual` mode a test can freeze
it, step it, and skew the pump's own clock away from it — so a bolus that takes
50 s of pump time finishes in one call rather than in 50 s of wall time, and a
driver assertion about a dose's start or end second is exact rather than
"within a tolerance".

`pump_offset_seconds` is a lie the pump tells about the time, not a change to
the emulator's clock: it shifts every timestamp on the wire
(`TimeSinceResetResponse.currentTime`, `CurrentBolusStatus.timestamp`,
`LastBolusStatus*.timestamp`, `TempRateResponse.startTimeRaw`, history
`pumpTimeSec`) while leaving delivery arithmetic untouched. It applies in both
modes.

```bash
# What the clock is doing now
curl http://127.0.0.1:8080/api/clock

# Freeze the pump at a chosen instant, with its own clock running 8 s fast
curl -X PUT http://127.0.0.1:8080/api/clock \
  -d '{"mode":"manual","now":"2024-03-05T12:00:00Z","frozen":true,"pump_offset_seconds":8}'

# Step 30 s of pump time; the simulator runs with it
curl -X POST http://127.0.0.1:8080/api/clock/advance -d '{"seconds":30}'

# Step the same 30 s in six pieces, so anything that has to happen partway
# through the jump (a temp rate expiring mid-bolus) happens in order
curl -X POST http://127.0.0.1:8080/api/clock/advance -d '{"seconds":30,"ticks":6}'

# Back to the host wall clock
curl -X PUT http://127.0.0.1:8080/api/clock -d '{"mode":"real","pump_offset_seconds":0}'
```

`POST /api/clock/advance` returns `409` while the pump is on the real clock.

### State snapshot and direct writes

`GET /api/state` returns everything a driver could observe plus the pump-side
truth behind it: identity, clock, auth and pairing, connection and radio, basal
(profile, current, suspend reason, temp rate), the bolus in progress, the last
bolus record, Control-IQ, reservoir, battery, CGM, alerts, the tail of the
history log, and the request log's current sequence number.

`PUT` (or `PATCH`) sets named fields and nothing else. It is deliberately dumb:
it raises no qualifying events and writes no history, because setting a field is
for staging a starting position (a pump that was already suspended before the
driver connected). For changes that must look to the driver like the pump doing
something, use the actions below.

```bash
curl http://127.0.0.1:8080/api/state

curl -X PUT http://127.0.0.1:8080/api/state \
  -d '{"reservoir_units":42.5,"battery_percent":17,"basal_rate":1.0,"suspended":true}'
```

Settable fields: `reservoir_units`, `battery_percent`, `battery_charging`,
`basal_rate`, `suspended`, `iob`, `tdd`, `closed_loop_enabled`,
`control_iq_mode`, `weight`, `total_daily_insulin`,
`bolus_rate_units_per_second`, `cgm_egv`, `cgm_session_active`, `pairing_code`,
`time_since_reset`, `api_version_major`, `api_version_minor`, `clear_alerts`.

### Pump-initiated actions

Each action moves every driver-visible response consistently and raises the
qualifying events a real pump would raise. A suspend is not a flag: it halts an
in-progress bolus and any running temp rate, records each ending in the history
log, raises an alarm when the reason is one, and emits a qualifying event for
each — the pile-up a driver's reconciliation actually has to survive.

```bash
# A bolus the pump started itself (source id 0, not the app's 8), taking 50 s
curl -X POST http://127.0.0.1:8080/api/state/bolus/start \
  -d '{"units":2.5,"duration_seconds":50}'
# ... or at an explicit delivery rate, with a chosen id and the app as source
curl -X POST http://127.0.0.1:8080/api/state/bolus/start \
  -d '{"units":2.5,"rate":0.05,"bolus_id":77,"source":"remote"}'

# Hold it open: still in progress, delivering nothing
curl -X POST http://127.0.0.1:8080/api/state/bolus/stall
curl -X POST http://127.0.0.1:8080/api/state/bolus/resume

# End it, cut short or in full (both write the LastBolusStatus record)
curl -X POST http://127.0.0.1:8080/api/state/bolus/abort
curl -X POST http://127.0.0.1:8080/api/state/bolus/complete
curl -X POST http://127.0.0.1:8080/api/state/bolus/abort -d '{"delivered_units":1.2}'

# Temp rate, by percentage of the profile rate or as an absolute U/hr
curl -X POST http://127.0.0.1:8080/api/state/tempbasal/start \
  -d '{"percent":150,"duration_minutes":30}'
curl -X POST http://127.0.0.1:8080/api/state/tempbasal/start \
  -d '{"rate":1.2,"duration_minutes":30}'
curl -X POST http://127.0.0.1:8080/api/state/tempbasal/stop

# Stop and restart delivery. reason: user | occlusion | alarm
curl -X POST http://127.0.0.1:8080/api/state/suspend -d '{"reason":"occlusion"}'
curl -X POST http://127.0.0.1:8080/api/state/resume
curl -X POST http://127.0.0.1:8080/api/state/resume -d '{"clear_alarms":false}'

# Stage history that predates the connection
curl -X POST http://127.0.0.1:8080/api/state/history/append \
  -d '{"type":"BolusCompleted","seconds_ago":3600,"data":{"bolusId":9,"unitsDelivered":1.5}}'

# Raise an arbitrary qualifying-event bitmask (here: remainingInsulin)
curl -X POST http://127.0.0.1:8080/api/state/qualifyingevent -d '{"bitmask":262144}'
```

Every action returns the full state snapshot, so a test rarely needs a second
call. An action that cannot apply (a second concurrent bolus, resuming an
unsuspended pump) returns `409`.

### Request log

The pump's own record of the exchange: every parsed inbound message with its
cargo, every outbound response with its fragments and **how many of them
actually went out**, every qualifying-event notification, every connection
change and every armed fault. It is the pump-side equivalent of a driver's
sent-message spy, and the only way to tell "the pump never answered" apart from
"the driver ignored the answer".

Entries live in a ring buffer with monotonically increasing sequence numbers.
`oldest_seq` in the response is the oldest record still held: a poller whose
`since` is below it lost records to the ring's turnover.

```bash
curl http://127.0.0.1:8080/api/log
curl 'http://127.0.0.1:8080/api/log?since=42'
curl -X DELETE http://127.0.0.1:8080/api/log
```

### Fault injection

Faults are armed against the next matching message, or against every match.
Scope one with `message` (either half of a request/response pair names the same
exchange, case-insensitively), `opcode` and/or `characteristic`; leave them all
out and the fault applies to the next message of any kind.

```bash
# The pump acts on the next SetTempRate and the answer never arrives.
# State changes are applied BEFORE the drop decision, so this is genuinely
# "applied but the response was lost", not "nothing happened".
curl -X POST http://127.0.0.1:8080/api/faults \
  -d '{"kind":"drop_response","message":"SetTempRateRequest"}'

# Hold every CurrentBolusStatus answer back by 2 s
curl -X POST http://127.0.0.1:8080/api/faults \
  -d '{"kind":"delay_response","message":"CurrentBolusStatusRequest","delay_ms":2000,"every":true}'

# Answer the next two with an ErrorResponse instead (see the caveat below)
curl -X POST http://127.0.0.1:8080/api/faults \
  -d '{"kind":"error_response","message":"InitiateBolusRequest","error_code":3,"count":2}'

# Cut the link instead of answering at all
curl -X POST http://127.0.0.1:8080/api/faults \
  -d '{"kind":"disconnect","message":"HistoryLogRequest","after":"request"}'

# Cut it after exactly 2 fragments of a multi-fragment response
curl -X POST http://127.0.0.1:8080/api/faults \
  -d '{"kind":"disconnect","message":"BolusCalcDataSnapshotRequest","after":"partial_response","fragments_sent":2}'

# Refuse the next quick-pair reconnect even though a long-term key is cached,
# forcing the client back into a full pairing
curl -X POST http://127.0.0.1:8080/api/faults -d '{"kind":"reject_quick_pair"}'

# Radio off and on (virtual transport only): drops the central and refuses
# new connections until it is restored
curl -X POST http://127.0.0.1:8080/api/faults -d '{"kind":"radio_off"}'
curl -X POST http://127.0.0.1:8080/api/faults -d '{"kind":"radio_on"}'

# List, clear one, clear all
curl http://127.0.0.1:8080/api/faults
curl -X DELETE http://127.0.0.1:8080/api/faults/3
curl -X DELETE http://127.0.0.1:8080/api/faults
```

Fault fields: `kind` (required), `message`, `opcode`, `characteristic`, `count`
(default 1), `every`, `delay_ms`, `error_code`, `after`, `fragments_sent`.
Radio faults are applied immediately rather than stored — "the radio is off" is
a condition, not something that happens to the next message — and return `501`
on a transport that cannot switch its radio (i.e. the real BLE one).

**`error_response` caveat.** `ErrorResponse` (opcode 77) cannot be produced
through cliparser: pumpX2 has no encodable `ErrorResponse` class, so the bytes
have to be framed natively. The router calls the
`handler.ErrorResponseEncoder` hook, which the native packet encoder fills in;
until it does, an armed `error_response` fault behaves as a drop and says so in
the emulator log and in the request log's `note` field, rather than letting a
harness believe it exercised an error path it did not. Note also that a driver
may treat an `ErrorResponse` on a control characteristic as no response at all.
