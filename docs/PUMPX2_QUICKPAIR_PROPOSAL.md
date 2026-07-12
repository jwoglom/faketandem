# Proposal: pumpX2 support for JPAKE "quick pair" reconnects

## Background

Real Tandem mobile apps don't always run the full 5-message JPAKE handshake
(`Jpake1aRequest` -> `Jpake1bRequest` -> `Jpake2Request` ->
`Jpake3SessionKeyRequest` -> `Jpake4KeyConfirmationRequest`). Once an app has
completed that exchange once with a given pump, a later BLE reconnect can
"quick pair": it sends `Jpake3SessionKeyRequest` as the very first message,
skipping the password-authenticated rounds 1a/1b/2 entirely, and reuses the
long-term secret derived during the original pairing to establish a fresh
per-connection session key (a new nonce run through
`Hkdf.build`/`HmacSha256.hmacSha256`).

faketandem observed this directly: a real app's first message on a fresh BLE
connection was `Jpake3SessionKeyRequest`, which pumpX2's `jpake-server` CLI
command rejected with `Expected Jpake1aRequest, got:
Jpake3SessionKeyRequest`, because that command always spawns a fresh
`EcJpake` state machine and expects the full round sequence from the start.

## What faketandem did on its side

`pkg/handler/jpake_quickreconnect.go` now hand-ports the exact crypto pumpX2
uses for rounds 3/4 (`Hkdf.build` and `HmacSha256.hmacSha256`, verified
against pumpX2's own `messages/builders/crypto/Hkdf.java` and
`HmacSha256.java`) and answers a quick-pair reconnect in Go, entirely without
spawning pumpX2's subprocess, using a long-term secret cached from an earlier
completed pairing (or seeded via `-jpake-long-term-key` / the web UI).

This works, but it means faketandem's "pumpx2" JPAKE mode is no longer a pure
pass-through to pumpX2 for every round -- rounds 1a/1b/2/(3/4 on a full
pairing) still go through the real `jpake-server` subprocess, but a
quick-pair's rounds 3/4 are answered by a second, independent Go
implementation of the same math. That's a parity risk: if pumpX2's HKDF/HMAC
construction ever changes, faketandem's copy could silently drift out of
sync with no test coverage catching it (short of the crypto conformance test
in `pkg/handler/jpake_quickreconnect_test.go`, which only checks the Go code
against itself).

## Proposed pumpX2 changes

### 1. A `jpake-server-resume` CLI mode

Add a sibling command to the existing `jpake-server` in
`cliparser/src/main/java/com/jwoglom/pumpx2/cliparser/Main.java` (see
`JPAKE_SETUP.md` for the existing `jpake-server` implementation this would
extend):

```
jpake-server-resume <pairingCode> <longTermSecretHex>
```

Instead of constructing a fresh `EcJpake` and running `getRound1()` /
`readRound1()` / `getRound2()` / `readRound2()` / `deriveSecret()`, this
mode would:

1. Decode `longTermSecretHex` directly as the `derivedSecret` (skip
   `EcJpake` entirely -- there's no round 1/2 negotiation to do).
2. Read the client's `Jpake3SessionKeyRequest` from stdin exactly like
   `jpake-server` currently does after round 2, and answer it with
   `JPAKE_3: {...}` using a fresh random nonce.
3. Read the client's `Jpake4KeyConfirmationRequest`, validate its HMAC using
   `Hkdf.build(serverNonce3, derivedSecret)` +
   `HmacSha256.hmacSha256(...)` (identical to `jpake-server`'s existing round
   3/4 code), and answer with `JPAKE_4: {...}`.

This is a small, additive change -- most of the round 3/4 logic in
`jpakeAuthServer` (see `JPAKE_SETUP.md` lines covering "Round 3: Session key
exchange" onward) can be factored into a shared helper both commands call,
parameterized on `derivedSecret` instead of computing it via `EcJpake`.

### 2. Why route through pumpX2 instead of keeping the Go implementation

Once `jpake-server-resume` exists, `PumpX2JPAKEAuthenticator` (or a thin
wrapper) could drive quick-pair reconnects through the real pumpX2 crypto
the same way it already does for full pairings, and
`QuickReconnectJPAKEAuthenticator` (the faketandem-side Go implementation)
could become a fallback for `-jpake-mode go`/environments without a pumpX2
checkout, rather than the only implementation. That removes the parity risk
above -- one code path (pumpX2 itself) is the source of truth for the JPAKE
wire format and math in both the full-pairing and quick-pair cases.

### 3. Optional: expose Hkdf/HmacSha256 as standalone CLI utilities

A smaller, independently useful change: add `hkdf <nonceHex>
<keyMaterialHex>` and `hmac-sha256 <keyHex> <dataHex>` cases to `Main.java`
that just print the hex-encoded result of
`Hkdf.build`/`HmacSha256.hmacSha256`. This would let
`pkg/handler/jpake_quickreconnect_test.go` (or a new integration test) shell
out to a real cliparser jar and assert faketandem's Go port produces
byte-identical output to pumpX2's actual implementation, closing the parity
gap even before `jpake-server-resume` lands.

## Non-goals

This proposal doesn't change anything about full pairing (rounds
1a/1b/2/3/4 from scratch) -- that already round-trips through the real
`jpake-server` process today and isn't affected.
