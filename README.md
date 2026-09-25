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
| `-virtual-mtu` | `23` | Virtual transport only: the ATT MTU to announce and fragment responses to. 23 is the Bluetooth spec minimum and gives the 20-byte notifications pumpX2 and the cliparser jar produce; a real Mobi over iOS negotiates a much larger MTU, so most responses arrive in a single notification. Changeable at runtime via `PUT /api/transport`. |
| `-virtual-ack-after-handling` | `false` | Acknowledge a write only after its handler has run, reproducing the Linux/gatt ordering instead of real-pump ordering (ack first). |
| `-api-addr` | `:8080` | Address of the HTTP/WebSocket API and web UI. |
| `-ui-dir` | `ui` beside the executable, else `./ui` | Where the web UI is served from. |
| `-pump-timezone` | the host's local zone | IANA zone the emulated pump keeps its clock in, e.g. `America/New_York`. A Tandem pump holds local time with no zone attached and every consumer decodes its wire timestamps on that assumption, so this is the zone pump-epoch seconds are encoded in. `UTC` turns the convention off. Changeable at runtime via `PUT /api/clock`. |
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

There are two separate offsets between the emulator's clock and what goes on
the wire, and they mean different things.

**`pump_timezone` is the pump's local-time convention, not a lie.** A Tandem
pump keeps its clock in local time with no zone attached: its pump-epoch
seconds count from 2008-01-01 00:00:00 *as read on the pump's own clock*, and
every consumer decodes on that assumption — TandemKit's
`Dates.fromJan12008ToUnixEpochSeconds` subtracts `TimeZone.current.secondsFromGMT()`.
The emulator therefore encodes with

```
pump_time_seconds = unix(instant) - 1199145600 + pump_timezone_offset_seconds + pump_offset_seconds
```

and a driver in the same zone decodes exactly the original instant back out.
Emitting UTC-based seconds instead (which this emulator used to do) puts every
timestamp one UTC offset from the pump's own record — four hours in EDT —
which is invisible until something compares a dose's second. The zone defaults
to the host's (or `-pump-timezone`); a `PUT` moves the pump to any zone, which
is how a test reproduces a traveling pump or a DST boundary. `UTC` turns the
convention off. `pump_timezone_offset_seconds` is read-only: the zone's offset
in force at `pump_now`, so a test never has to work out which side of a DST
transition the pump is on.

**`pump_offset_seconds` is a lie the pump tells about the time**, not a change
to the emulator's clock: it shifts every timestamp on the wire
(`TimeSinceResetResponse.currentTime`, `CurrentBolusStatus.timestamp`,
`LastBolusStatus*.timestamp`, `TempRateResponse.startTimeRaw`, history
`pumpTimeSec`) while leaving delivery arithmetic untouched. It applies in both
modes and stacks on top of the zone.

**One rule governs every time field the harness API reports:**

- wall-clock fields (`time`, `start_time`, `end_time`, `temp_start`,
  `temp_end`, `cleared_time`) are **true instants** on the pump's Clock, RFC3339
  in UTC, with no skew and no zone folded in;
- `*_pump_seconds` / `pump_seconds` / `pump_time` fields are **what went on the
  wire**, with both offsets applied.

Either is derivable from the other with the two offsets `/api/clock` reports,
and every path — protocol handler, harness action, simulator tick, backdated
staged record — obeys it.

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

# Move the pump's clock to another zone (wire timestamps move, delivery does not)
curl -X PUT http://127.0.0.1:8080/api/clock -d '{"pump_timezone":"America/New_York"}'

# Back to the host wall clock
curl -X PUT http://127.0.0.1:8080/api/clock -d '{"mode":"real","pump_offset_seconds":0}'
```

`GET /api/clock` reports `mode`, `now`, `frozen`, `pump_now`,
`pump_time_seconds`, `pump_offset_seconds`, `pump_timezone`,
`pump_timezone_offset_seconds` and `time_since_reset`. An unknown
`pump_timezone` returns `400`.

`POST /api/clock/advance` returns `409` while the pump is on the real clock.

### State snapshot and direct writes

`GET /api/state` returns everything a driver could observe plus the pump-side
truth behind it: identity, clock, auth and pairing, connection and radio, basal
(profile, current, suspend reason, temp rate), the bolus in progress, the last
bolus record, Control-IQ, reservoir, battery, CGM, standing alerts,
`alarms_history` (every alarm that has been raised *and cleared*, with both
ends of the interval, which `alerts` cannot show), the tail of the history log,
and the request log's current sequence number.

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

### History log

`GET /api/history?since=<sequence>&limit=N` is the full history feed. The
50-record tail in `GET /api/state` is a convenience and stays as it was, but a
scenario that runs for a while loses records off the front of it without being
told; this endpoint does not.

`since` is exclusive — the last sequence you already have — so a poller hands
back the previous page's `next_since` unchanged. `limit` defaults to 200 and is
capped at 5000; `truncated` says the limit cut the page short.

```bash
curl http://127.0.0.1:8080/api/history
curl 'http://127.0.0.1:8080/api/history?since=120&limit=500'
```

Each entry carries:

| Field | Meaning |
|---|---|
| `sequence` | The record's sequence number, as `HistoryLogStatusResponse` reports it. |
| `type_id` / `type` | The pumpX2 numeric type id and its name. |
| `time` | The record's **true instant** on the pump's Clock (RFC3339, UTC). |
| `pump_seconds` | The same instant **as it went on the wire**: pump-epoch seconds with the skew and the pump's zone applied. `pump_time` is the same value under the name the state snapshot has always used. |
| `data` | The record's own **wire fields**, named exactly as pumpX2 and TandemKit name them. Nothing else is in here. |
| `extra` | Pump-side context the 26-byte record format has no room for — a temp rate's delivered volume, the reason string behind a suspend, the alert type behind an alarm. It never reaches the wire, and is kept separate so it cannot be mistaken for a record field. |
| `source_nibble` | The high nibble of the record's type-id word (1 on Mobi captures, 0 on older ones). The driver masks it off. |

Records are written by one writer per type, whichever path produced them, so a
consumer never has to know whether a record came from the protocol handler, a
harness action or a simulator tick. Old field names (`units`, `minutes`,
`unitsDelivered`, `endReasonId`, …) are still accepted as **input** aliases when
staging history, so existing scenario files keep working.

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

### Link parameters (ATT MTU)

`GET /api/transport` reports the ATT MTU in force and what it lets one
notification carry; `PUT` (or `PATCH`) changes it.

The emulator defaults to 23, the Bluetooth spec minimum, which allows 20-byte
notifications: 2 bytes of `[remainingPackets][txId]` framing plus 18 bytes of
message, exactly what pumpX2's `Packetize` and the cliparser jar produce. A real
Mobi paired with the official iOS app negotiates a much larger MTU, so most
responses reach the phone as a **single** notification with `remaining=0`
instead of a run of 18-byte fragments.

Raising the MTU here reproduces that: responses are re-framed to the largest
chunk it allows, with the message bytes unchanged. Both receivers handle either
framing without a special case — TandemKit's `BTResponseParser` and pumpX2's
`PacketArrayList` read the remaining-fragment counter and never a fragment's
length — so this is the knob for exercising a driver's reassembler against both
the fragmented and the unfragmented case.

```bash
curl http://127.0.0.1:8080/api/transport
# {"att_mtu":23,"max_notification_bytes":20,"max_message_bytes_per_notification":18,"mtu_configurable":true}

# What a real Mobi over iOS looks like: most responses in one notification.
curl -X PUT http://127.0.0.1:8080/api/transport -d '{"att_mtu":247}'

# Back to the fragmented default.
curl -X PUT http://127.0.0.1:8080/api/transport -d '{"att_mtu":23}'
```

Only the virtual transport supports this (`mtu_configurable` says so): the Linux
GATT transport lets gatt negotiate the MTU with the connecting central and does
not expose the result, so a `PUT` there returns 501. Set the starting value with
`-virtual-mtu`.

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

# Drop the second fragment of every multi-fragment response on CurrentStatus,
# so the central sees a message that starts and never finishes (virtual only)
curl -X POST http://127.0.0.1:8080/api/faults \
  -d '{"kind":"drop_fragment","characteristic":"CurrentStatus","index":1,"every":true}'

# Or thin the stream: lose every 5th notification fragment, wherever it falls
curl -X POST http://127.0.0.1:8080/api/faults \
  -d '{"kind":"drop_fragment","every_nth":5,"every":true}'

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
(default 1), `every`, `delay_ms`, `error_code`, `after`, `fragments_sent`,
`index`, `every_nth`.
Radio faults are applied immediately rather than stored — "the radio is off" is
a condition, not something that happens to the next message — and return `501`
on a transport that cannot switch its radio (i.e. the real BLE one).

**`drop_fragment`.** The other fault kinds work in whole messages; this one
works in single BLE notifications, which is the loss a real radio actually
produces. Take either `index` — the fragment's position within its own message,
so `0` is the first notification of every message and `1` the second — or
`every_nth`, which drops every Nth fragment seen on the characteristic
regardless of where message boundaries fall. It cannot be scoped by `message`
or `opcode` (a fragment on the wire carries neither; such a request is refused
with `400`), and it returns `501` on a transport that cannot filter
notifications. Arming one restarts the fragment counting. The pump's own
request log still records the response as fully sent — the loss happens below
the router, which is exactly the asymmetry a driver has to cope with.

**`error_response`.** `ErrorResponse` (opcode 77) cannot be produced through
cliparser — pumpX2 has no encodable `ErrorResponse` class — so it is framed by
the native encoder in `pkg/protocol` and sent on `CurrentStatus`, where the
driver looks for it, whatever characteristic the rejected request arrived on.
The cargo is `[requestCodeId][errorCodeId]`: the opcode of the request being
rejected and the `error_code` the fault carries. Clearing
`handler.ErrorResponseEncoder` makes the fault behave as a drop and say so in
the emulator log and in the request log's `note` field, rather than letting a
harness believe it exercised an error path it did not. Note that a driver may
treat an `ErrorResponse` on a control characteristic as no response at all.

### Signed responses

Most `Control` responses — `SuspendPumpingResponse`, `InitiateBolusResponse`,
`SetTempRateResponse`, `BolusPermissionResponse` and the rest of the set pumpX2
marks `signed` — carry a 24-byte trailer: a little-endian `timeSinceReset`
followed by an HMAC-SHA1 over every preceding byte of the message, keyed with
the session key derived during JPAKE.

These are now signed correctly. The cliparser subprocess cannot do it: its
signing inputs arrive only through the environment, and it uses the raw ASCII
bytes of `PUMP_AUTHENTICATION_KEY` as the HMAC key rather than decoding it, so a
binary JPAKE-derived key cannot be handed to it at all — left alone it signs
every signed message with the ASCII of its own `IGNORE_HMAC_SIGNATURE_EXCEPTION`
placeholder, which no driver will accept. The router therefore re-signs: the
cargo cliparser produced is kept verbatim, and only the trailer and the CRC are
rebuilt, with the pump's real key and its current time since reset. Responses
that are not signed go out exactly as cliparser encoded them, byte for byte.

Which messages are signed comes from `pkg/protocol/signed_messages.go`, a table
generated from TandemKit's `MessageProps` declarations. Regenerate it with:

```bash
go test ./pkg/protocol -run TestSignedMessageTableMatchesTandemKit -update
```

That test also runs as a check: with a TandemKit checkout present (beside this
repo, or at `FAKETANDEM_TANDEMKIT_CORE`) it fails if the table and the catalog
have drifted apart in either direction. A jar-gated test in `pkg/pumpx2`
independently cross-checks the table against pumpX2 itself, by encoding the
same message twice under different keys and seeing whether cliparser's output
changes.
