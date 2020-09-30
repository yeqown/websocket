package websocket

import (
	"context"
	"testing"
	"time"
)

func Test_dialWithContext(t *testing.T) {
	// go startWebsocketServer()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	do := &options{
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

// 	do := &options{
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
