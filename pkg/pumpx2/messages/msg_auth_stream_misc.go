package messages

import "bytes"

func init() {
	register("request.authentication.CentralChallengeRequest", classImpl{
		parse: func(m *Message, raw []byte) { asmChallengeRequestParse(m, raw, "centralChallenge") },
		ctors: map[string]func(*Message, args){
			"int,byte[]": func(m *Message, a args) {
				m.Cargo = asmArraycopyCargo(10, firstTwoBytesLittleEndian(a.Int(0)), a.Bytes(1))
				m.setInt("appInstanceId", a.Int(0))
				m.setBytes("centralChallenge", a.Bytes(1))
			},
		},
	})

	for _, name := range []string{
		"request.authentication.Jpake1aRequest",
		"request.authentication.Jpake1bRequest",
		"request.authentication.Jpake2Request",
	} {
		register(name, classImpl{
			parse: func(m *Message, raw []byte) { asmChallengeRequestParse(m, raw, "centralChallenge") },
			ctors: map[string]func(*Message, args){
				"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
				"int,byte[]": func(m *Message, a args) {
					cargo := make([]byte, 167)
					copy(cargo, combine(firstTwoBytesLittleEndian(a.Int(0)), a.Bytes(1)))
					m.Cargo = cargo
					m.setInt("appInstanceId", a.Int(0))
					m.setBytes("centralChallenge", a.Bytes(1))
				},
			},
		})
	}

	register("request.authentication.Jpake3SessionKeyRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			m.Cargo = raw
			m.setInt("challengeParam", readShort(copyOfRange(raw, 0, 2), 0))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int": func(m *Message, a args) {
				m.Cargo = asmArraycopyCargo(2, firstTwoBytesLittleEndian(a.Int(0)))
				m.setInt("challengeParam", a.Int(0))
			},
		},
	})

	for _, name := range []string{
		"request.authentication.Jpake4KeyConfirmationRequest",
		"response.authentication.Jpake4KeyConfirmationResponse",
	} {
		register(name, classImpl{
			parse: asmJpake4Parse,
			ctors: map[string]func(*Message, args){
				"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
				"int,byte[],byte[],byte[]": func(m *Message, a args) {
					appInstanceID, nonce, reserved, hashDigest := a.Int(0), a.Bytes(1), a.Bytes(2), a.Bytes(3)
					checkf(len(nonce) == 8, "nonce was %d not 8", len(nonce))
					check(len(reserved) == 8)
					checkf(len(hashDigest) == 32, "hashDigest was %d not 32", len(hashDigest))
					m.Cargo = asmArraycopyCargo(50, firstTwoBytesLittleEndian(appInstanceID), nonce, reserved, hashDigest)
					m.setInt("appInstanceId", appInstanceID)
					m.setBytes("nonce", nonce)
					m.setBytes("reserved", reserved)
					m.setBytes("hashDigest", hashDigest)
				},
			},
		})
	}

	register("request.authentication.PumpChallengeRequest", classImpl{
		parse: func(m *Message, raw []byte) { asmChallengeRequestParse(m, raw, "pumpChallengeHash") },
		ctors: map[string]func(*Message, args){
			"int,byte[]": func(m *Message, a args) {
				appInstanceID, hash := a.Int(0), a.Bytes(1)
				m.parseSelf(asmArraycopyCargo(22, firstTwoBytesLittleEndian(appInstanceID), hash))
				check(readShort(copyOfRange(m.Cargo, 0, 2), 0) == appInstanceID)
				check(bytes.Equal(copyOfRange(m.Cargo, 2, len(m.Cargo)), hash))
			},
		},
	})

	register("response.authentication.CentralChallengeResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("appInstanceId", readShort(raw, 0))
			m.setBytes("centralChallengeHash", copyOfRange(raw, 2, 22))
			m.setBytes("hmacKey", copyOfRange(raw, 22, 30))
		},
		ctors: map[string]func(*Message, args){
			"int,byte[],byte[]": func(m *Message, a args) {
				appInstanceID, hash, hmacKey := a.Int(0), a.Bytes(1), a.Bytes(2)
				m.parseSelf(combine(firstTwoBytesLittleEndian(appInstanceID), combine(hash, hmacKey)))
				check(readShort(m.Cargo, 0) == appInstanceID)
				check(bytes.Equal(copyOfRange(m.Cargo, 2, 22), hash))
				check(bytes.Equal(copyOfRange(m.Cargo, 22, 30), hmacKey))
			},
		},
	})

	for _, name := range []string{
		"response.authentication.Jpake1aResponse",
		"response.authentication.Jpake1bResponse",
		"response.authentication.Jpake2Response",
	} {
		register(name, classImpl{
			parse: asmJpakeResponseParse,
			ctors: map[string]func(*Message, args){
				"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
				"int,byte[]": func(m *Message, a args) {
					appInstanceID, hash := a.Int(0), a.Bytes(1)
					m.parseSelf(combine(firstTwoBytesLittleEndian(appInstanceID), hash))
					check(readShort(m.Cargo, 0) == appInstanceID)
					parsedHash := copyOfRange(m.Cargo, 2, m.size())
					check(bytes.Equal(parsedHash, asmCopyOf(hash, len(parsedHash))))
				},
			},
		})
	}

	register("response.authentication.Jpake3SessionKeyResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("appInstanceId", readShort(copyOfRange(raw, 0, 2), 0))
			m.setBytes("deviceKeyNonce", copyOfRange(raw, 2, 10))
			m.setBytes("deviceKeyReserved", copyOfRange(raw, 10, 18))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"int,byte[],byte[]": func(m *Message, a args) {
				appInstanceID, nonce, reserved := a.Int(0), a.Bytes(1), a.Bytes(2)
				m.parseSelf(combine(firstTwoBytesLittleEndian(appInstanceID), nonce, reserved))
				check(readShort(copyOfRange(m.Cargo, 0, 2), 0) == appInstanceID)
				check(bytes.Equal(copyOfRange(m.Cargo, 2, 10), nonce))
				check(bytes.Equal(copyOfRange(m.Cargo, 10, 18), reserved))
			},
		},
	})

	register("response.authentication.PumpChallengeResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("appInstanceId", readShort(raw, 0))
			m.setBool("success", raw[2] == 1)
		},
		ctors: map[string]func(*Message, args){
			"int,boolean": func(m *Message, a args) {
				success := byte(0)
				if a.Bool(1) {
					success = 1
				}
				m.parseSelf(combine(firstTwoBytesLittleEndian(a.Int(0)), []byte{success}))
			},
		},
	})

	for _, name := range []string{
		"request.controlStream.NonexistentDetectingCartridgeStateStreamRequest",
		"request.controlStream.NonexistentEnterChangeCartridgeModeStateStreamRequest",
		"request.controlStream.NonexistentExitFillTubingModeStateStreamRequest",
		"request.controlStream.NonexistentFillCannulaStateStreamRequest",
		"request.controlStream.NonexistentFillTubingStateStreamRequest",
		"request.controlStream.NonexistentLoadCartridgeStateStreamRequest",
		"request.controlStream.NonexistentPrimeNudgeStateStreamRequest",
		"request.controlStream.NonexistentPumpingStateStreamRequest",
	} {
		register(name, classImpl{
			parse: func(m *Message, raw []byte) {},
			ctors: map[string]func(*Message, args){
				"": func(m *Message, _ args) { m.Cargo = []byte{} },
			},
		})
	}

	register("request.historyLog.NonexistentHistoryLogStreamRequest", classImpl{
		parse: func(m *Message, raw []byte) { m.Cargo = raw },
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
		},
	})

	register("response.ErrorResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			m.setInt("requestCodeId", sb(raw[0]))
			errorCodeID := sb(raw[1])
			m.setInt("errorCodeId", errorCodeID)
			asmSetEnum(m, "errorCode", asmErrorCodes, errorCodeID)
			if len(raw) > 2 {
				m.setBytes("remainingBytes", dropFirstN(raw, 2))
			}
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				m.parseSelf([]byte{byte(a.Int(0)), byte(a.Int(1))})
			},
		},
	})

	register("response.controlStream.DetectingCartridgeStateStreamResponse", classImpl{
		parse: asmDetectingCartridgeParse,
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
			"int": func(m *Message, a args) {
				m.Cargo = combine(firstTwoBytesLittleEndian(a.Int(0)))
				m.setInt("percentComplete", a.Int(0))
			},
		},
	})

	register("response.controlStream.LoadCartridgeStateStreamResponse", classImpl{
		parse: asmDetectingCartridgeParse,
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				raw := combine([]byte{byte(a.Int(0))})
				m.Cargo = raw
				m.parseSelf(raw)
			},
		},
	})

	register("response.controlStream.EnterChangeCartridgeModeStateStreamResponse", classImpl{
		parse: func(m *Message, raw []byte) { asmStateIDParse(m, raw, asmChangeCartridgeStates) },
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))})
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("response.controlStream.FillCannulaStateStreamResponse", classImpl{
		parse: func(m *Message, raw []byte) { asmStateIDParse(m, raw, asmFillCannulaStates) },
		ctors: map[string]func(*Message, args){
			"int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))})
				m.parseSelf(m.Cargo)
			},
		},
	})

	register("response.controlStream.FillTubingStateStreamResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("buttonState", sb(raw[0]))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) {
				m.Cargo = a.Bytes(0)
				m.parseSelf(a.Bytes(0))
			},
		},
	})

	exitFillTubingStateID := func(m *Message, a args) {
		m.Cargo = combine([]byte{byte(a.Int(0))})
		m.parseSelf(m.Cargo)
	}
	exitFillTubingPumping := func(m *Message, a args) {
		isSet := byte(0)
		if a.Bool(0) {
			isSet = 1
		}
		m.Cargo = combine([]byte{isSet}, toUint32(a.Long(1)))
		m.parseSelf(m.Cargo)
	}
	register("response.controlStream.ExitFillTubingModeStateStreamResponse", classImpl{
		parse: asmExitFillTubingParse,
		ctors: map[string]func(*Message, args){
			"int":          exitFillTubingStateID,
			"boolean,long": exitFillTubingPumping,
		},
	})
	register("response.controlStream.PrimeNudgeStateStreamResponse", classImpl{
		parse: asmExitFillTubingParse,
		ctors: map[string]func(*Message, args){
			"int": exitFillTubingStateID,
		},
	})
	register("response.controlStream.PumpingStateStreamResponse", classImpl{
		parse: asmExitFillTubingParse,
		ctors: map[string]func(*Message, args){
			"boolean,long": exitFillTubingPumping,
		},
	})

	// Its parse would print pumpX2's history log records, which are not ported.
	register("response.historyLog.HistoryLogStreamResponse", classImpl{})
}

var asmErrorCodes = map[int32]string{
	0: "UNDEFINED_ERROR",
	1: "CRC_MISMATCH",
	3: "TRANSACTION_ID_MISMATCH",
	4: "BAD_CARGO_LENGTH",
	6: "BAD_OPCODE",
	7: "INVALID_REQUIRED_PARAMETER",
	8: "MESSAGE_BUFFER_FULL",
	9: "INVALID_AUTHENTICATION_ERROR",
}

var (
	asmChangeCartridgeStates   = map[int32]string{2: "READY_TO_CHANGE"}
	asmFillCannulaStates       = map[int32]string{2: "CANNULA_FILLED"}
	asmExitFillTubingModeState = map[int32]string{0: "NOT_COMPLETE", 1: "TUBING_FILLED"}
)

// asmCopyOf is Arrays.copyOf: truncated or zero-padded to n.
func asmCopyOf(b []byte, n int) []byte {
	out := make([]byte, n)
	copy(out, b)
	return out
}

// asmArraycopyCargo is `new byte[n]` filled by System.arraycopy(Bytes.combine(parts), 0, cargo, 0, n).
func asmArraycopyCargo(n int, parts ...[]byte) []byte {
	combined := combine(parts...)
	if len(combined) < n {
		throw("java.lang.ArrayIndexOutOfBoundsException", "arraycopy: last source index %d out of bounds for byte[%d]", n, len(combined))
	}
	cargo := make([]byte, n)
	copy(cargo, combined)
	return cargo
}

func asmChallengeRequestParse(m *Message, raw []byte, field string) {
	m.Cargo = raw
	m.setInt("appInstanceId", readShort(copyOfRange(raw, 0, 2), 0))
	m.setBytes(field, copyOfRange(raw, 2, len(raw)))
}

func asmJpakeResponseParse(m *Message, raw []byte) {
	checkf(len(raw) <= m.size(), "size is %d not %d", len(raw), m.size())
	m.Cargo = asmCopyOf(raw, m.size())
	m.setInt("appInstanceId", readShort(m.Cargo, 0))
	m.setBytes("centralChallengeHash", copyOfRange(m.Cargo, 2, m.size()))
}

func asmJpake4Parse(m *Message, raw []byte) {
	m.Cargo = raw
	m.setInt("appInstanceId", readShort(copyOfRange(raw, 0, 2), 0))
	m.setBytes("nonce", copyOfRange(raw, 2, 10))
	m.setBytes("reserved", copyOfRange(raw, 10, 18))
	m.setBytes("hashDigest", copyOfRange(raw, 18, 50))
}

func asmDetectingCartridgeParse(m *Message, raw []byte) {
	raw = m.removeSignedRequestHmacBytes(raw)
	check(len(raw) == 1 || len(raw) == 2)
	m.Cargo = raw
	if len(raw) == 2 {
		percentComplete := readShort(raw, 0)
		m.setInt("percentComplete", percentComplete)
		m.setInt("stateId", percentComplete)
		m.setBool("loadCartridgeState", false)
	} else {
		stateID := sb(raw[0])
		m.setInt("stateId", stateID)
		m.setInt("percentComplete", stateID)
		m.setBool("loadCartridgeState", true)
	}
}

func asmStateIDParse(m *Message, raw []byte, states map[int32]string) {
	raw = m.removeSignedRequestHmacBytes(raw)
	check(len(raw) == m.size())
	m.Cargo = raw
	stateID := sb(raw[0])
	m.setInt("stateId", stateID)
	asmSetEnum(m, "state", states, stateID)
}

func asmExitFillTubingParse(m *Message, raw []byte) {
	raw = m.removeSignedRequestHmacBytes(raw)
	check(len(raw) == 1 || len(raw) == 5)
	m.Cargo = raw
	stateID := ub(raw[0])
	m.setInt("stateId", stateID)
	asmSetEnum(m, "state", asmExitFillTubingModeState, stateID)
	if len(raw) == 5 {
		m.setBool("isPumpingStateSetAfterStartUp", raw[0] != 0)
		m.setLong("stateBitmask", readUint32(raw, 1))
	} else {
		m.setBool("isPumpingStateSetAfterStartUp", false)
		m.setLong("stateBitmask", 0)
	}
}

func asmSetEnum(m *Message, field string, names map[int32]string, id int32) {
	if name, ok := names[id]; ok {
		m.setObject(field, name)
	} else {
		m.setNull(field)
	}
}
