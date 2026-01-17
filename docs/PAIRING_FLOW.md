# Complete Pod Pairing Flow Documentation

This document describes the FULL pairing flow for a new Tandem pump pod, synthesizing information from the codebase implementation, reverse engineering analysis, and protocol documentation.

## Table of Contents

1. [Overview](#overview)
2. [Phase 1: BLE Discovery and Advertising](#phase-1-ble-discovery-and-advertising)
3. [Phase 2: BLE Connection Establishment](#phase-2-ble-connection-establishment)
4. [Phase 3: Service and Characteristic Discovery](#phase-3-service-and-characteristic-discovery)
5. [Phase 4: Authentication Flow](#phase-4-authentication-flow)
6. [Phase 5: Session Key Derivation](#phase-5-session-key-derivation)
7. [Phase 6: Authenticated Communication](#phase-6-authenticated-communication)
8. [Security Analysis](#security-analysis)
9. [Implementation Reference](#implementation-reference)

---

## Overview

The Tandem pump pairing flow involves multiple layers of security:

1. **BLE-level discovery control** - The pump controls when it's discoverable
2. **Application-level pairing** - 6-digit pairing code entered on pump
3. **J-PAKE authentication** - Password-authenticated key exchange
4. **Session key derivation** - HKDF-based key derivation
5. **Message signing** - HMAC-SHA1 for authenticated commands

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          CLIENT (Mobile App)                             │
├─────────────────────────────────────────────────────────────────────────┤
│  1. Scan for BLE advertisements with Tandem service UUID                 │
│  2. Connect to pump                                                      │
│  3. Discover services and characteristics                                │
│  4. Enable notifications on characteristics                              │
│  5. Initiate J-PAKE authentication                                       │
│  6. Derive session key                                                   │
│  7. Send authenticated commands                                          │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ BLE
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          PUMP (Server/Peripheral)                        │
├─────────────────────────────────────────────────────────────────────────┤
│  1. Advertise with manufacturer data indicating pairing state            │
│  2. Accept BLE connections when discoverable                             │
│  3. Expose GATT services and characteristics                             │
│  4. Participate in J-PAKE as server                                      │
│  5. Derive shared session key                                            │
│  6. Validate signed commands                                             │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: BLE Discovery and Advertising

### Pairing States

The pump operates in one of four pairing states, controlled by the user through the pump's physical interface:

| State | Manufacturer Data | Description |
|-------|------------------|-------------|
| `NotDiscoverable` | `0x10` | Pump rejects all connections |
| `DiscoverableOnly` | `0x10` | Pump accepts connections, no pairing |
| `PairStep1` | `0x11` | Pump ready for initial pairing |
| `PairStep2` | `0x12` | Pump continuing pairing process |

**Source:** `pkg/bluetooth/pairing_state.go:7-15`

### Advertisement Packet Structure

The pump advertises with the following data:

```
Advertisement Packet:
├── Flags: 0x06 (LE General Discoverable + BR/EDR Not Supported)
│   └── Or 0x04 if not discoverable
├── Incomplete List of 16-bit UUIDs: 0xFDFB (Tandem Service)
├── TX Power Level: 0x04
└── Manufacturer Specific Data:
    ├── Company ID: 0x059D (Tandem Diabetes Care)
    └── Data: [0x00, 0x01, <pairing_state_byte>]
```

**Source:** `pkg/bluetooth/bluetooth_linux.go:216-270`

### Service UUID

```
Primary Service UUID: 0000fdfb-0000-1000-8000-00805f9b34fb
```

### Device Discovery Steps

1. **Client scans** for BLE devices advertising the Tandem service UUID
2. **Client checks** manufacturer data to determine pairing state
3. **If `PairStep1` or `PairStep2`**, client can proceed with pairing
4. **If `DiscoverableOnly`**, client can connect but not pair new device
5. **If `NotDiscoverable`**, pump will reject connection attempts

---

## Phase 2: BLE Connection Establishment

### Connection Flow

```
Client                                           Pump
   │                                               │
   │──────── BLE Connection Request ──────────────>│
   │                                               │
   │<─────── BLE Connection Accept ────────────────│
   │         (if discoverable)                     │
   │                                               │
   │<─────── OR Connection Reject ─────────────────│
   │         (if not discoverable)                 │
```

### Connection Parameters

The pump advertises preferred connection parameters:

```go
// From Generic Access Service (0x1800)
PeripheralPreferredConnectionParameters: [0x18, 0x00, 0x28, 0x00, 0x00, 0x00, 0xf4, 0x01]
// Connection Interval: 24-40 (30-50ms)
// Slave Latency: 0
// Supervision Timeout: 500 (5s)
```

**Source:** `pkg/bluetooth/bluetooth_linux.go:188`

---

## Phase 3: Service and Characteristic Discovery

### GATT Services

The pump exposes these services:

| Service | UUID | Description |
|---------|------|-------------|
| Generic Attribute | `0x1801` | Standard GATT service |
| Generic Access | `0x1800` | Device name, appearance |
| Device Information | `0x180A` | Manufacturer, model, serial |
| Unknown FDFA | `0000fdfa-0000-1000-8000-00805f9b34fb` | Unknown purpose (Mobi) |
| **Tandem Pump** | `0000fdfb-0000-1000-8000-00805f9b34fb` | **Main pump service** |

### Tandem Pump Service Characteristics

| Characteristic | UUID | Properties | Purpose |
|----------------|------|------------|---------|
| CurrentStatus | `7B83FFF6-9F77-4E5C-8064-AAE2C24838B9` | Write, Notify | Status requests/responses |
| QualifyingEvents | `7B83FFF7-9F77-4E5C-8064-AAE2C24838B9` | Write, Notify | Event notifications |
| HistoryLog | `7B83FFF8-9F77-4E5C-8064-AAE2C24838B9` | Notify | History log streaming |
| **Authorization** | `7B83FFF9-9F77-4E5C-8064-AAE2C24838B9` | Write, Notify | **Authentication messages** |
| Control | `7B83FFFC-9F77-4E5C-8064-AAE2C24838B9` | Write, Notify | Control commands |
| ControlStream | `7B83FFFD-9F77-4E5C-8064-AAE2C24838B9` | Write, Notify | Streaming control |

**Source:** `pkg/bluetooth/bluetooth.go:22-28`

### Characteristic Discovery Steps

1. **Discover primary services** by UUID
2. **Discover characteristics** for Tandem Pump service
3. **Enable notifications** (CCCD) on all characteristics:
   - CurrentStatus
   - QualifyingEvents
   - HistoryLog
   - Authorization
   - Control
   - ControlStream

---

## Phase 4: Authentication Flow

### Authentication Overview

The pump supports two authentication methods:

1. **J-PAKE (Modern)** - Password-Authenticated Key Exchange using Elliptic Curves
2. **Legacy Challenge-Response** - HMAC-based challenge-response

### J-PAKE Authentication Protocol

J-PAKE (Password Authenticated Key Exchange by Juggling) provides:
- Zero-knowledge proof of password knowledge
- Resistance to offline dictionary attacks
- Forward secrecy

#### Pre-requisites

1. User enters 6-digit pairing code on pump display
2. User enters same code in mobile app
3. Both sides derive shared password from pairing code

#### J-PAKE Message Flow

```
Client (Mobile App)                              Pump (Server)
       │                                              │
       │═══════════════════════════════════════════════│
       │          ROUND 1A: Exchange First Half        │
       │═══════════════════════════════════════════════│
       │                                              │
       │──── CentralChallengeRequest ─────────────────>│
       │     (appInstanceId)                          │
       │                                              │
       │<─── CentralChallengeResponse ────────────────│
       │     (initiates JPAKE)                        │
       │                                              │
       │──── Jpake1aRequest ──────────────────────────>│
       │     (centralChallengeHash: 165 bytes)        │
       │     Client's gX1, gX2 (first half)           │
       │                                              │
       │<─── Jpake1aResponse ─────────────────────────│
       │     (centralChallengeHash: 165 bytes)        │
       │     Server's gX1, gX2 (first half)           │
       │                                              │
       │═══════════════════════════════════════════════│
       │          ROUND 1B: Exchange Second Half       │
       │═══════════════════════════════════════════════│
       │                                              │
       │──── Jpake1bRequest ──────────────────────────>│
       │     (centralChallengeHash: 165 bytes)        │
       │     Client's gX1, gX2 (second half)          │
       │                                              │
       │<─── Jpake1bResponse ─────────────────────────│
       │     (centralChallengeHash: 165 bytes)        │
       │     Server's gX1, gX2 (second half)          │
       │                                              │
       │═══════════════════════════════════════════════│
       │          ROUND 2: Key Material Exchange       │
       │═══════════════════════════════════════════════│
       │                                              │
       │──── Jpake2Request ───────────────────────────>│
       │     (centralChallengeHash)                   │
       │     Client's A = g^((x1+x2+x3)*s)            │
       │                                              │
       │<─── Jpake2Response ──────────────────────────│
       │     (centralChallengeHash)                   │
       │     Server's B = g^((x1+x2+x3)*s)            │
       │                                              │
       │═══════════════════════════════════════════════│
       │          ROUND 3: Session Key Request         │
       │═══════════════════════════════════════════════│
       │                                              │
       │──── Jpake3SessionKeyRequest ─────────────────>│
       │     (Request session key nonce)              │
       │                                              │
       │<─── Jpake3SessionKeyResponse ────────────────│
       │     (deviceKeyNonce: 8 bytes)                │
       │     (deviceKeyReserved: 4 bytes)             │
       │                                              │
       │═══════════════════════════════════════════════│
       │          ROUND 4: Key Confirmation            │
       │═══════════════════════════════════════════════│
       │                                              │
       │──── Jpake4KeyConfirmationRequest ────────────>│
       │     (nonce: 8 bytes)                         │
       │     (hashDigest: HMAC of nonce)              │
       │                                              │
       │<─── Jpake4KeyConfirmationResponse ───────────│
       │     (nonce: 8 bytes)                         │
       │     (hashDigest: HMAC of nonce)              │
       │                                              │
       │═══════════════════════════════════════════════│
       │          AUTHENTICATION COMPLETE              │
       │═══════════════════════════════════════════════│
```

**Source:** `JPAKE_SETUP.md`, `pkg/handler/jpake_pumpx2.go`

### Message Structure

#### Packet Format

Messages are split into packets for BLE transmission:

```
Packet Header (2 bytes):
├── Byte 0: Remaining packet count (0 = last packet)
└── Byte 1: Transaction ID (0-255)

Payload:
├── Bytes 2-N: Message data
└── Last 2 bytes (final packet): CRC16

Packet Sizes:
├── Authorization characteristic: 40 bytes max
└── Other characteristics: 18 bytes max
```

**Source:** `SIMULATOR_IMPROVEMENT_PLAN.md:160-171`

#### J-PAKE Round 1 Data Structure

```
Round 1 Total Size: 330 bytes
├── First half (1a): 165 bytes
│   ├── gX1: Public value 1
│   └── ZKP for gX1: Zero-knowledge proof
└── Second half (1b): 165 bytes
    ├── gX2: Public value 2
    └── ZKP for gX2: Zero-knowledge proof
```

### Cryptographic Details

#### Pairing Code to Password

```go
// Convert 6-digit pairing code to JPAKE password bytes
password := JpakeAuthBuilder.pairingCodeToBytes(pairingCode)
```

The pairing code is converted to a byte array used as the shared password in J-PAKE.

#### EcJpake Implementation

The J-PAKE implementation uses:
- **Elliptic Curve**: NIST P-256 (secp256r1)
- **Library**: io.particle.crypto.EcJpake (BouncyCastle-based)
- **Role**: Server (pump) or Client (mobile app)

```java
// Server-side initialization
EcJpake serverJpake = new EcJpake(
    EcJpake.Role.SERVER,
    JpakeAuthBuilder.pairingCodeToBytes(pairingCode)
);
```

**Source:** `JPAKE_SETUP.md:47-51`

---

## Phase 5: Session Key Derivation

### Derived Secret

After J-PAKE round 2, both parties derive the same shared secret:

```java
byte[] derivedSecret = serverJpake.deriveSecret();
```

### HKDF Key Derivation

The session key is derived using HKDF (HMAC-based Key Derivation Function):

```java
// Key derivation using server nonce and derived secret
byte[] sessionKey = Hkdf.build(serverNonce, derivedSecret);
```

**Source:** `JPAKE_SETUP.md:171-174`

### Key Confirmation HMAC

Both parties confirm they derived the same key using HMAC-SHA256:

```java
// Client computes HMAC of their nonce
byte[] clientHash = HmacSha256.hmacSha256(clientNonce, sessionKey);

// Server verifies client's HMAC
byte[] expectedClientHash = HmacSha256.hmacSha256(clientNonce, sessionKey);
if (!Arrays.equals(expectedClientHash, clientHashDigest)) {
    throw new Exception("Client HMAC validation failed");
}
```

**Source:** `JPAKE_SETUP.md:169-177`

---

## Phase 6: Authenticated Communication

### Post-Authentication State

After successful J-PAKE:

1. **Pump marks connection as authenticated**
2. **Session key stored** for message signing
3. **Authenticated commands now accepted**

```go
// State change after authentication
pumpState.SetAuthenticated(authKey)
bridge.SetAuthenticationKey(hex.EncodeToString(authKey))
```

**Source:** `pkg/handler/router.go:221-225`

### Message Signing (HMAC-SHA1)

Authenticated messages require HMAC-SHA1 signatures:

```
Signed Message Structure:
├── Message Payload
├── Time Since Reset (4 bytes): Pump uptime in seconds
└── HMAC-SHA1 Signature (20 bytes): Computed over payload + timestamp
```

### Authentication-Required Messages

The following message types require authentication:

| Message Type | Description |
|--------------|-------------|
| `CurrentStatusRequest` | Get pump status |
| `BolusPermissionRequest` | Request bolus permission |
| `InitiateBolusRequest` | Start insulin delivery |
| `BolusTerminationRequest` | Cancel bolus |
| `ProfileBasalRequest` | Get basal profile |
| `HistoryLogRequest` | Get history entries |
| Various settings requests | Pump configuration |

Messages that do NOT require authentication:

| Message Type | Description |
|--------------|-------------|
| `ApiVersionRequest` | Get API version |
| `CentralChallengeRequest` | Initiate auth |
| `Jpake*Request` | J-PAKE rounds |
| `PumpChallengeRequest` | Legacy auth |
| `TimeSinceResetRequest` | Get pump uptime |

**Source:** `pkg/handler/router.go:62-101`

---

## Security Analysis

### Security Properties

1. **Zero-Knowledge Proof**: Neither party reveals their password
2. **Offline Dictionary Attack Resistance**: J-PAKE prevents offline password cracking
3. **Forward Secrecy**: Compromise of long-term keys doesn't compromise past sessions
4. **Mutual Authentication**: Both parties prove knowledge of the pairing code
5. **Message Integrity**: HMAC-SHA1 prevents tampering
6. **Replay Protection**: Time-since-reset prevents replay attacks

### Potential Attack Vectors

| Attack | Mitigation |
|--------|------------|
| BLE Eavesdropping | J-PAKE encrypts key exchange |
| Brute Force Pairing | 6-digit code = 1,000,000 combinations |
| Replay Attacks | Time-since-reset in signed messages |
| MITM | J-PAKE requires password knowledge |
| Offline Dictionary | J-PAKE design prevents this |

### Manufacturer Data Analysis

The manufacturer data byte reveals pairing state:
- `0x10`: Normal operation (not pairing)
- `0x11`: Pairing step 1 (awaiting connection)
- `0x12`: Pairing step 2 (pairing in progress)

This allows clients to detect when pump is ready for pairing.

---

## Implementation Reference

### Complete Pairing Sequence (Step by Step)

#### Step 1: Prepare Pump for Pairing

```
User Action: Navigate to pump settings → Bluetooth → Pair New Device
Pump Action:
  1. Display 6-digit pairing code on screen
  2. Set pairing state to PairStep1 (0x11)
  3. Begin advertising with updated manufacturer data
```

#### Step 2: Client Discovery

```
Client Action:
  1. Scan for BLE devices with UUID 0xFDFB
  2. Check manufacturer data for 0x11 or 0x12
  3. Display pump to user for selection
```

#### Step 3: BLE Connection

```
Client Action: Initiate BLE connection to pump
Pump Action: Accept connection (if discoverable)
Client Action:
  1. Discover services
  2. Discover characteristics
  3. Enable notifications on all pump characteristics
```

#### Step 4: API Version Check

```
Client → Pump: ApiVersionRequest
Pump → Client: ApiVersionResponse { apiVersion: 5 }
```

#### Step 5: Initiate Authentication

```
Client → Pump: CentralChallengeRequest { appInstanceId: <random> }
Pump → Client: CentralChallengeResponse { appInstanceId: <echo> }
```

#### Step 6: J-PAKE Round 1a

```
Client Action: Generate client round 1 (330 bytes total)
Client → Pump: Jpake1aRequest { centralChallengeHash: <first 165 bytes> }
Pump Action: Generate server round 1 (330 bytes total)
Pump → Client: Jpake1aResponse { centralChallengeHash: <first 165 bytes> }
```

#### Step 7: J-PAKE Round 1b

```
Client → Pump: Jpake1bRequest { centralChallengeHash: <second 165 bytes> }
Pump Action: Combine client round 1, process with EcJpake
Pump → Client: Jpake1bResponse { centralChallengeHash: <second 165 bytes> }
```

#### Step 8: J-PAKE Round 2

```
Client Action: Combine server round 1, compute round 2
Client → Pump: Jpake2Request { centralChallengeHash: <round 2 data> }
Pump Action: Process client round 2, compute server round 2
Pump → Client: Jpake2Response { centralChallengeHash: <round 2 data> }
Both: Derive shared secret from JPAKE
```

#### Step 9: Session Key Exchange (Round 3)

```
Client → Pump: Jpake3SessionKeyRequest { }
Pump Action: Generate 8-byte server nonce
Pump → Client: Jpake3SessionKeyResponse {
  deviceKeyNonce: <8 bytes>,
  deviceKeyReserved: <4 bytes zeros>
}
Both: Derive session key = HKDF(serverNonce, derivedSecret)
```

#### Step 10: Key Confirmation (Round 4)

```
Client Action:
  1. Generate 8-byte client nonce
  2. Compute HMAC = HMAC-SHA256(clientNonce, sessionKey)
Client → Pump: Jpake4KeyConfirmationRequest {
  nonce: <8 bytes>,
  hashDigest: <HMAC>
}
Pump Action:
  1. Verify client HMAC
  2. Generate 8-byte server nonce
  3. Compute HMAC = HMAC-SHA256(serverNonce, sessionKey)
Pump → Client: Jpake4KeyConfirmationResponse {
  nonce: <8 bytes>,
  hashDigest: <HMAC>
}
Client Action: Verify server HMAC
```

#### Step 11: Authentication Complete

```
Both: Mark connection as authenticated
Pump: May update pairing state to PairStep2 or NotDiscoverable
Client: Can now send authenticated commands
```

### Code Locations

| Component | File | Description |
|-----------|------|-------------|
| Pairing States | `pkg/bluetooth/pairing_state.go` | State definitions |
| BLE Advertising | `pkg/bluetooth/bluetooth_linux.go:216-270` | Advertisement packets |
| J-PAKE Handler | `pkg/handler/jpake.go` | J-PAKE message routing |
| J-PAKE Authenticator | `pkg/handler/jpake_auth.go` | Go implementation |
| PumpX2 J-PAKE | `pkg/handler/jpake_pumpx2.go` | pumpX2 integration |
| Auth Challenge | `pkg/handler/auth_challenge.go` | Initial auth handler |
| Message Router | `pkg/handler/router.go` | Message dispatch |
| Protocol Layer | `pkg/protocol/` | Packet reassembly |
| pumpX2 Bridge | `pkg/pumpx2/bridge.go` | Java interop |

---

## Appendix: Troubleshooting

### Common Pairing Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Connection rejected | Pump not discoverable | Check pairing state, start pairing mode |
| J-PAKE fails round 2 | Pairing code mismatch | Verify codes match on both sides |
| HMAC validation failed | Wrong session key | Re-initiate J-PAKE |
| Timeout waiting for response | BLE connection lost | Reconnect and restart |
| "Expected JpakeXRequest, got null" | Invalid packet data | Check packet encoding |

### Debug Logging

Enable verbose logging to debug pairing:

```bash
./faketandem -v -pumpx2-path /path/to/pumpX2 -jpake-mode pumpx2
```

This shows:
- All BLE packet data (RX/TX)
- J-PAKE round transitions
- Shared secret derivation
- Authentication state changes

---

## References

- `JPAKE_SETUP.md` - Server-side J-PAKE implementation
- `SIMULATOR_IMPROVEMENT_PLAN.md` - Architecture overview
- `docs/JPAKE_PUMPX2_INTEGRATION_TEST_ISSUES.md` - Packet format details
- pumpX2 repository: https://github.com/jwoglom/pumpX2
- J-PAKE RFC: https://tools.ietf.org/html/rfc8236
