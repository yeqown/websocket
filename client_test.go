package websocket

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_parseURL(t *testing.T) {
	type args struct {
		URL string
	}
	tests := []struct {
		name    string
		args    args
		want    *DialOption
		wantErr bool
	}{
		{
			name: "case 0",
			args: args{
				URL: "ws://www.baidu.com",
			},
			want: &DialOption{
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
			want: &DialOption{
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
			want: &DialOption{
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

func Test_dialWithContext(t *testing.T) {
	// go startWebsocketServer()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	do := &DialOption{
		host:     "127.0.0.1",
		port:     "8080",
		schema:   "ws",
		path:     "/echo",
		rawquery: "",
	}

	_, err := dialWithContext(ctx, do)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

// func Test_sendAndRecv(t *testing.T) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	do := &DialOption{
// 		host:     "localhost",
// 		port:     "8080",
// 		schema:   "ws",
// 		path:     "/echo",
// 		rawquery: "",
// 	}

// 	conn, err := dialWithContext(ctx, do)
// 	if err != nil {
// 		t.Error(err)
// 		t.FailNow()
// 	}
// 	_ = conn
// 	// defer conn.Close()

// 	if err := conn.SendMessage("0000"); err != nil {
// 		t.Error(err)
// 		t.FailNow()
// 	}

// 	mt, msg, err := conn.ReadMessage()
// 	t.Logf("messageType=%d, msg=%s, err=%v", mt, msg, err)
// 	if err != nil {
// 		t.Error(err)
// 		t.FailNow()
// 	}
// }

func TestWithTLS(t *testing.T) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	got := WithTLS(tlsConfig)
	assert.NotEmpty(t, got.TLSConfig)
}
