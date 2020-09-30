package websocket

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseURL(t *testing.T) {
	type args struct {
		URL string
	}
	tests := []struct {
		name    string
		args    args
		want    *options
		wantErr bool
	}{
		{
			name: "case 0",
			args: args{
				URL: "ws://www.baidu.com",
			},
			want: &options{
				host:     "www.baidu.com",
				port:     "80",
				schema:   "ws",
				path:     "",
				rawquery: "",
			},
			wantErr: false,
		},
		{
			name: "case 1",
			args: args{
				URL: "ws://www.baidu.com:1234/path?query=q",
			},
			want: &options{
				host:     "www.baidu.com",
				port:     "1234",
				schema:   "ws",
				path:     "/path",
				rawquery: "query=q",
			},
			wantErr: false,
		},
		{
			name: "case 2",
			args: args{
				URL: "wss://www.baidu.com:4433/path?query=q",
			},
			want: &options{
				host:     "www.baidu.com",
				port:     "4433",
				schema:   "wss",
				path:     "/path",
				rawquery: "query=q",
			},
			wantErr: false,
		},
		{
			name: "case 3",
			args: args{
				URL: "wsx://www.baidu.com",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURL(tt.args.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.Equal(t, tt.want, got) {
				t.Errorf("parseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithTLS(t *testing.T) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	do := options{}
	optWithTLS := WithTLS(tlsConfig)
	optWithTLS(&do)

	assert.Equal(t, tlsConfig, do.tlsConfig)
}
