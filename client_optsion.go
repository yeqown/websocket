package websocket

import (
	"crypto/tls"
	"net/url"
)

type options struct {
	host     string
	port     string
	schema   string
	path     string
	rawquery string

	// option fields
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

// WithTLS .
func WithTLS(cfg *tls.Config) DialOption {
	return func(do *options) {
		do.tlsConfig = cfg
	}
}
