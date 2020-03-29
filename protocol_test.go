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
			},
			want: &Frame{
				Fin:              1,
				RSV1:             0,
				RSV2:             0,
				RSV3:             0,
				OpCode:           opCodeText,
				Mask:             1,
				PayloadLen:       0,
				PayloadExtendLen: 0,
				MaskingKey:       0xF367,
				Payload:          nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructFrame(tt.args.opcode, tt.args.finnal)
			if tt.want.Mask == 1 {
				tt.want.MaskingKey = got.MaskingKey
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_constructDataFrame(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want *Frame
	}{
		{
			name: "case 0",
			args: args{
				data: []byte("hello"),
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
			got := constructDataFrame(tt.args.data)
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
			got := constructControlFrame(tt.args.opcode)
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
