// Copyright (c) 2018 YeQiang
// MIT License
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

package websocket

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockFrame(payload []byte) *Frame {
	frm := Frame{
		Fin:              1,   // uint8  1 bit
		RSV1:             0,   // uint8  1 bit
		RSV2:             0,   // uint8  1 bit
		RSV3:             0,   // uint8  1 bit
		OpCode:           1,   // OpCode 4 bits
		Mask:             1,   // uint8  1 bit
		PayloadLen:       0,   // uint8  7 bits
		PayloadExtendLen: 0,   // uint64 64 bits
		MaskingKey:       0,   // uint64 32 bits
		Payload:          nil, // []byte no limit by RFC6455
	}

	if frm.Mask == 1 {
		(&frm).genMaskingKey()
	}

	if len(payload) == 0 {
		frm.setPayload([]byte("hello"))
	} else {
		frm.setPayload(payload)
	}

	return &frm
}

// mockFragmentFrames .
func mockFragmentFrames(noMask bool) []*Frame {
	var mask uint16 = 1

	if noMask {
		mask = 0
	}

	frame1 := &Frame{
		Fin: 0,
		// RSV1:             0,
		// RSV2:             0,
		// RSV3:             0,
		OpCode: opCodeText,
		Mask:   mask,
		// PayloadLen:       0,
		// PayloadExtendLen: 0,
		// MaskingKey:       0,
		Payload: nil,
	}

	frame2 := &Frame{
		Fin: 0,
		// RSV1:             0,
		// RSV2:             0,
		// RSV3:             0,
		OpCode: opCodeContinuation,
		Mask:   mask,
		// PayloadLen:       0,
		// PayloadExtendLen: 0,
		// MaskingKey:       0,
		Payload: nil,
	}

	frame3 := &Frame{
		Fin: 1,
		// RSV1:   0,
		// RSV2:   0,
		// RSV3:   0,
		OpCode: opCodeContinuation,
		Mask:   mask,
		// PayloadLen:       0,
		// PayloadExtendLen: 0,
		// MaskingKey:       0,
		Payload: nil,
	}

	frms := []*Frame{frame1, frame2, frame3}

	for idx, v := range frms {
		if v.Mask == 1 {
			frms[idx].genMaskingKey()
		}

		frms[idx].setPayload(
			append([]byte("frame"), byte(idx+1)),
		)
	}

	return frms
}

// decodeToFrame .
// !!!!!! should noly be test called !!!!!!
func decodeToFrame(buf []byte) (*Frame, error) {
	if len(buf) < minFrameHeaderSize {
		return nil, ErrInvalidData
	}

	// 2 means: Header (2 Byte)
	frm := parseFrameHeader(buf[:2])

	var (
		payloadExtendLen uint64     // this could be non exist, payloadExtendLen = 0
		cur              uint64 = 2 // after header
	)

	switch frm.PayloadLen {
	case 126:
		// has 16bit + 32bit = 6B
		payloadExtendLen = uint64(binary.BigEndian.Uint16(buf[cur : cur+2]))
		cur += 2
	case 127:
		// has 64bit + 32bit = 12B
		payloadExtendLen = uint64(binary.BigEndian.Uint64(buf[cur : cur+8]))
		cur += 8
	}
	frm.PayloadExtendLen = payloadExtendLen

	// get masking key
	if frm.Mask == 1 {
		frm.MaskingKey = binary.BigEndian.Uint32(buf[cur : cur+4])
		cur += 4
	}

	var payloadlength uint64 = uint64(frm.PayloadLen)
	if frm.PayloadExtendLen != 0 {
		payloadlength = frm.PayloadExtendLen
	}
	frm.Payload = buf[cur : cur+payloadlength]

	return frm, nil
}

func Test_EncodeFrame_Decode(t *testing.T) {
	src := mockFrame(nil)
	buf := encodeFrameTo(src)
	debugPrintEncodedFrame(buf)

	dst, err := decodeToFrame(buf)
	if err != nil {
		t.Log(err)
	}
	assert.Equal(t, dst, src)

	t.Logf("payload=%s", dst.Payload)
}

func Test_constructFrame(t *testing.T) {
	type args struct {
		opcode OpCode
		finnal bool
		noMask bool
	}
	tests := []struct {
		name string
		args args
		want *Frame
	}{
		{
			name: "case 0",
			args: args{
				opcode: opCodeText,
				finnal: true,
				noMask: true,
			},
			want: &Frame{
				Fin:              1,
				RSV1:             0,
				RSV2:             0,
				RSV3:             0,
				OpCode:           opCodeText,
				Mask:             0,
				PayloadLen:       0,
				PayloadExtendLen: 0,
				MaskingKey:       0,
				Payload:          nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructFrame(tt.args.opcode, tt.args.finnal, tt.args.noMask)
			if tt.want.Mask == 1 {
				tt.want.MaskingKey = got.MaskingKey
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_constructDataFrame(t *testing.T) {
	type args struct {
		data   []byte
		noMask bool
		opcode OpCode
	}
	tests := []struct {
		name string
		args args
		want *Frame
	}{
		{
			name: "case 0",
			args: args{
				data:   []byte("hello"),
				noMask: false,
				opcode: opCodeText,
			},
			want: &Frame{
				Fin:              1,
				RSV1:             0,
				RSV2:             0,
				RSV3:             0,
				OpCode:           opCodeText,
				Mask:             1,
				PayloadLen:       uint16(len([]byte("hello"))),
				PayloadExtendLen: 0,
				MaskingKey:       0,
				Payload:          nil,
			},
		},
		{
			name: "case 1",
			args: args{
				data:   []byte("hello"),
				noMask: false,
				opcode: opCodeBinary,
			},
			want: &Frame{
				Fin:              1,
				RSV1:             0,
				RSV2:             0,
				RSV3:             0,
				OpCode:           opCodeBinary,
				Mask:             1,
				PayloadLen:       uint16(len([]byte("hello"))),
				PayloadExtendLen: 0,
				MaskingKey:       0,
				Payload:          nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructDataFrame(tt.args.data, tt.args.noMask, tt.args.opcode)
			if tt.want.Mask == 1 {
				tt.want.MaskingKey = got.MaskingKey
			}
			tt.want.setPayload([]byte("hello"))

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_constructControlFrame(t *testing.T) {
	type args struct {
		opcode  OpCode
		noMask  bool
		payload []byte
	}
	tests := []struct {
		name string
		args args
		want *Frame
	}{
		{
			name: "case 0",
			args: args{
				opcode:  opCodePing,
				noMask:  true,
				payload: []byte("payload"),
			},
			want: &Frame{
				Fin:              1,
				RSV1:             0,
				RSV2:             0,
				RSV3:             0,
				OpCode:           opCodePing,
				Mask:             0,
				PayloadLen:       7,
				PayloadExtendLen: 0,
				MaskingKey:       0,
				Payload:          []byte("payload"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructControlFrame(tt.args.opcode, tt.args.noMask, tt.args.payload)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_FrameMaskAndUnmask(t *testing.T) {
	frm := mockFrame(nil)
	want := frm.Payload
	t.Logf("before %+v", frm)
	frm.maskPayload()
	t.Logf("masked %+v", frm)
	frm.maskPayload()
	t.Logf("unmasked %+v", frm)
	got := frm.Payload

	assert.Equal(t, got, want)
}

func Test_Mask(t *testing.T) {
	var maskingKey uint32 = 0x9acb0442
	masks := genMasks(maskingKey)
	expected := [4]byte{
		0x9a,
		0xcb,
		0x04,
		0x42,
	}

	assert.Equal(t, expected, masks)
}

func Test_Frame_valid(t *testing.T) {
	type fields struct {
		Fin              uint16
		RSV1             uint16
		RSV2             uint16
		RSV3             uint16
		OpCode           OpCode
		Mask             uint16
		PayloadLen       uint16
		PayloadExtendLen uint64
		MaskingKey       uint32
		Payload          []byte
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "case 0",
			fields: fields{
				Fin: 1,
				// RSV1:             0,
				// RSV2:             0,
				// RSV3:             0,
				// OpCode:           0,
				// Mask:             0,
				PayloadLen: 5,
				// PayloadExtendLen: 0,
				// MaskingKey:       0,
				Payload: []byte("hello"),
			},
			wantErr: false,
		},
		{
			name: "case 0",
			fields: fields{
				Fin: 1,
				// RSV1:             0,
				// RSV2:             0,
				// RSV3:             0,
				// OpCode:           0,
				Mask:       1,
				PayloadLen: 5,
				// PayloadExtendLen: 0,
				// MaskingKey:       0,
				Payload: []byte("hello"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frm := &Frame{
				Fin:              tt.fields.Fin,
				RSV1:             tt.fields.RSV1,
				RSV2:             tt.fields.RSV2,
				RSV3:             tt.fields.RSV3,
				OpCode:           tt.fields.OpCode,
				Mask:             tt.fields.Mask,
				PayloadLen:       tt.fields.PayloadLen,
				PayloadExtendLen: tt.fields.PayloadExtendLen,
				MaskingKey:       tt.fields.MaskingKey,
				Payload:          tt.fields.Payload,
			}
			if err := frm.valid(); (err != nil) != tt.wantErr {
				t.Errorf("Frame.valid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_Frame_setPayload_over65535(t *testing.T) {
	frm := mockFrame(nil)
	over65535Byte := strings.Repeat("s", 65535+10)
	frm.setPayload([]byte(over65535Byte))

	t.Log(frm.PayloadLen, frm.PayloadExtendLen, len(over65535Byte))

	assert.Equal(t, uint16(127), frm.PayloadLen)
	assert.Equal(t, len(over65535Byte), len(frm.Payload))
	assert.Equal(t, uint64(65535+10), frm.PayloadExtendLen)

	p := encodeFrameTo(frm)
	dstFrm, err := decodeToFrame(p)
	if err != nil {
		// assert.Empty(t, err)
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, frm, dstFrm)
}

func Test_Frame_setPayload_over125less65535(t *testing.T) {
	SetDebug(true)

	frm := mockFrame(nil)
	less65535byte := strings.Repeat("s", 65535)
	frm.setPayload([]byte(less65535byte))

	t.Log("origin frame: ", frm.PayloadLen, frm.PayloadExtendLen, len(less65535byte))

	assert.Equal(t, uint16(126), frm.PayloadLen)
	assert.Equal(t, len(less65535byte), len(frm.Payload))
	assert.Equal(t, uint64(65535), frm.PayloadExtendLen)

	p := encodeFrameTo(frm)
	// debugPrintEncodedFrame(p[:8])

	dstFrm, err := decodeToFrame(p)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Log("decoded frame: ", frm.PayloadLen, frm.PayloadExtendLen, len(less65535byte))
	assert.Equal(t, frm, dstFrm)
	// assert.Equal(t, frm.MaskingKey, dstFrm.MaskingKey)
	// assert.Equal(t, frm.Mask, dstFrm.Mask)
	// assert.Equal(t, frm.Payload, dstFrm.Payload)
	// assert.Equal(t, frm.PayloadLen, dstFrm.PayloadLen)
	// assert.Equal(t, frm.PayloadExtendLen, dstFrm.PayloadExtendLen)
}

func Test_fragmentDataFrames(t *testing.T) {
	data := make([]byte, 0, 65535*2+20)
	part1 := strings.Repeat("a", 65535)
	part2 := strings.Repeat("b", 65535)
	part3 := strings.Repeat("c", 20)

	data = append(data, part1...)
	data = append(data, part2...)
	data = append(data, part3...)

	frames := fragmentDataFrames(data, true, opCodeText)

	assert.Equal(t, 3, len(frames))

	assert.Equal(t, uint16(0), frames[0].Fin)
	assert.Equal(t, opCodeText, frames[0].OpCode)
	assert.Equal(t, []byte(part1), frames[0].Payload)

	assert.Equal(t, uint16(0), frames[1].Fin)
	assert.Equal(t, opCodeContinuation, frames[1].OpCode)
	assert.Equal(t, []byte(part2), frames[1].Payload)

	assert.Equal(t, uint16(1), frames[2].Fin)
	assert.Equal(t, opCodeContinuation, frames[2].OpCode)
	assert.Equal(t, []byte(part3), frames[2].Payload)
}

func Test_fragmentDataFrames_times(t *testing.T) {
	data := make([]byte, 0, 65535*2)
	part1 := strings.Repeat("a", 65535)
	part2 := strings.Repeat("b", 65535)

	data = append(data, part1...)
	data = append(data, part2...)

	frames := fragmentDataFrames(data, true, opCodeText)

	assert.Equal(t, 2, len(frames))

	assert.Equal(t, uint16(0), frames[0].Fin)
	assert.Equal(t, opCodeText, frames[0].OpCode)
	assert.Equal(t, []byte(part1), frames[0].Payload)

	assert.Equal(t, uint16(1), frames[1].Fin)
	assert.Equal(t, opCodeContinuation, frames[1].OpCode)
	assert.Equal(t, []byte(part2), frames[1].Payload)
}

func Benchmark_encodeFrameTo(b *testing.B) {
	frame := constructFrame(opCodePing, true, true)

	for i := 0; i < b.N; i++ {
		//byts := encodeFrameTo(frame)
		byts := encodeFrameToV2(frame)
		_ = byts
	}
}
