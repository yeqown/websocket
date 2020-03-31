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
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockFrame() *Frame {
	frm := Frame{
		Fin:              1,               // uint8  1 bit
		RSV1:             0,               // uint8  1 bit
		RSV2:             0,               // uint8  1 bit
		RSV3:             0,               // uint8  1 bit
		OpCode:           1,               // OpCode 4 bits
		Mask:             1,               // uint8  1 bit
		PayloadLen:       0,               // uint8  7 bits
		PayloadExtendLen: 0,               // uint64 64 bits
		MaskingKey:       0,               // uint64 32 bits
		Payload:          []byte("hello"), // []byte no limit by RFC6455
	}

	frm.PayloadLen = uint16(len(frm.Payload))
	println("payload len=", frm.PayloadLen, "want=", 5)

	return &frm
}

// mockFragmentFrames .
func mockFragmentFrames() []*Frame {
	frame1 := &Frame{
		Fin: 0,
		// RSV1:             0,
		// RSV2:             0,
		// RSV3:             0,
		OpCode: opCodeText,
		Mask:   1,
		// PayloadLen:       0,
		// PayloadExtendLen: 0,
		// MaskingKey:       0,
		Payload: nil,
	}
	frame1.setPayload([]byte("frame1"))

	frame2 := &Frame{
		Fin: 0,
		// RSV1:             0,
		// RSV2:             0,
		// RSV3:             0,
		OpCode: opCodeContinuation,
		Mask:   1,
		// PayloadLen:       0,
		// PayloadExtendLen: 0,
		// MaskingKey:       0,
		Payload: nil,
	}
	frame2.setPayload([]byte("frame2"))

	frame3 := &Frame{
		Fin: 1,
		// RSV1:   0,
		// RSV2:   0,
		// RSV3:   0,
		OpCode: opCodeContinuation,
		Mask:   1,
		// PayloadLen:       0,
		// PayloadExtendLen: 0,
		// MaskingKey:       0,
		Payload: nil,
	}
	frame3.setPayload([]byte("frame3"))

	return []*Frame{frame1, frame2, frame3}
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
		payloadExtendLen = uint64(binary.BigEndian.Uint16(buf[cur : cur+8]))
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
	src := mockFrame()
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructDataFrame(tt.args.data, tt.args.noMask)
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
		opcode OpCode
		noMask bool
	}
	tests := []struct {
		name string
		args args
		want *Frame
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructControlFrame(tt.args.opcode, tt.args.noMask)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_FrameMaskAndUnmask(t *testing.T) {
	frm := mockFrame()
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
