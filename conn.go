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
	"encoding/binary"
	"io"
	"net"
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

// Conn . implement net.Conn by wrap an TCP connection
type Conn struct {
	conn net.Conn

	bufRD *bufio.Reader
	bufWR *bufio.Writer

	State ConnState
}

// TODO do more work to deal with TCP packet
func newConn(netconn net.Conn) (*Conn, error) {
	c := Conn{
		conn: netconn,

		bufRD: bufio.NewReader(netconn),
		bufWR: bufio.NewWriter(netconn),

		State: Connecting,
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
		p, err = c.read(6)
		if err != nil {
			debugErrorf("Conn.readFrame failed to c.read(header), err=%v", err)
			return nil, err
		}
		payloadExtendLen = uint64(binary.BigEndian.Uint16(p[:2]))
		remaining = payloadExtendLen
	case 127:
		// has 64bit + 32bit = 12B
		p, err = c.read(12)
		if err != nil {
			debugErrorf("Conn.readFrame failed to c.read(header), err=%v", err)
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

	// TODO: valid frame format
	// handle with close, ping, pong frame
	switch frmWithoutPayload.OpCode {
	case opCodeText, opCodeBinary:
		// TODO: support binary data format
		// data frame, pass
	case opCodePing:
		c.handlePing()
		return frmWithoutPayload, nil
	case opCodePong:
		c.handlePong()
		return frmWithoutPayload, nil
	case opCodeClose:
		err = c.handleClose(frmWithoutPayload)
		return frmWithoutPayload, err
	case opCodeContinuation:
		// TODO: support fragment
	}

	logger.Debugf("c.read(%d) into payload data", remaining)
	// FIXME: big remaining(uint64) cast loss precision
	// read blocked here
	payload, err := c.read(int(remaining))
	if err != nil {
		debugErrorf("Conn.readFrame failed to c.read(payload), err=%v", err)
		return nil, err
	}

	logger.Debugf("got payload=%s", payload)
	return frmWithoutPayload.setPayload(payload), nil
}

// sendDataFrame .
// send data frame [text, binary]
func (c *Conn) sendDataFrame(data []byte) (err error) {
	frm := constructDataFrame(data)
	if err = c.sendFrame(frm); err != nil {
		debugErrorf("c.send failed to c.sendFrame err=%v", err)
		return
	}
	return nil
}

// sendControlFrame .
// send control frame [ping, pong, close, continuation]
func (c *Conn) sendControlFrame(opcode OpCode) (err error) {
	frm := constructControlFrame(opcode)
	if err = c.sendFrame(frm); err != nil {
		debugErrorf("c.send failed to c.sendFrame err=%v", err)
		return
	}
	return nil
}

func (c *Conn) sendFrame(frm *Frame) (err error) {
	logger.Debugf("Conn.sendFrame with frame=%+v", frm)
	data := encodeFrameTo(frm)
	// debugPrintEncodedFrame(data)
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
	msg = frm.Payload

	return
}

// SendMessage .
func (c *Conn) SendMessage(text string) (err error) {
	return c.sendDataFrame([]byte(text))
}

// TODO:
func (c *Conn) handlePing() (err error) {
	return nil
}

// TODO:
func (c *Conn) handlePong() (err error) {
	return nil
}

// handle close frame
// to READ close code and text info
func (c *Conn) handleClose(frm *Frame) (err error) {
	p, err := c.read(int(frm.PayloadLen))
	if err != nil {
		debugErrorf("Conn.readFrame failed to c.read(header), err=%v", err)
		return err
	}

	code := binary.BigEndian.Uint16(p[:2])
	message := p[2:]
	err = &CloseError{
		Code: int(code),
		Text: string(message),
	}
	logger.Debugf("c.handleClose got a frame with closeError=%v", err)

	c.close()
	return
}

func (c *Conn) close() (err error) {
	// FIXME: only do following work when Conn is not recving or sending
	// wait other work finishing
	if err = c.sendControlFrame(opCodeClose); err != nil {
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
	if err := c.close(); err != nil {
		debugErrorf("Conn.Close failed to close, err=%v", err)
	}
}

// Connected .
func (c *Conn) Connected() bool {
	return c.State == Connected
}
