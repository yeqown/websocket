package websocket

import (
	"crypto/tls"
	"net/url"
)

type options struct {
	// host eg. foo.com
	host string
	// port eg. 80 or 443
	port string
	// schema eg. ws or wss, wss means ws with TLS
	schema string
	// path eg. /ws
	path string
	// rawquery contains parameters to build a connection
	rawquery string

	// tlsConfig with TLS config or not
	tlsConfig *tls.Config
}

func (o options) needTLS() bool {
	return o.schema == "wss"
}

type DialOption func(o *options)

// parseURL to parse WebSocket URL into DialOption with base options
// includes: schema, host, port, path, raw query
func parseURL(URL string) (*options, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	do := options{
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

// WithTLS generate DialOption with tls.Config.
//
//		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
//		&tls.Config{
//			Certificates: []tls.Certificate{cert},
//		}
//
func WithTLS(cfg *tls.Config) DialOption {
	return func(do *options) {
		do.tlsConfig = cfg
	}
}
