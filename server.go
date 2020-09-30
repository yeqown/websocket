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
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
)

// HandshakeError .
type HandshakeError struct {
	Text string
}

func (e HandshakeError) Error() string {
	return fmt.Sprintf("HandshakeError(Text=%s)", e.Text)
}

func newHandshakeError(reason string) HandshakeError {
	return HandshakeError{Text: "websocket: the client is not using the websocket protocol: " + reason}
}

// Upgrader std.HTTP / fasthttp / gin etc
type Upgrader struct {
	CheckOrigin func(req *http.Request) bool
}

// Upgrade handle websocket upgrade request
//
// NOTICE: why returnError and hackHandshakeResponse both exists:
// https://stackoverflow.com/questions/32657603/why-do-i-get-the-error-message-http-response-write-on-hijacked-connection
//
// TODO: set and timeout context ?
func (ug Upgrader) Upgrade(w http.ResponseWriter, req *http.Request, fn func(conn *Conn)) error {
	// check METHOD == GET
	if req.Method != http.MethodGet {
		debugErrorf("Upgrader.Upgrade handshake got method=%s is not GET", req.Method)
		return ug.returnError(w, http.StatusMethodNotAllowed, newHandshakeError("method not allowed").Error())
	}

	// handshake check according to RFC6455
	// almost checking is about headers
	if err := ug.handshakeCheck(w, req); err != nil {
		debugErrorf("Upgrader.Upgrade failed to ug.handshakeCheck, err=%v", err)
		return ug.returnError(w, http.StatusBadRequest, err.Error())
	}

	// check origin
	if ug.CheckOrigin != nil && !ug.CheckOrigin(req) {
		debugErrorf("Upgrader.Upgrade failed to ug.CheckOrigin got false")
		return ug.returnError(w, http.StatusForbidden, newHandshakeError("origin not allowed").Error())
	}

	h, ok := w.(http.Hijacker)
	if !ok {
		debugErrorf("Upgrader.Upgrade failed to cast w => http.Hijacker")
		_ = ug.returnError(w, http.StatusInternalServerError, "not implement http.Hijacker")
		return nil
	}

	var (
		brw     *bufio.ReadWriter
		netconn net.Conn
		err     error
	)

	// get underlying tcp connection
	netconn, brw, err = h.Hijack()
	if err != nil {
		debugErrorf("Upgrader.Upgrade failed to h.Hijack, err=%v", err)
		_ = ug.returnError(w, http.StatusInternalServerError, err.Error())
		return nil
	}
	// _ = brw

	// server verified client handshake then make up the response.
	var respHeaders = http.Header{}
	respHeaders.Set("Connection", "upgrade")
	respHeaders.Set("Upgrade", "websocket")
	challengeKey := req.Header.Get("Sec-WebSocket-Key")
	respHeaders.Set("Sec-WebSocket-Accept", computeAcceptKey(challengeKey))
	// TODO: support Sec-WebSocket-Protocol header

	// finish response and send
	// FIXED: http.Hijacker could not h.Hijack twice
	if err = hackHandshakeResponse(brw.Writer, respHeaders, "101"); err != nil {
		_ = netconn.Close()
		debugErrorf("Upgrader.Upgrade could not write response, err=%v", err)
		return err
	}
	logger.Debugf("Upgrader.Upgrade hackHandshakeResponse finished")

	conn, _ := newConn(netconn, true)
	conn.State = Connected
	// start a goroutine to handle with websocket.Conn
	go func() {
		defer func() {
			if err, ok := recover().(error); ok {
				logger.Errorf("Upgrader.Upgrade fn panic: err=%v", err)
				debug.PrintStack()
			}
		}()

		fn(conn)
	}()

	return nil
}

// handshakeCheck . check request headers and set necessary headers to Response
func (ug Upgrader) handshakeCheck(w http.ResponseWriter, req *http.Request) error {
	h := req.Header.Get("Connection")
	if h != "Upgrade" {
		return newHandshakeError("'upgrade' token not found in 'Connection' header")
	}

	h = req.Header.Get("Upgrade")
	if h != "websocket" {
		return newHandshakeError("'websocket' token not found in 'Upgrade' header")
	}

	h = req.Header.Get("Sec-Websocket-Version")
	if h != "13" {
		return newHandshakeError("websocket: unsupported version: 13 not found in 'Sec-Websocket-Version' header")
	}

	if h = req.Header.Get("Sec-Websocket-Version"); h == "" {
		return newHandshakeError("websocket: not a websocket handshake: 'Sec-WebSocket-Key' header is missing or blank")
	}

	return nil
}

// returnError . will write error into HTTP response and
// return error to http.Handler
func (ug Upgrader) returnError(w http.ResponseWriter, statusCode int, reason string) error {
	err := errors.New(reason)
	http.Error(w, reason, statusCode)
	// w.WriteHeader(statusCode)
	// w.Write([]byte(reason))
	return err
}

// hackHandshakeResponse . assemble HTTP protocol to response because of
// the http.Hijacker, more detail could be found in:
// https://stackoverflow.com/questions/32657603/why-do-i-get-the-error-message-http-response-write-on-hijacked-connection
//
// FIXED: flush data into Connection
func hackHandshakeResponse(buf *bufio.Writer, respHeaders http.Header, body string) (err error) {
	// buf := bytes.NewBuffer(nil)
	_, _ = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	for k, vs := range respHeaders {
		for _, v := range vs {
			_, _ = buf.WriteString(fmt.Sprintf("%s:%v\r\n", k, v))
		}
	}
	_, _ = buf.WriteString("\r\n")
	return buf.Flush()
}
