## bench 

### 1 encodeFrameTo

#### DIFF

```diff
+func calcBufLen(frm *Frame) (bufLen int) {
+    bufLen = 2
+    
+    // FIXED: fill payloadExtendLen into 8 byte
+    switch frm.PayloadLen {
+        case 126:
+            bufLen += 2
+        case 127:
+            bufLen += 8
+    }
+    
+    // FIXED: if not mask, then no set masking key
+    if frm.Mask == 1 {
+        bufLen += 4
+    }
+    
+    return
+}

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

+   var (
+     ptr = 0
+     buf = make([]byte, calcBufLen(frm))
+   )
-   // start from 0th byte
-   // fill part1 into 2 byte
-   buf := make([]byte, 2, minFrameHeaderSize+8)
-   binary.BigEndian.PutUint16(buf[:2], part1)
+   // header
+   binary.BigEndian.PutUint16(buf[ptr:2], part1)
+   ptr += 2
-   // FIXED: fill payloadExtendLen into 8 byte
-   switch frm.PayloadLen {
-   case 126:
-       payloadExtendBuf := make([]byte, 2)
-       binary.BigEndian.PutUint16(payloadExtendBuf[:2], uint16(frm.PayloadExtendLen))
-       buf = append(buf, payloadExtendBuf...)
+       binary.BigEndian.PutUint16(buf[ptr:ptr+2], uint16(frm.PayloadExtendLen))
+       ptr += 2
-   case 127:
-       payloadExtendBuf := make([]byte, 8)
-       binary.BigEndian.PutUint64(payloadExtendBuf[:8], frm.PayloadExtendLen)
-       buf = append(buf, payloadExtendBuf...)
+       binary.BigEndian.PutUint64(buf[ptr:ptr+8], frm.PayloadExtendLen)
+       ptr += 8
-   }

-   // FIXED: if not mask, then no set masking key
-   if frm.Mask == 1 {
-       // fill fmtMaskingKey into 4 byte
-       maskingKeyBuf := make([]byte, 4)
-       binary.BigEndian.PutUint32(maskingKeyBuf[:4], frm.MaskingKey)
-       buf = append(buf, maskingKeyBuf...)
+       binary.BigEndian.PutUint32(buf[ptr:ptr+4], frm.MaskingKey)
+       ptr += 4
-   }

-   // header done, start writing body
-   buf = append(buf, frm.Payload...)
+   copy(buf[ptr:], frm.Payload)

    return buf
}
```
#### BENCH CMP

```sh
$ benchstat encodeFrameTo.old encodeFrameTo.new
name              old time/op  new time/op  delta
_encodeFrameTo-4  29.7ns ±11%  26.9ns ±75%  -9.36%  (p=0.008 n=24+29)
```

### 2 setPayload

#### DIFF

```diff
// setPayload . automatic mask or unmask payload data
func (frm *Frame) setPayload(payload []byte) *Frame {
-    frm.Payload = make([]byte, len(payload))
-    copy(frm.Payload, payload)
-    if len(payload) > 256 {
-        logger.Debugf("Frame.setPayload got frm.Payload over 256, so ignore to display")
-    } else {
-        logger.Debugf("Frame.setPayload got frm.Payload=%v", frm.Payload)
-    }
    
+   frm.Payload = payload
+   if _debug {
+       if len(payload) > 256 {
+           logger.Debugf("Frame.setPayload got frm.Payload over 256, so ignore to display")
+       } else {
+           logger.Debugf("Frame.setPayload got frm.Payload=%v", frm.Payload)
+       }
+   }
    
    if frm.Mask == 1 {
        // true: if mask has been set, then calc masking-key with payload
        frm.maskPayload()
    }

    frm.autoCalcPayloadLen()
    return frm
}
```

#### BENCH CMP

```shell
$ benchstat bench/setPayload.old bench/setPayload.new
name                           old time/op  new time/op  delta
_Frame_SetPayload_less126-4    3.42µs ±39%  0.22µs ±21%  -93.49%  (p=0.000 n=27+29)
_Frame_SetPayload_65535-4      92.2µs ±23%  82.3µs ±24%  -10.72%  (p=0.000 n=26+28)
_Frame_SetPayload_more65535-4   172µs ± 1%   159µs ±14%   -7.91%  (p=0.000 n=29+28)
```
