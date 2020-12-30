## bench 

### 1 encodeFrameTo

#### BEFORE

```go
func encodeFrameTo(frm *Frame) []byte {
	var (
		part1 uint16 // from FIN to PayloadLen
	)

	part1 |= frm.Fin << finOffset
	part1 |= frm.RSV1 << rsv1Offset
	part1 |= frm.RSV2 << rsv2Offset
	part1 |= frm.RSV3 << rsv3Offset
	part1 |= uint16(frm.OpCode) << opcodeOffset
	part1 |= frm.Mask << maskOffset
	part1 |= frm.PayloadLen << payloadLenOffset

	// start from 0th byte
	// fill part1 into 2 byte
	buf := make([]byte, 2, minFrameHeaderSize+8)
	binary.BigEndian.PutUint16(buf[:2], part1)

	// FIXED: fill payloadExtendLen into 8 byte
	switch frm.PayloadLen {
	case 126:
		payloadExtendBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(payloadExtendBuf[:2], uint16(frm.PayloadExtendLen))
		buf = append(buf, payloadExtendBuf...)
	case 127:
		payloadExtendBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(payloadExtendBuf[:8], frm.PayloadExtendLen)
		buf = append(buf, payloadExtendBuf...)
	}

	// FIXED: if not mask, then no set masking key
	if frm.Mask == 1 {
		// fill fmtMaskingKey into 4 byte
		maskingKeyBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(maskingKeyBuf[:4], frm.MaskingKey)
		buf = append(buf, maskingKeyBuf...)
	}

	// header done, start writing body
	buf = append(buf, frm.Payload...)

	return buf
}
```

#### AFTER

```go
func calcBufLen(frm *Frame) (bufLen int) {
	bufLen = 2

	// FIXED: fill payloadExtendLen into 8 byte
	switch frm.PayloadLen {
	case 126:
		bufLen += 2
	case 127:
		bufLen += 8
	}

	// FIXED: if not mask, then no set masking key
	if frm.Mask == 1 {
		bufLen += 4
	}

	return
}

// encodeFrameToV2 .
func encodeFrameToV2(frm *Frame) []byte {
	var (
		part1 uint16 // from FIN to PayloadLen
	)

	part1 |= frm.Fin << finOffset
	part1 |= frm.RSV1 << rsv1Offset
	part1 |= frm.RSV2 << rsv2Offset
	part1 |= frm.RSV3 << rsv3Offset
	part1 |= uint16(frm.OpCode) << opcodeOffset
	part1 |= frm.Mask << maskOffset
	part1 |= frm.PayloadLen << payloadLenOffset

	// write into buf
	var (
		ptr = 0
		buf = make([]byte, calcBufLen(frm))
	)
	// header
	binary.BigEndian.PutUint16(buf[ptr:2], part1)
	ptr += 2
	// payload ext len
	switch frm.PayloadLen {
	case 126:
		binary.BigEndian.PutUint16(buf[ptr:ptr+2], uint16(frm.PayloadExtendLen))
		ptr += 2
	case 127:
		binary.BigEndian.PutUint64(buf[ptr:ptr+8], frm.PayloadExtendLen)
		ptr += 8
	}

	if frm.Mask == 1 {
		binary.BigEndian.PutUint32(buf[ptr:ptr+4], frm.MaskingKey)
		ptr += 4
	}

	// write payload
	copy(buf[ptr:], frm.Payload)
	return buf
}
```

#### BENCH CMP

```sh
$ benchstat encodeFrameTo.old encodeFrameTo.new
name              old time/op  new time/op  delta
_encodeFrameTo-4  29.7ns ±11%  26.9ns ±75%  -9.36%  (p=0.008 n=24+29)
```