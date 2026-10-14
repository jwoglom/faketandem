package messages

func init() {
	register("request.currentStatus.ApiVersionRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			if len(raw) == 0 {
				return
			}
			checkf(len(raw) == m.size(), "got length %d", len(raw))
			m.Cargo = dropFirstN(raw, 3)
		},
		ctors: map[string]func(*Message, args){
			"":       func(m *Message, _ args) { m.Cargo = []byte{} },
			"byte[]": func(m *Message, a args) { m.Cargo = a.Bytes(0) },
		},
	})

	register("response.currentStatus.ApiVersionResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.setInt("majorVersion", readShort(raw, 0))
			m.setInt("minorVersion", readShort(raw, 2))
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"int,int": func(m *Message, a args) {
				major, minor := a.Int(0), a.Int(1)
				cargo := make([]byte, 4)
				copy(cargo, combine(firstTwoBytesLittleEndian(major), firstTwoBytesLittleEndian(minor)))
				m.Cargo = cargo
				m.setInt("majorVersion", major)
				m.setInt("minorVersion", minor)
			},
		},
	})

	register("request.currentStatus.CurrentBasalStatusRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
		},
		ctors: map[string]func(*Message, args){
			"": func(m *Message, _ args) { m.Cargo = []byte{} },
		},
	})

	register("response.currentStatus.CurrentBasalStatusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("profileBasalRate", readUint32(raw, 0))
			m.setLong("currentBasalRate", readUint32(raw, 4))
			m.setInt("basalModifiedBitmask", sb(raw[8]))
		},
		ctors: map[string]func(*Message, args){
			"long,long,int": func(m *Message, a args) {
				m.Cargo = combine(toUint32(a.Long(0)), toUint32(a.Long(1)), []byte{byte(a.Int(2))})
				m.setLong("profileBasalRate", a.Long(0))
				m.setLong("currentBasalRate", a.Long(1))
				m.setInt("basalModifiedBitmask", a.Int(2))
			},
		},
	})

	register("response.control.InitiateBolusResponse", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setInt("status", sb(raw[0]))
			m.setInt("bolusId", readShort(raw, 1))
			m.setInt("statusTypeId", sb(raw[5]))
		},
		ctors: map[string]func(*Message, args){
			"int,int,int": func(m *Message, a args) {
				m.Cargo = combine([]byte{byte(a.Int(0))}, firstTwoBytesLittleEndian(a.Int(1)), []byte{0, 0}, []byte{byte(a.Int(2))})
				m.setInt("status", a.Int(0))
				m.setInt("bolusId", a.Int(1))
				m.setInt("statusTypeId", a.Int(2))
			},
		},
	})

	register("request.control.InitiateBolusRequest", classImpl{
		parse: func(m *Message, raw []byte) {
			raw = m.removeSignedRequestHmacBytes(raw)
			check(len(raw) == m.size())
			m.Cargo = raw
			m.setLong("totalVolume", readUint32(raw, 0))
			m.setInt("bolusID", readShort(raw, 4))
			m.setInt("bolusTypeBitmask", sb(raw[8]))
			m.setLong("foodVolume", readUint32(raw, 9))
			m.setLong("correctionVolume", readUint32(raw, 13))
			m.setInt("bolusCarbs", readShort(raw, 17))
			m.setInt("bolusBG", readShort(raw, 19))
			m.setLong("bolusIOB", readUint32(raw, 21))
			m.setLong("extendedVolume", readUint32(raw, 25))
			m.setLong("extendedSeconds", readUint32(raw, 29))
			m.setLong("extended3", readUint32(raw, 33))
		},
		ctors: map[string]func(*Message, args){
			"byte[]": func(m *Message, a args) { m.parseSelf(a.Bytes(0)) },
			"long,int,int,long,long,int,int,long": func(m *Message, a args) {
				initiateBolusRequest(m, a.Long(0), a.Int(1), a.Int(2), a.Long(3), a.Long(4), a.Int(5), a.Int(6), a.Long(7), 0, 0, 0)
			},
			"long,int,int,long,long,int,int,long,long,long,long": func(m *Message, a args) {
				initiateBolusRequest(m, a.Long(0), a.Int(1), a.Int(2), a.Long(3), a.Long(4), a.Int(5), a.Int(6), a.Long(7), a.Long(8), a.Long(9), a.Long(10))
			},
		},
	})
}

func initiateBolusRequest(m *Message, totalVolume int64, bolusID, bolusTypeBitmask int32, foodVolume, correctionVolume int64,
	bolusCarbs, bolusBG int32, bolusIOB, extendedVolume, extendedSeconds, extended3 int64) {
	const minBolusMilliunits, minExtendedBolusMilliunits = 50, 400
	check(totalVolume >= minBolusMilliunits || totalVolume+extendedVolume >= minExtendedBolusMilliunits)
	check(bolusID > 0)
	check(foodVolume >= 0)
	check(correctionVolume >= 0)
	check(bolusCarbs >= 0)
	check(bolusBG >= 0)
	check(bolusIOB >= 0)
	m.Cargo = combine(
		toUint32(totalVolume),
		firstTwoBytesLittleEndian(bolusID),
		[]byte{0, 0},
		[]byte{byte(bolusTypeBitmask)},
		toUint32(foodVolume),
		toUint32(correctionVolume),
		firstTwoBytesLittleEndian(bolusCarbs),
		firstTwoBytesLittleEndian(bolusBG),
		toUint32(bolusIOB),
		toUint32(extendedVolume),
		toUint32(extendedSeconds),
		toUint32(extended3),
	)
	m.setLong("totalVolume", totalVolume)
	m.setInt("bolusID", bolusID)
	m.setInt("bolusTypeBitmask", bolusTypeBitmask)
	m.setLong("foodVolume", foodVolume)
	m.setLong("correctionVolume", correctionVolume)
	m.setInt("bolusCarbs", bolusCarbs)
	m.setInt("bolusBG", bolusBG)
	m.setLong("bolusIOB", bolusIOB)
	m.setLong("extendedVolume", extendedVolume)
	m.setLong("extendedSeconds", extendedSeconds)
	m.setLong("extended3", extended3)
}
