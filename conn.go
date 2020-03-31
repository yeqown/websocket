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
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

var (
	ErrMaskNotSet = errors.New("mask is not set")
	ErrMaskSet    = errors.New("mask is set")
)

// ConnState .
type ConnState string

const (
	// Connecting one state of ConnState
	Connecting ConnState = "connecting"
	// Connected one state of ConnState
	Connected ConnState = "connected"
	// Closing one state of ConnState
	Closing ConnState = "closing"
	// Closed one state of ConnState
	Closed ConnState = "closed"
)

// MessageType .
// it is reference to frame.opcode
type MessageType uint8

const (
	// NoFrame .
	NoFrame MessageType = 0
	// TextMessage .
	TextMessage MessageType = MessageType(opCodeText)
	// BinaryMessage .
	BinaryMessage MessageType = MessageType(opCodeBinary)
	// CloseMessage .
	CloseMessage MessageType = MessageType(opCodeClose)
	// PingMessage .
	PingMessage MessageType = MessageType(opCodePing)
	// PongMessage .
	PongMessage MessageType = MessageType(opCodePong)
)

// Conn .
type Conn struct {
	// conn is underlying TCP connection to send and recv byte stream.
	// on client side it's opened by net.Dial(protocol, addr)
	// on server side it can be got by (http.ResponseWrite).(http.Hijacker).Hijack()
	conn net.Conn

	bufRD *bufio.Reader
	bufWR *bufio.Writer

	// State marks Conn current state, and it is basis of controlling the Conn.
	// TODO: maybe import an state machine to manage with
	State ConnState

	// isServer, true means Conn is working on server side
	// false means Conn is working on client side
	// it helps Conn support server-side code
	isServer bool

	// TODO: add flag to mark current Conn is working for transferring 'Application Data'
}

// newConn build an websocket.Conn to handle with websocket.Frame
// there is some different between server side and client.
func newConn(netconn net.Conn, isServer bool) (*Conn, error) {
	c := Conn{
		conn: netconn,

		bufRD: bufio.NewReader(netconn),
		bufWR: bufio.NewWriter(netconn),

		State: Connecting,

		isServer: isServer,
	}

	return &c, nil
}

// read n bytes from conn read buffer
// inspired by gorilla/websocket
func (c *Conn) read(n int) ([]byte, error) {
	p, err := c.bufRD.Peek(n)
	if err == io.EOF {
		err = ErrUnexpectedEOF
		return nil, err
	}
	c.bufRD.Discard(len(p))
	return p, err
}

func (c *Conn) readFrame() (*Frame, error) {
	if !c.Connected() {
		return nil, errors.New("websocket: could not send if state not Connected")
	}

	p, err := c.read(2)
	// this would be blocked, if no data comes
	if err != nil {
		debugErrorf("Conn.readFrame failed to c.read(header), err=%v", err)
		return nil, err
	}

	// parse frame header
	frmWithoutPayload := parseFrameHeader(p)
	logger.Debugf("Conn.readFrame got frmWithoutPayload=%+v", frmWithoutPayload)

	var (
		payloadExtendLen uint64 // this could be non exist
		remaining        uint64
	)

	switch frmWithoutPayload.PayloadLen {
	case 126:
		// has 16bit + 32bit = 6B
		p, err = c.read(2)
		if err != nil {
			debugErrorf("Conn.readFrame failed to c.read(2) payloadlen with 16bit, err=%v", err)
			return nil, err
		}
		payloadExtendLen = uint64(binary.BigEndian.Uint16(p[:2]))
		remaining = payloadExtendLen
	case 127:
		// has 64bit + 32bit = 12B
		p, err = c.read(8)
		if err != nil {
			debugErrorf("Conn.readFrame failed to c.read(8) payloadlen with 16bit, err=%v", err)
			return nil, err
		}
		payloadExtendLen = uint64(binary.BigEndian.Uint16(p[:8]))
		remaining = payloadExtendLen
	default:
		remaining = uint64(frmWithoutPayload.PayloadLen)
	}
	frmWithoutPayload.PayloadExtendLen = payloadExtendLen

	// get masking key
	if frmWithoutPayload.Mask == 1 {
		// only 32bit masking key to read
		p, err = c.read(4)
		if err != nil {
			debugErrorf("Conn.readFrame failed to c.read(header), err=%v", err)
			return nil, err
		}
		frmWithoutPayload.MaskingKey = binary.BigEndian.Uint32(p)
	}

	// valid in common rules
	if err := frmWithoutPayload.valid(); err != nil {
		debugErrorf("Conn.readFrame is not valid(frm.valid) in common rules, err=%v", err)
		c.close(CloseProtocolError)
		return nil, err
	}

	// valid in Conn rules
	if err := c.validFrame(frmWithoutPayload); err != nil {
		debugErrorf("Conn.readFrame is not valid(conn.validFrame) for Conn rules, err=%v", err)
		return nil, err
	}

	// FIXME: big remaining(uint64) cast loss precision
	// read blocked here
	logger.Debugf("Conn.readFrame c.read(%d) into payload data", remaining)
	payload, err := c.read(int(remaining))
	if err != nil {
		debugErrorf("Conn.readFrame failed to c.read(payload), err=%v", err)
		return nil, err
	}
	logger.Debugf("Conn.readFrame got payload=%s then set into frmWithoutPayload", payload)
	frmWithoutPayload.setPayload(payload)

	// handle with close, ping, pong frame
	switch frmWithoutPayload.OpCode {
	case opCodeText, opCodeBinary, opCodeContinuation:
		// DONE: support fragment
		// TODO: support binary data format
	case opCodePing:
		c.handlePing(frmWithoutPayload)
	case opCodePong:
		c.handlePong(frmWithoutPayload)
	case opCodeClose:
		err = c.handleClose(frmWithoutPayload)
	}

	return frmWithoutPayload, err
}

// sendDataFrame .
// send data frame [text, binary]
func (c *Conn) sendDataFrame(data []byte) (err error) {
	frm := constructDataFrame(data, c.isServer)
	if err = c.sendFrame(frm); err != nil {
		debugErrorf("c.send failed to c.sendFrame err=%v", err)
		return
	}
	return nil
}

// sendControlFrame .
// send control frame [ping, pong, close, continuation]

func (c *Conn) sendControlFrame(opcode OpCode, payload []byte) (err error) {
	frm := constructControlFrame(opcode, c.isServer, payload)
	frm.setPayload(payload)
	if err = c.sendFrame(frm); err != nil {
		debugErrorf("c.send failed to c.sendFrame err=%v", err)
		return
	}
	return nil
}

// FIXED could not send while Conn.State is not "connected"
func (c *Conn) sendFrame(frm *Frame) (err error) {
	if !c.Connected() {
		return errors.New("websocket: could not send if state not Connected")
	}

	logger.Debugf("Conn.sendFrame with frame=%+v", frm)
	data := encodeFrameTo(frm)
	_, err = c.bufWR.Write(data)
	if err != nil {
		debugErrorf("c.sendFrame failed to c.bufWR.Write, err=%v", err)
		return err
	}
	err = c.bufWR.Flush()
	return err
}

// ReadMessage . it will block to read message
func (c *Conn) ReadMessage() (mt MessageType, msg []byte, err error) {
	frm, err := c.readFrame()
	if err != nil {
		debugErrorf("Conn.ReadMessage failed to c.readFrame, err=%v", err)
		return NoFrame, nil, err
	}
	mt = MessageType(frm.OpCode)

	// read fragment of frame
	buf := bytes.NewBuffer(nil)
	buf.Write(frm.Payload)
	for !frm.isFinnal() {
		if frm, err = c.readFrame(); err != nil {
			debugErrorf("Conn.ReadMessage failed to c.readFrame, err=%v", err)
			return NoFrame, nil, err
		}
		buf.Write(frm.Payload)
	}

	msg = buf.Bytes()
	return
}

// SendMessage .
// TODO: send with specified messagetType [Text, Binary]
func (c *Conn) SendMessage(text string) (err error) {
	return c.sendDataFrame([]byte(text))
}

// frame MUST contains 125 Byte or less payload
func (c *Conn) handlePing(frm *Frame) (err error) {
	return c.pong(frm.Payload)
}

// frame MUST contains same payload with PING frame payload
func (c *Conn) handlePong(frm *Frame) (err error) {
	// if recv pong frame, do nothing?
	return nil
}

// handle close frame
// to READ close code and text info
func (c *Conn) handleClose(frm *Frame) (err error) {
	code := binary.BigEndian.Uint16(frm.Payload[:2])
	message := frm.Payload[2:]
	err = &CloseError{
		Code: int(code),
		Text: string(message),
	}
	logger.Debugf("c.handleClose got a frame with closeError=%v", err)

	c.close(int(code))
	return
}

// ping .
func (c *Conn) ping() (err error) {
	return c.sendControlFrame(opCodePing, []byte("ping"))
}

// pong .
func (c *Conn) pong(pingPayload []byte) (err error) {
	return c.sendControlFrame(opCodePong, pingPayload)
}

// DONE: add close message to close frame
func (c *Conn) close(closeCode int) (err error) {
	// FIXME: only do following work when Conn is not recving or sending
	// wait other work finishing
	p := make([]byte, 2, 16)
	closeErr := &CloseError{Code: closeCode}
	binary.BigEndian.PutUint16(p[:2], uint16(closeCode))
	p = append(p, []byte(closeErr.Error())...)
	logger.Debugf("c.close sending close frame, payload=%s", p)

	if err = c.sendControlFrame(opCodeClose, p); err != nil {
		debugErrorf("c.handleClose failed to c.sendControlFrame, err=%v", err)
		return
	}

	if c.conn != nil {
		// close underlying TCP connection
		defer c.conn.Close()
	}
	// update Conn's State to 'Closed'
	c.State = Closed
	return nil
}

// Close .
func (c *Conn) Close() {
	if err := c.close(CloseAbnormalClosure); err != nil {
		debugErrorf("Conn.Close failed to close, err=%v", err)
	}
}

// Connected .
func (c *Conn) Connected() bool {
	return c.State == Connected
}

// readFrame to call
func (c *Conn) validFrame(frm *Frame) error {
	if c.isServer {
		// frame from client
		if frm.Mask != 1 {
			return ErrMaskNotSet
		}
	} else {
		// frame from server
		if frm.Mask != 0 {
			return ErrMaskSet
		}
	}

	return nil
}
