# JPAKE Server-Side Implementation Setup

This document describes the changes required to pumpX2 for proper server-side JPAKE authentication.

## Overview

The faketandem pump simulator now uses pumpX2's actual cryptographic implementation for J-PAKE authentication instead of stub code. This requires adding a new `jpake-server` command to pumpX2's cliparser.

## Required pumpX2 Changes

### File: `cliparser/src/main/java/com/jwoglom/pumpx2/cliparser/Main.java`

#### 1. Add jpake-server case to main switch statement

**Location:** Around line 167-172

**Change:**
```java
// OLD CODE:
case "jpake":
    System.out.println(jpakeAuthEncoder(args[1]));
default:
    System.err.println("Nothing to do.");
    break;

// NEW CODE:
case "jpake":
    System.out.println(jpakeAuthEncoder(args[1]));
    break;
case "jpake-server":
    System.out.println(jpakeAuthServer(args[1]));
    break;
default:
    System.err.println("Nothing to do.");
    break;
```

**Note:** The original code was missing a `break` after the `jpake` case, causing it to fall through to `default`.

#### 2. Add jpakeAuthServer method

**Location:** After the jpakeAuthEncoder method (around line 243)

**Add these two methods:**

```java
private static String jpakeAuthServer(String pairingCode) {
    try {
        // Initialize server-side EcJpake with the pairing code
        io.particle.crypto.EcJpake serverJpake = new io.particle.crypto.EcJpake(
            io.particle.crypto.EcJpake.Role.SERVER,
            JpakeAuthBuilder.pairingCodeToBytes(pairingCode)
        );

        Scanner scanner = new Scanner(System.in);
        byte txId = 0;

        // Generate server's round 1 (330 bytes total)
        byte[] serverRound1 = serverJpake.getRound1();

        // Step 1a: Send first 165 bytes
        byte[] serverRound1a = Arrays.copyOfRange(serverRound1, 0, 165);
        JSONObject resp1a = new JSONObject();
        resp1a.put("messageName", "Jpake1aResponse");
        resp1a.put("centralChallengeHash", Hex.encodeHexString(serverRound1a));
        System.out.println("JPAKE_1A: " + encode(txId, "Jpake1aResponse", resp1a.toString()));
        System.out.flush();
        txId++;

        // Read client's round 1a
        if (!scanner.hasNextLine()) {
            return errorJson("No client response for round 1a");
        }
        String clientHex1a = scanner.nextLine().trim();
        Message clientMsg1a = parse(clientHex1a);
        if (!(clientMsg1a instanceof com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake1aRequest)) {
            return errorJson("Expected Jpake1aRequest, got: " + (clientMsg1a != null ? clientMsg1a.getClass().getName() : "null"));
        }
        byte[] clientRound1a = ((com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake1aRequest) clientMsg1a).getCentralChallengeHash();

        // Step 1b: Send next 165 bytes
        byte[] serverRound1b = Arrays.copyOfRange(serverRound1, 165, 330);
        JSONObject resp1b = new JSONObject();
        resp1b.put("messageName", "Jpake1bResponse");
        resp1b.put("centralChallengeHash", Hex.encodeHexString(serverRound1b));
        System.out.println("JPAKE_1B: " + encode(txId, "Jpake1bResponse", resp1b.toString()));
        System.out.flush();
        txId++;

        // Read client's round 1b
        if (!scanner.hasNextLine()) {
            return errorJson("No client response for round 1b");
        }
        String clientHex1b = scanner.nextLine().trim();
        Message clientMsg1b = parse(clientHex1b);
        if (!(clientMsg1b instanceof com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake1bRequest)) {
            return errorJson("Expected Jpake1bRequest, got: " + (clientMsg1b != null ? clientMsg1b.getClass().getName() : "null"));
        }
        byte[] clientRound1b = ((com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake1bRequest) clientMsg1b).getCentralChallengeHash();

        // Combine client's round 1 data and feed to EcJpake
        byte[] clientRound1Full = Bytes.combine(clientRound1a, clientRound1b);
        serverJpake.readRound1(clientRound1Full);

        // Generate and send server's round 2
        byte[] serverRound2 = serverJpake.getRound2();
        JSONObject resp2 = new JSONObject();
        resp2.put("messageName", "Jpake2Response");
        resp2.put("centralChallengeHash", Hex.encodeHexString(serverRound2));
        System.out.println("JPAKE_2: " + encode(txId, "Jpake2Response", resp2.toString()));
        System.out.flush();
        txId++;

        // Read client's round 2
        if (!scanner.hasNextLine()) {
            return errorJson("No client response for round 2");
        }
        String clientHex2 = scanner.nextLine().trim();
        Message clientMsg2 = parse(clientHex2);
        if (!(clientMsg2 instanceof com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake2Request)) {
            return errorJson("Expected Jpake2Request, got: " + (clientMsg2 != null ? clientMsg2.getClass().getName() : "null"));
        }
        byte[] clientRound2 = ((com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake2Request) clientMsg2).getCentralChallengeHash();
        serverJpake.readRound2(clientRound2);

        // Derive the shared secret
        byte[] derivedSecret = serverJpake.deriveSecret();

        // Round 3: Session key exchange
        // Read client's session key request
        if (!scanner.hasNextLine()) {
            return errorJson("No client response for round 3");
        }
        String clientHex3 = scanner.nextLine().trim();
        Message clientMsg3 = parse(clientHex3);
        if (!(clientMsg3 instanceof com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake3SessionKeyRequest)) {
            return errorJson("Expected Jpake3SessionKeyRequest, got: " + (clientMsg3 != null ? clientMsg3.getClass().getName() : "null"));
        }

        // Generate server nonce for round 3
        byte[] serverNonce3 = new byte[8];
        new java.security.SecureRandom().nextBytes(serverNonce3);
        byte[] reserved3 = new byte[4]; // All zeros

        JSONObject resp3 = new JSONObject();
        resp3.put("messageName", "Jpake3SessionKeyResponse");
        resp3.put("deviceKeyNonce", Hex.encodeHexString(serverNonce3));
        resp3.put("deviceKeyReserved", Hex.encodeHexString(reserved3));
        System.out.println("JPAKE_3: " + encode(txId, "Jpake3SessionKeyResponse", resp3.toString()));
        System.out.flush();
        txId++;

        // Round 4: Key confirmation
        // Read client's key confirmation
        if (!scanner.hasNextLine()) {
            return errorJson("No client response for round 4");
        }
        String clientHex4 = scanner.nextLine().trim();
        Message clientMsg4 = parse(clientHex4);
        if (!(clientMsg4 instanceof com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake4KeyConfirmationRequest)) {
            return errorJson("Expected Jpake4KeyConfirmationRequest, got: " + (clientMsg4 != null ? clientMsg4.getClass().getName() : "null"));
        }

        com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake4KeyConfirmationRequest req4 =
            (com.jwoglom.pumpx2.pump.messages.request.authentication.Jpake4KeyConfirmationRequest) clientMsg4;
        byte[] clientNonce4 = req4.getNonce();
        byte[] clientHashDigest = req4.getHashDigest();

        // Validate client's HMAC
        byte[] expectedClientHash = com.jwoglom.pumpx2.pump.messages.builders.crypto.HmacSha256.hmacSha256(
            clientNonce4,
            com.jwoglom.pumpx2.pump.messages.builders.crypto.Hkdf.build(serverNonce3, derivedSecret)
        );

        if (!Arrays.equals(expectedClientHash, clientHashDigest)) {
            return errorJson("Client HMAC validation failed");
        }

        // Generate server's response for round 4
        byte[] serverNonce4 = new byte[8];
        new java.security.SecureRandom().nextBytes(serverNonce4);
        byte[] serverHashDigest = com.jwoglom.pumpx2.pump.messages.builders.crypto.HmacSha256.hmacSha256(
            serverNonce4,
            com.jwoglom.pumpx2.pump.messages.builders.crypto.Hkdf.build(serverNonce3, derivedSecret)
        );

        JSONObject resp4 = new JSONObject();
        resp4.put("messageName", "Jpake4KeyConfirmationResponse");
        resp4.put("nonce", Hex.encodeHexString(serverNonce4));
        resp4.put("hashDigest", Hex.encodeHexString(serverHashDigest));
        System.out.println("JPAKE_4: " + encode(txId, "Jpake4KeyConfirmationResponse", resp4.toString()));
        System.out.flush();
        txId++;

        // Success! Return the derived secret
        JSONObject result = new JSONObject();
        result.put("derivedSecret", Hex.encodeHexString(derivedSecret));
        result.put("serverNonce", Hex.encodeHexString(serverNonce3));
        result.put("messageName", "JpakeAuthResult");
        result.put("txId", "" + txId);
        return result.toString();

    } catch (Exception e) {
        JSONObject result = new JSONObject();
        result.put("error", "Exception during server JPAKE authentication: " + e.getMessage());
        e.printStackTrace();
        return result.toString();
    }
}

private static String errorJson(String message) {
    JSONObject result = new JSONObject();
    result.put("error", message);
    return result.toString();
}
```

## Testing the Changes

### 1. Apply the changes to pumpX2

```bash
cd /path/to/pumpX2
# Apply the changes shown above to cliparser/src/main/java/com/jwoglom/pumpx2/cliparser/Main.java
```

### 2. Build pumpX2

```bash
cd /path/to/pumpX2
./gradlew cliparser
```

### 3. Test jpake-server command

```bash
# Using gradle:
./gradlew cliparser -q --console=plain --args="jpake-server 123456"

# Using JAR:
java -jar cliparser/build/libs/cliparser.jar jpake-server 123456
```

You should see:
```
JPAKE_1A: {encoded JSON response}
JPAKE_1B: {encoded JSON response}
```

The command will then wait for client requests on stdin.

### 4. Run faketandem with pumpx2 mode

```bash
./faketandem -pumpx2-path /path/to/pumpX2 -jpake-mode pumpx2
```

## How It Works

1. **Server Initialization:**
   - Creates `EcJpake` instance with `Role.SERVER` and pairing code
   - Generates server's round1 challenge (330 bytes)
   - Splits into two parts for Jpake1aResponse (165 bytes) and Jpake1bResponse (165 bytes)

2. **Client-Server Exchange:**
   - Server outputs responses as: `JPAKE_XX: {json}`
   - Reads client requests from stdin (hex format)
   - Validates each client message type
   - Uses EcJpake to validate cryptographic proofs

3. **Shared Secret Derivation:**
   - After round 2, derives shared secret using EcJpake
   - Generates random nonce for session key exchange
   - Validates client's HMAC using Hkdf and HmacSha256
   - Returns final derived secret on successful authentication

4. **Go Integration:**
   - `PumpX2JPAKEAuthenticator` spawns jpake-server process
   - Uses goexpect for interactive stdin/stdout communication
   - Parses JPAKE_XX JSON responses
   - Caches responses and returns to client via BLE

## Security Notes

- Uses BouncyCastle's EC-JPAKE implementation (same as actual Tandem pumps)
- Proper zero-knowledge proofs for each round
- HMAC validation using HKDF key derivation
- Cryptographically secure random nonce generation
- Validates all client messages and proof values

## Troubleshooting

**Error: "Plugin [id: 'com.android.application'] was not found"**
- Network connectivity issue with Gradle
- Try building with local Gradle: `gradle cliparser`

**Error: "Expected JpakeXXRequest, got: null"**
- Client sent invalid hex data
- Check that client is sending proper Tandem pump messages

**Error: "Client HMAC validation failed"**
- Pairing code mismatch between client and server
- Client using wrong derived secret
- Check that both sides are using the same 6-digit pairing code

**Error: "Validation failed" from EcJpake**
- Cryptographic proof validation failed
- Client sent invalid zero-knowledge proofs
- Possible man-in-the-middle attack or corrupted data
