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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Dial .
func Dial(URL string, opts ...DialOption) (*Conn, error) {
	do, err := parseURL(URL)
	if err != nil {
		return nil, err
	}

	// apply options
	for _, opt := range opts {
		opt(do)
	}
	logger.Debugf("Dial got final DialOption is: %+v", do)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return dialWithContext(ctx, do)
}

var (
	// ErrInvalidSchema .
	ErrInvalidSchema = errors.New("invalid schema")
)

// dialWithContext to dail connection with server or client.
// wsURL = "ws://host[:port]/path?rawquery"
// wssURL = "wss://host[:port]/path?rawquery".
//
// 0. prepare [schema, headers]
// 1. build an TCP connection
// 2. send HTTP request to handshake and upgrade
// 3. finish building WebSocket connection
//
func dialWithContext(ctx context.Context, do *options) (*Conn, error) {
	var (
		schema string
	)

	switch do.schema {
	case "ws":
		schema = "http"
	case "wss":
		schema = "https"
	default:
		return nil, ErrInvalidSchema
	}

	url := fmt.Sprintf("%s://%s:%s%s?%s", schema, do.host, do.port, do.path, do.rawquery)
	logger.Debugf("http request url=%s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.Errorf("dialWithContext failed to generate request, err=%v", err)
		return nil, err
	}

	// set headers, RFC6455 Section-4.1 page[17+]
	reqHeaders := http.Header{}
	reqHeaders.Add("Connection", "Upgrade")
	reqHeaders.Add("Host", fmt.Sprintf("%s:%s", do.host, do.port))
	reqHeaders.Add("Upgrade", "websocket")
	reqHeaders.Add("Sec-WebSocket-Version", "13")
	secKey, _ := generateChallengeKey()
	reqHeaders.Add("Sec-WebSocket-Key", secKey)
	// copy reqHeaders into req.Header

	for k, v := range reqHeaders {
		req.Header.Set(k, v[0])
	}
	logger.Debugf("dialWithContext send request with headers=%+v", req.Header)

	// dial tcp conn
	netconn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", do.host, do.port))
	if err != nil {
		logger.Errorf("dialWithContext failed to dial remote over TCP, err=%v", err)
		return nil, err
	}

	if do.needTLS() {
		// true: TLS handshake
		tlsconn := tls.Client(netconn, do.tlsConfig)
		netconn = tlsconn
		if err = tlsHandshake(tlsconn, do.tlsConfig); err != nil {
			logger.Errorf("dialWithContext TLS handshake, with tlsConfig=%+v err=%v", do.tlsConfig, err)
			return nil, err
		}
	}

	// handle newConn
	conn, err := newConn(netconn, false)
	if err != nil {
		logger.Errorf("dialWithContext failed to newConn, err=%v", err)
		return nil, err
	}

	// with context
	// send request and handshake
	if err = req.WithContext(ctx).Write(conn.bufWR); err != nil {
		logger.Errorf("dialWithContext failed to write Upgrade Request, err=%v", err)
		return nil, err
	}
	_ = conn.bufWR.Flush()

	// handle response
	resp, err := http.ReadResponse(conn.bufRD, req)
	if err != nil {
		logger.Errorf("dialWithContext failed to read response, err=%v", err)
		return nil, err
	}
	logger.Debugf("dialWithContext got response status=%d headers=%+v", resp.StatusCode, resp.Header)

	// verify response headers
	if keep, err := shouldKeep(resp); !keep {
		logger.Errorf("dialWithContext could not open connection, err=%v", err)
		return nil, err
	}

	conn.State = Connected
	return conn, nil
}

// shouldKeep to figure out: should client keep current websocket connection
// related to status code and response headers
func shouldKeep(resp *http.Response) (keep bool, err error) {
	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		logger.Error("shouldKeep could not read response.Body")
		return false, err
	}
	defer resp.Body.Close()

	// check status
	if resp.StatusCode != 101 {
		err = fmt.Errorf("invalid status=%d, response=%s", resp.StatusCode, string(body))
		return
	}

	// check headers
	if h := resp.Header.Get("Sec-WebSocket-Accept"); h == "" {
		err = fmt.Errorf("response=%s", string(body))
		return
	}

	keep = true
	return
}

func tlsHandshake(tlsconn *tls.Conn, cfg *tls.Config) error {
	if err := tlsconn.Handshake(); err != nil {
		return err
	}

	if !cfg.InsecureSkipVerify {
		if err := tlsconn.VerifyHostname(cfg.ServerName); err != nil {
			return err
		}
	}
	return nil
}
