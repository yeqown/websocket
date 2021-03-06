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

// Package websocket .
package websocket

import (
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"strconv"
)

const (
	CloseNormalClosure           = 1000
	CloseGoingAway               = 1001
	CloseProtocolError           = 1002
	CloseUnsupportedData         = 1003
	CloseNoStatusReceived        = 1005
	CloseAbnormalClosure         = 1006
	CloseInvalidFramePayloadData = 1007
	ClosePolicyViolation         = 1008
	CloseMessageTooBig           = 1009
	CloseMandatoryExtension      = 1010
	CloseInternalServerErr       = 1011
	CloseServiceRestart          = 1012
	CloseTryAgainLater           = 1013
	CloseTLSHandshake            = 1015
)

// CloseError .
type CloseError struct {
	Code int
	Text string
}

func (e *CloseError) Error() string {
	s := []byte("websocket: close ")
	s = strconv.AppendInt(s, int64(e.Code), 10)
	switch e.Code {
	case CloseNormalClosure:
		s = append(s, " (normal)"...)
	case CloseGoingAway:
		s = append(s, " (going away)"...)
	case CloseProtocolError:
		s = append(s, " (protocol error)"...)
	case CloseUnsupportedData:
		s = append(s, " (unsupported data)"...)
	case CloseNoStatusReceived:
		s = append(s, " (no status)"...)
	case CloseAbnormalClosure:
		s = append(s, " (abnormal closure)"...)
	case CloseInvalidFramePayloadData:
		s = append(s, " (invalid payload data)"...)
	case ClosePolicyViolation:
		s = append(s, " (policy violation)"...)
	case CloseMessageTooBig:
		s = append(s, " (message too big)"...)
	case CloseMandatoryExtension:
		s = append(s, " (mandatory extension missing)"...)
	case CloseInternalServerErr:
		s = append(s, " (internal server error)"...)
	case CloseTLSHandshake:
		s = append(s, " (TLS handshake error)"...)
	}
	if e.Text != "" {
		s = append(s, ": "...)
		s = append(s, e.Text...)
	}
	return string(s)
}

var (
	// ErrUnexpectedEOF .
	ErrUnexpectedEOF = &CloseError{Code: CloseAbnormalClosure, Text: io.ErrUnexpectedEOF.Error()}
	// ErrInvalidFrame .
	ErrInvalidFrame = &CloseError{Code: CloseProtocolError, Text: "invalid frame: "}
)

// OpCode (4bit) it decides how to parse payload data.
type OpCode uint16

const (
	opCodeContinuation OpCode = 0  // 0x0 denotes a continuation frame .
	opCodeText         OpCode = 1  // 0x1 denotes a text frame
	opCodeBinary       OpCode = 2  // 0x2 denotes a binary frame
	opCodeClose        OpCode = 8  // 0x8 denotes a connection close
	opCodePing         OpCode = 9  // 0x9 denotes a ping
	opCodePong         OpCode = 10 // 0xA denotes a pong
	// opCodeReserved            = 3 - 7   // *  %x3-7 are reserved for further non-control frames
	// opCode                    = 11 - 16 // *  %xB-F are reserved for further control frames
)

const (
	finBitLen        = 1
	rsv1BitLen       = 1
	rsv2BitLen       = 1
	rsv3BitLen       = 1
	opcodeBitLen     = 4
	maskBitLen       = 1
	payloadLenBitLen = 7
	// payloadExtendLenBitLen = 16 / 64
	maskingKeyBitLen = 32

	// headerSize = 3*4B + 2B = 14B = 112 bit (max)
	// headerSize = finBitLen + rsv1BitLen + rsv2BitLen + rsv3BitLen + opcodeBitLen + maskBitLen + payloadLenBitLen + payloadExtendLenBitLen(16/64) + maskingKeyBitLen

	finOffset        = 15 // 1st bit
	rsv1Offset       = 14 // 2nd bit
	rsv2Offset       = 13 // 3rd bit
	rsv3Offset       = 12 // 4th bit
	opcodeOffset     = 8  // 5th-8th bits
	maskOffset       = 7  // 9th bit
	payloadLenOffset = 0  // 10th - 16th bits

	finMask        = 0x8000 // 1000 0000 0000 0000
	rsv1Mask       = 0x4000 // 0100
	rsv2Mask       = 0x2000 // 0010
	rsv3Mask       = 0x1000 // 0001
	opcodeMask     = 0x0F00 // 0000 1111 0000 0000
	maskMask       = 0x0080 // 0000 0000 1000 0000
	payloadLenMask = 0x007F // 0000 0000 0111 1111
)

// Frame create an struct to contains WebSocket base frame data, and
// help to assemble and read data over TCP bytes stream
//
// NOTICE:
// this definition wastes more space to help understand
// and each field should be unexported
//
// !!!!CURRENTLY, THIS FRAME NOT CONSIDER ABOUT FRAGMENT!!!!
//
type Frame struct {
	Fin    uint16 // 1 bit
	RSV1   uint16 // 1 bit, 0
	RSV2   uint16 // 1 bit, 0
	RSV3   uint16 // 1 bit, 0
	OpCode OpCode // 4 bits
	Mask   uint16 // 1 bit

	// Payload length:  7 bits, 7+16 bits, or 7+64 bits
	//
	// if PayloadLen = 0 - 125, actual_payload_length = PayloadLen
	// if PayloadLen = 126, 	actual_payload_length = PayloadExtendLen[:16]
	// if PayloadLen = 127, 	actual_payload_length = PayloadExtendLen[:]
	PayloadLen       uint16 // 7 bits
	PayloadExtendLen uint64 // 64 bits

	MaskingKey uint32 // 32 bits
	Payload    []byte // no limit by RFC6455
}

func (frm *Frame) autoCalcPayloadLen() {
	var (
		payloadLen       = uint64(len(frm.Payload))
		payloadExtendLen uint64
	)
	// auto set payload len and payload extended length
	if payloadLen <= 125 {
		// true: payload length is less than 126
		payloadExtendLen = 0
	} else if payloadLen <= 65535 {
		// true: payload length is bigger than 126 less than
		// 63355 = 2^16
		payloadExtendLen = payloadLen
		// payloadExtendLen = payloadExtendLen
		payloadLen = 126
	} else {
		payloadExtendLen = payloadLen
		payloadLen = 127
	}

	frm.PayloadLen = uint16(payloadLen)
	frm.PayloadExtendLen = payloadExtendLen
}

// genMaskingKey generate random masking-key
func (frm *Frame) genMaskingKey() {
	frm.MaskingKey = rand.Uint32()
}

// type maskMode string

// const (
// 	mask   maskMode = "mask"
// 	unmask maskMode = "unmask"
// )

// setPayload . automatic mask or unmask payload data
func (frm *Frame) setPayload(payload []byte) *Frame {
	frm.Payload = payload
	//if _debug {
	//	if len(payload) > 256 {
	//		logger.Debugf("Frame.setPayload got frm.Payload over 256, so ignore to display")
	//	} else {
	//		logger.Debugf("Frame.setPayload got frm.Payload=%v", frm.Payload)
	//	}
	//}

	if frm.Mask == 1 {
		// true: if mask has been set, then calc masking-key with payload
		frm.maskPayload()
	}

	frm.autoCalcPayloadLen()
	return frm
}

func genMasks(maskingKey uint32) [4]byte {
	return [4]byte{
		byte((maskingKey >> 24) & 0x00FF),
		byte((maskingKey >> 16) & 0x00FF),
		byte((maskingKey >> 8) & 0x00FF),
		byte((maskingKey >> 0) & 0x00FF),
	}
}

// maskPayload to calc payload with mask
//
// Octet i of the transformed data ("transformed-octet-i") is the XOR of
// octet i of the original data ("original-octet-i") with octet at index
// i modulo 4 of the masking key ("masking-key-octet-j"):
//
// j                   = i MOD 4
// transformed-octet-i = original-octet-i XOR masking-key-octet-j
//
func (frm *Frame) maskPayload() {
	// masked := make([]byte, len(frm.Payload))
	masks := genMasks(frm.MaskingKey)
	for i, v := range frm.Payload {
		j := i % 4
		frm.Payload[i] = v ^ masks[j] // ^ means XOR
	}
	// frm.Payload = masked
}

// // unmaskPayload .
// func (frm *Frame) unmaskPayload() {
// 	masks := genMasks(frm.MaskingKey)
// 	for i, v := range frm.Payload {
// 		j := i % 4
// 		frm.Payload[i] = (v ^ masks[j]) // ^ means XOR
// 	}
// }

//// to mark current frame is used as control or data
//func (frm *Frame) isControl() bool {
//	return frm.OpCode == opCodePing || frm.OpCode == opCodePong ||
//		frm.OpCode == opCodeClose || frm.OpCode == opCodeContinuation
//}

// isFinal .
func (frm *Frame) isFinal() bool {
	return frm.Fin == 1
}

func (frm *Frame) valid() error {
	var err = ErrInvalidFrame
	if frm.RSV1 != 0 || frm.RSV2 != 0 || frm.RSV3 != 0 {
		err.Text += "reserved bit is not 0"
		return err
	}

	if frm.Mask == 1 && frm.MaskingKey == 0 {
		err.Text += "masking key not set"
		return err
	}

	// return ErrInvalidFrame
	return nil
}

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

	bufLen += len(frm.Payload)
	return
}

// encodeFrameTo .
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

const (
	// 2B(header) + 4B(maskingKey) = 6B
	minFrameHeaderSize = (finBitLen + rsv1BitLen + rsv2BitLen +
		rsv3BitLen + opcodeBitLen + maskBitLen +
		payloadLenBitLen + maskingKeyBitLen) / 8
)

var (
	// ErrInvalidData .
	ErrInvalidData = errors.New("invalid websocket data frame")
)

// parseFrameHeader . this is used for parse WebSocket frame header
// header should be (headerSize / Byte) = 112bit / 8bit = 14Byte
func parseFrameHeader(header []byte) *Frame {
	var (
		frm   = new(Frame)
		part1 = binary.BigEndian.Uint16(header[:2])
	)

	frm.Fin = (part1 & finMask) >> finOffset
	frm.RSV1 = (part1 & rsv1Mask) >> rsv1Offset
	frm.RSV2 = (part1 & rsv2Mask) >> rsv2Offset
	frm.RSV3 = (part1 & rsv3Mask) >> rsv3Offset
	frm.OpCode = OpCode((part1 & opcodeMask) >> opcodeOffset)
	frm.Mask = (part1 & maskMask) >> maskOffset
	frm.PayloadLen = (part1 & payloadLenMask) >> payloadLenOffset

	return frm
}

// fragmentDataFrames if data is too large so that could not be send in one frame.
// TODO: opcode should opCodeContinuation and opCodeText
func fragmentDataFrames(data []byte, noMask bool, opcode OpCode) []*Frame {
	s := len(data)
	start, end, n := 0, 0, s/65535

	frames := make([]*Frame, 0, n+1)
	for i := 1; i <= n; i++ {
		start, end = (i-1)*65535, i*65535
		frames = append(frames, constructDataFrame(data[start:end], noMask, opCodeContinuation))
	}

	if end < s {
		frames = append(frames, constructDataFrame(data[end:], noMask, opCodeContinuation))
	}

	frames[0].OpCode = opcode
	// DONE: final frame should be FIN
	frames[len(frames)-1].Fin = 1

	return frames
}

// constructDataFrame payload length is less than 65535
// FIXED: default opCodeText, need support binary
func constructDataFrame(payload []byte, noMask bool, opcode OpCode) *Frame {
	// if opcode is opCodeContinuation, means this frame is not the final frame
	final := opcode != opCodeContinuation
	frm := constructFrame(opcode, final, noMask)
	// logger.Debugf("init: %+v", frm)
	frm.setPayload(payload)
	// logger.Debugf("with payload: %+v", frm)
	// frm.autoCalcPayloadLen()
	// logger.Debugf("calc payload len: %+v", frm)
	return frm
}

func constructControlFrame(opcode OpCode, noMask bool, payload []byte) *Frame {
	frm := constructFrame(opcode, true, noMask)
	if len(payload) != 0 {
		frm.setPayload(payload)
		// frm.autoCalcPayloadLen()
	}
	return frm
}

func constructFrame(opcode OpCode, final bool, noMask bool) *Frame {
	var (
		fin  uint16 = 1
		mask uint16 = 1
	)

	if !final {
		fin = 0
	}

	if noMask {
		mask = 0
	}

	frm := Frame{
		Fin:              fin,
		RSV1:             0,
		RSV2:             0,
		RSV3:             0,
		OpCode:           opcode,
		Mask:             mask, // open mask mode
		PayloadLen:       0,    // this will be calc in encodeFrameTo()
		PayloadExtendLen: 0,    // this will be calc in encodeFrameTo()
		MaskingKey:       0,    // masking key generate
	}

	// generate masking key if necessary
	if frm.Mask == 1 {
		(&frm).genMaskingKey()
	}

	return &frm
}
