# JPAKE pumpX2 Integration Test Issues

This document describes the issues encountered while implementing the `TestPumpX2JPAKEAuthenticator_FullFlow` integration test, which runs pumpX2's `jpake` client against `jpake-server` for end-to-end JPAKE authentication testing.

## Overview

The JPAKE (Password Authenticated Key Exchange) protocol requires cryptographic handshaking between a client (mobile app) and server (pump). The integration test spawns two pumpX2 processes and pipes BLE packets between them:

```
┌─────────────────┐         ┌─────────────────┐
│  jpake-server   │ ◄─────► │     jpake       │
│  (pump side)    │         │  (client side)  │
└─────────────────┘         └─────────────────┘
        │                           │
        │     BLE Packets           │
        └───────────────────────────┘
              via stdin/stdout
```

## Test Architecture

The test (`pkg/handler/jpake_integration_test.go`) does the following:

1. Spawns `jpake-server <pairingCode>` - acts as the pump
2. Spawns `jpake <pairingCode>` - acts as the mobile app client
3. Reads JSON output from each process containing `packets` arrays
4. Forwards packets from server to client and vice versa
5. Validates that both sides derive the same shared secret

### JPAKE Protocol Flow

| Round | Server Output | Client Output |
|-------|--------------|---------------|
| 1A | `JPAKE_1A: {packets: [...]}` | `ROUND_1A_SENT: {packets: [...]}` |
| 1B | `JPAKE_1B: {packets: [...]}` | `ROUND_1B_SENT: {packets: [...]}` |
| 2 | `JPAKE_2: {packets: [...]}` | `ROUND_2_SENT: {packets: [...]}` |
| 3 | `JPAKE_3: {packets: [...]}` | `CONFIRM_3_SENT: {packets: [...]}` |
| 4 | `JPAKE_4: {packets: [...]}` | Final confirmation |

Each message is split into multiple BLE packets (typically 10 packets for rounds 1-2, 2 packets for round 3).

## Problems Encountered

### Problem 1: Initial Timeout Waiting for JPAKE_1B

**Symptom:** Test timed out after 30 seconds waiting for `JPAKE_1B` response.

**Root Cause:** The `jpake-server` waits for client input after outputting `JPAKE_1A` before generating `JPAKE_1B`. The original test was waiting for both responses before sending client data.

**Fix:** Restructured the test to read/write interactively - forward server output to client immediately, then forward client output back to server.

### Problem 2: Empty Packets from EncodeMessage

**Symptom:** `bridge.EncodeMessage` returned empty packets for JPAKE messages, causing `ArrayIndexOutOfBoundsException` in pumpX2.

**Root Cause:** The encoding path for JPAKE messages in the Go bridge wasn't properly extracting packet data.

**Initial Workaround:** Used raw `centralChallengeHash` directly. However, this was rejected as a workaround - the test needed to use actual BLE packet format.

**Real Fix:** Rewrote the test to use actual pumpX2 client-server handshake instead of mocking.

### Problem 3: "PacketArrayList needs more packets" with Single Packet

**Symptom:** When forwarding only the first packet from a 10-packet message, pumpX2 reported needing more packets.

**Root Cause:** Each BLE message is split across multiple packets. The first packet's header byte indicates how many more packets follow (e.g., `09` means 9 more packets).

**Fix:** Extract and forward ALL packets from the `packets` array, not just the first one.

### Problem 4: Packet Boundary Detection Failure (Concatenated)

**Symptom:** When concatenating all packets into a single hex string, the parser found incorrect packet boundaries for JPAKE_3 (txId=3).

**Example:**
```
rawHex: 010327031200003a2354497658599b000000000000030000002fa8
i=0 num=1
spl:0003
messages:[01032703120]    <-- Only 11 chars, should be 40!
remainder: 0003a2354497658599b0000000000
```

**Root Cause:** The pumpX2 CLI's `BTResponseParser` uses pattern matching to find packet boundaries. It searches for the next packet's sequence pattern (e.g., `0003` for the last packet with txId=3).

The pattern `0003` appeared at position 10-13 inside the message data (`1200003a`), causing the parser to split at the wrong location.

**Why it worked for earlier messages:**
- JPAKE_1A/1B/2 use txId 0/1/2 with patterns like `0800`, `0000`, `0001`, `0002`
- These patterns are less likely to appear in random cryptographic data
- All packets were 40 hex chars (20 bytes), providing consistent spacing

**Why JPAKE_3 failed:**
- txId=3 means searching for pattern `0003`
- Only 2 packets (40 + 14 chars), not uniform size
- Pattern `0003` is common in message data (appears in length fields, etc.)

### Problem 5: "Empty data" with One Packet Per Line

**Symptom:** Sending each packet on its own line caused `java.lang.IllegalArgumentException: Empty data`.

**Example:**
```
testRequest uuid=AUTHORIZATION
PumpX2:DEBUG: BTResponseParser: PacketArrayList needs more packets: 09002000...
extraRawHex: java.lang.IllegalArgumentException: Empty data
```

**Root Cause:** The pumpX2 CLI reads ONE line and expects ALL packets on that line. After parsing the first packet and finding it needs more, it tries to read `extraRawHex` from the same line's remainder - but there is no remainder because each packet was on a separate line.

### Problem 6: Space-Separated Packets (Current Attempt)

**Current Approach:** Send all packets space-separated on one line:
```
packet1 packet2 packet3 ... packet10
```

**Hypothesis:** The parser might split on spaces first before applying the pattern-matching algorithm.

**Status:** Untested as of this writing.

## Root Cause Analysis

The fundamental issue is in pumpX2's `BTResponseParser` packet boundary detection algorithm:

```java
// Pseudocode of the problematic algorithm
void parseMultiPacket(String hexData) {
    int sequenceNum = parseFirstByte(hexData);  // e.g., 09 = 9 more packets
    int txId = parseSecondByte(hexData);        // e.g., 03 for JPAKE_3

    // Find next packet by searching for pattern
    for (int i = sequenceNum - 1; i >= 0; i--) {
        String pattern = String.format("%02x%02x", i, txId);  // e.g., "0003"
        int pos = hexData.indexOf(pattern);  // FRAGILE: pattern may appear in data!
        // Split at pos...
    }
}
```

This algorithm is **fragile** because:
1. It assumes sequence patterns won't appear in message data
2. It doesn't account for variable packet sizes
3. It finds the FIRST occurrence of the pattern, not necessarily the correct one

## Suggested pumpX2 Improvements

### Option 1: Fixed-Size Packet Splitting

Add a `--packet-size` option to use fixed-size splitting instead of pattern matching:

```java
// In CLIParser or BTResponseParser
if (packetSize > 0) {
    // Split by fixed size instead of pattern matching
    for (int i = 0; i < hexData.length(); i += packetSize * 2) {
        packets.add(hexData.substring(i, Math.min(i + packetSize * 2, hexData.length())));
    }
}
```

Usage:
```bash
./gradlew cliparser --args="jpake-server 123456 --packet-size 20"
```

### Option 2: Delimiter-Based Splitting

Accept a delimiter between packets (space, comma, semicolon):

```java
// In testRequest or parse function
if (hexData.contains(" ")) {
    String[] packets = hexData.split(" ");
    for (String packet : packets) {
        processPacket(packet);
    }
}
```

This allows:
```
packet1 packet2 packet3
# or
packet1,packet2,packet3
```

### Option 3: Explicit Packet Count Header

Add a header indicating the number of packets and their sizes:

```
10:40,40,40,40,40,40,40,40,40,24:hexdata...
```

Format: `count:size1,size2,...:hexdata`

### Option 4: JSON Input Mode

Accept structured JSON input instead of raw hex:

```json
{
  "packets": [
    "09002000a700004104...",
    "0800b473a487a4ab...",
    ...
  ]
}
```

This would be the most robust solution as it explicitly provides packet boundaries.

### Option 5: Fix Pattern Matching Algorithm

Instead of finding the FIRST occurrence of the pattern, find the occurrence at the expected position:

```java
// Current (buggy):
int pos = hexData.indexOf(pattern);

// Fixed:
int expectedPos = (sequenceNum - i) * PACKET_SIZE * 2;
if (hexData.substring(expectedPos, expectedPos + 4).equals(pattern)) {
    pos = expectedPos;
}
```

## Test File Reference

The integration test is located at:
- `pkg/handler/jpake_integration_test.go` - `TestPumpX2JPAKEAuthenticator_FullFlow`

Key functions:
- Server packet forwarding: Lines 155-179
- Client packet forwarding: Lines 192-215

## Commits Related to This Issue

| Commit | Description |
|--------|-------------|
| `3f905c9` | Rewrite JPAKE test to use actual client-server handshake |
| `2568bfd` | Fix: send all BLE packets to client, not just the first one |
| `d5f48c6` | Fix: concatenate all BLE packets into single line for client |
| `d361250` | Fix: forward client ROUND_XX_SENT packets to server |
| `effd632` | Fix: send BLE packets one per line instead of concatenating |
| `d23466e` | Fix: send packets space-separated on one line instead of newlines |

## Conclusion

The JPAKE integration test exposed a fundamental limitation in pumpX2's CLI packet parsing. The pattern-matching algorithm for finding packet boundaries is fragile and fails when:

1. Sequence patterns appear in message data
2. Transaction IDs are non-zero (especially txId >= 3)
3. Packets have variable sizes

The recommended fix is to modify pumpX2 to support explicit packet boundaries through one of the options above, with **Option 2 (delimiter-based splitting)** or **Option 4 (JSON input mode)** being the most practical solutions.
