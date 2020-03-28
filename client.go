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
	"net/url"
	"time"

	"github.com/imdario/mergo"
)

// WithTLS .
func WithTLS(cfg *tls.Config) *DialOption {
	return &DialOption{tlsConfig: cfg}
}

// WithContext .
func WithContext(ctx context.Context) *DialOption {
	return &DialOption{ctx: ctx}
}

// DialOption .
type DialOption struct {
	host     string
	port     string
	schema   string
	path     string
	rawquery string

	tlsConfig *tls.Config

	ctx context.Context
}

func (do DialOption) needTLS() bool {
	return do.schema == "wss"
}

// Dial .
func Dial(URL string, opts ...DialOption) (*Conn, error) {
	dst, err := parseURL(URL)
	if err != nil {
		return nil, err
	}

	// mergego.Merge opts
	for _, opt := range opts {
		mergo.Merge(dst, opt)
	}

	if dst.ctx == nil {
		var cancel context.CancelFunc
		dst.ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}

	logger.Debugf("Dial got finnal DialOption is: %+v", dst)
	return dialWithContext(dst.ctx, dst)
}

// parseURL to parse WebSocket URL into DialOption with base options
// includes: schema, host, port, path, rawquery
func parseURL(URL string) (*DialOption, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	do := DialOption{
		schema:   u.Scheme,
		host:     u.Hostname(),
		port:     u.Port(),
		path:     u.Path,
		rawquery: u.RawQuery,
	}

	if do.port == "" {
		switch do.schema {
		case "ws":
			do.port = "80"
		case "wss":
			do.port = "443"
		default:
			return nil, ErrInvalidSchema
		}
	}

	return &do, nil
}

var (
	// ErrInvalidSchema .
	ErrInvalidSchema = errors.New("invalid schema")
)

// dialWithContext to dail connection with server or client
// ws-URI = "ws:" "//" host [ ":" port ] path [ "?"
// wss-URI = "wss:" "//" host [ ":" port ] path [ "?"
//
// 0. prepare [schema, headers]
// 1. build an TCP connection
// 2. send HTTP request to handshake and upgrade
// 3. finish building WebSocket connection
//
func dialWithContext(ctx context.Context, opt *DialOption) (*Conn, error) {
	var (
		httpSchema string = ""
	)

	switch opt.schema {
	case "ws":
		httpSchema = "http"
	case "wss":
		httpSchema = "https"
	default:
		return nil, ErrInvalidSchema
	}

	url := fmt.Sprintf("%s://%s:%s%s?%s", httpSchema, opt.host, opt.port, opt.path, opt.rawquery)
	logger.Debugf("http request url=%s", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.Errorf("dialWithContext failed to generate request, err=%v", err)
		return nil, err
	}

	// set headers, RFC6455 Section-4.1 page[17+]
	reqHeaders := http.Header{}
	reqHeaders.Add("Connection", "Upgrade")
	reqHeaders.Add("Host", fmt.Sprintf("%s:%s", opt.host, opt.port))
	reqHeaders.Add("Upgrade", "websocket")
	reqHeaders.Add("Sec-WebSocket-Version", "13")
	secKey, _ := generateChallengeKey()
	reqHeaders.Add("Sec-WebSocket-Key", secKey)
	// copy reqHeaders into req.Header
	if err = mergo.MapWithOverwrite(&req.Header, reqHeaders); err != nil {
		logger.Errorf("dialWithContext failed to merge request headers, err=%v", err)
		return nil, err
	}
	logger.Debugf("dialWithContext send requet with headers=%+v", req.Header)

	// dial tcp conn
	netconn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", opt.host, opt.port))
	if err != nil {
		logger.Errorf("dialWithContext failed to dial remote over TCP, err=%v", err)
		return nil, err
	}

	if opt.needTLS() {
		// true: TLS handshake
		tlsconn := tls.Client(netconn, opt.tlsConfig)
		netconn = tlsconn
		err = tlsHandshake(tlsconn, opt.tlsConfig)
	}
	if err != nil {
		logger.Errorf("dialWithContext TLS handshake, err=%v", err)
		return nil, err
	}

	// handle newConn
	wsConn, err := newConn(netconn)
	if err != nil {
		logger.Errorf("dialWithContext failed to newConn, err=%v", err)
		return nil, err
	}

	// with context
	// send request and handshake
	if err = req.WithContext(ctx).Write(wsConn.bufWR); err != nil {
		logger.Errorf("dialWithContext failed to write Upgrade Request, err=%v", err)
		return nil, err
	}
	wsConn.bufWR.Flush()

	// handle response
	resp, err := http.ReadResponse(wsConn.bufRD, req)
	if err != nil {
		logger.Errorf("dialWithContext failed to read response, err=%v", err)
		return nil, err
	}
	logger.Debugf("dialWithContext got response status=%d headers=%+v", resp.StatusCode, resp.Header)

	// verify reponse headers
	if keep, err := shouldKeep(resp); !keep {
		logger.Errorf("dialWithContext could not open connection, err=%v", err)
		return nil, err
	}

	return wsConn, nil
}

// shouldKeep to figure out: should client keep current websocket connection
// related to status code and repsone headers
func shouldKeep(resp *http.Response) (keep bool, err error) {
	body := make([]byte, 0, 1024)
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
