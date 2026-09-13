# Virtual GATT link protocol (faketandem <-> TandemKit), v1

Purpose: let TandemKit's real BLE stack (TandemBluetoothManager / TandemPeripheralManager) talk to
faketandem without a radio, one layer below CoreBluetooth. Carries GATT semantics only. Pump messages
are opaque bytes.

## Transport
- TCP, one connection == one BLE connection. faketandem listens (default 127.0.0.1:8081, flag
  `-virtual-addr`). TandemKit connects.
- Newline-delimited JSON (one object per line, UTF-8, `\n` terminated). Binary as lowercase hex.
- Every message has `"type"`. Requests from the central carry an integer `"id"`; the matching
  reply echoes it. Unsolicited peripheral messages (`notify`, `disconnect`) have no id.
- Field names are snake_case.

## Connection handshake
1. Central opens TCP. Peripheral immediately sends `hello`:
   {"type":"hello","protocol_version":1,"peripheral_id":"<UUID string, stable across runs>",
    "name":"tslim X2 ***123","manufacturer_data":"<hex, same bytes as BLE advertisement>",
    "pairing_state":"NotDiscoverable|DiscoverableOnly|PairStep1|PairStep2",
    "services":[{"uuid":"0000fdfb-0000-1000-8000-00805f9b34fb",
                 "characteristics":[{"uuid":"7b83fff6-...","properties":["write","notify"]}, ...]},
                {"uuid":"0000180a-...","characteristics":[{"uuid":"00002a29-...","properties":["read"]}, ...]}]}
   UUIDs are full 128-bit lowercase strings (expand 16-bit ones to 0000xxxx-0000-1000-8000-00805f9b34fb).
2. If the pump is not connectable (pairing state NotDiscoverable), the peripheral sends
   {"type":"disconnect","reason":"not_discoverable"} after hello and closes. This mirrors the Linux
   behavior of rejecting connections in that state. A central may open a connection purely to read
   `hello` as a stand-in for a scan/advertisement; closing right after hello is allowed and does not
   count as a BLE connection (the peripheral fires its connection handler only after the central
   sends `attach`).
3. Central sends {"type":"attach","id":1} to declare it is a real connection. Peripheral replies
   {"type":"attach_ack","id":1} and fires its ConnectionHandler(true).

## Operations (central -> peripheral, each answered)
- {"type":"write","id":N,"characteristic":"<uuid>","value":"<hex>","with_response":true}
  -> {"type":"write_ack","id":N} (or {"type":"write_ack","id":N,"error":"<string>"})
  The ack is sent IMMEDIATELY on receipt (real-pump ordering: ack before any response notification).
  Handling is dispatched asynchronously. Flag `-virtual-ack-after-handling` reproduces the Linux/gatt
  ordering (handler runs synchronously, ack after notify) for parity testing.
- {"type":"read","id":N,"characteristic":"<uuid>"}
  -> {"type":"read_result","id":N,"value":"<hex>"} or with "error".
- {"type":"subscribe","id":N,"characteristic":"<uuid>","enabled":true}
  -> {"type":"subscribe_ack","id":N}. Notifications for a characteristic are delivered only while
  subscribed (matches CoreBluetooth). Notifies attempted while unsubscribed are dropped and logged.
- {"type":"disconnect","id":N} -> peripheral closes socket; ConnectionHandler(false).

## Peripheral -> central, unsolicited
- {"type":"notify","characteristic":"<uuid>","value":"<hex>"}  (BLE notify, never indicate;
  value is exactly one ATT notification payload, i.e. one fragment as the pump would send it)
- {"type":"disconnect","reason":"<string>"} followed by socket close (pump-initiated drop,
  e.g. ShutdownConnection / fault injection).

## Sizes and the ATT MTU
The link is transparent: a `write` value is delivered as one ATT write of that exact length; a `notify`
is one notification.

The peripheral announces the ATT MTU in force as `att_mtu` on the `hello` message. It defaults to 23,
the Bluetooth spec minimum, which allows 20-byte notifications: 2 bytes of `[remainingPackets][txId]`
framing plus 18 bytes of message, exactly what pumpX2's `Packetize` and the cliparser jar produce. Set
it with `-virtual-mtu`, or at runtime with `PUT /api/transport {"att_mtu":247}`.

A real Mobi paired with the official iOS app negotiates a much larger MTU, so most responses reach the
phone as a **single** notification with `remaining=0` rather than a run of 18-byte fragments. Raising
the MTU here reproduces that: the pump re-frames each response (cliparser-encoded and natively encoded
alike) to the largest chunk the MTU allows, keeping the message bytes identical. Both receivers handle
either framing without a special case -- TandemKit's `BTResponseParser` and pumpX2's `PacketArrayList`
read the remaining-fragment counter (`data[0] & 0x0F`) and never a fragment's length, stripping 5 bytes
from the first fragment and 2 from each continuation. A driver's reassembler should be exercised
against both.

The peripheral will not send a notification larger than the announced MTU allows: an oversized notify
fails with an error rather than going out, so a fragmentation bug cannot pass here and then fail
against real hardware.

`att_mtu` is an additive, optional field; a v1 client that ignores it still works, since the peripheral
never sends more than it announces.

## Errors / lifecycle
- Unknown `type` -> {"type":"error","id":N?,"error":"unknown type"}; connection stays up.
- Socket EOF or error at any time == BLE disconnect for both sides.
- Only one attached central at a time (as BLE). A second TCP connection while one is attached gets
  hello + {"type":"disconnect","reason":"busy"}.
