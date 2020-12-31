package websocket

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockConn(rw io.ReadWriter) *Conn {
	return &Conn{
		conn: nil,

		bufRD: bufio.NewReaderSize(rw, 65535),
		bufWR: bufio.NewWriter(rw),

		State: Connected,

		isServer: true,
	}
}

func Test_Conn_SendAndRead(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	srcFrm := mockFrame(nil)

	// mock server send, it will no mask, then should mask payload manually
	err := conn.sendFrame(srcFrm)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	// unmask payload to origin
	srcFrm.maskPayload()

	// mock server read client, it will mask
	dstFrm, err := conn.readFrame()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assert.Equal(t, srcFrm, dstFrm)
}

func Test_Conn_SendAndRead_over125(t *testing.T) {
	// SetDebug(true)

	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	payload := strings.Repeat("s", 65535)
	srcFrm := mockFrame([]byte(payload))

	// mock server send, it will no mask, then should mask payload manually
	err := conn.sendFrame(srcFrm)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	// unmask payload to origin
	srcFrm.maskPayload()

	// mock server read client, it will mask
	dstFrm, err := conn.readFrame()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// SetDebug(true)
	debugPrintFrame(srcFrm)
	debugPrintFrame(dstFrm)

	assert.Equal(t, srcFrm, dstFrm)
}

func Test_Conn_SendAndRead_over65535(t *testing.T) {
	SetDebug(true)

	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	payload := strings.Repeat("s", 65535+10)
	srcFrm := mockFrame([]byte(payload))

	// mock server send, it will no mask, then should mask payload manually
	err := conn.sendFrame(srcFrm)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	// unmask payload to origin
	srcFrm.maskPayload()

	// mock server read client, it will mask
	dstFrm, err := conn.readFrame()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assert.Equal(t, srcFrm, dstFrm)
}

func Test_Conn_Fragment(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	frms := mockFragmentFrames(true)

	wantPayload := make([]byte, 0, 18)
	wantMT := MessageType(frms[0].OpCode)

	// mock server send
	for _, v := range frms {
		err := conn.sendFrame(v)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		wantPayload = append(wantPayload, v.Payload...)
	}

	t.Logf("want mt=%d, want payload=%s", wantMT, wantPayload)
	// mock client read
	conn.isServer = false
	mt, payload, err := conn.ReadMessage()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, wantMT, mt)
	assert.Equal(t, wantPayload, payload)
}

func Test_Conn_PingPong(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)

	// server send ping
	if err := conn.Ping(); err != nil {
		t.Error(err)
		t.FailNow()
	}

	// client handle ping
	conn.isServer = false
	pingFrm, err := conn.readFrame()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, pingFrm.OpCode, opCodePing)
	assert.Equal(t, pingFrm.Fin, uint16(1))
	assert.GreaterOrEqual(t, pingFrm.PayloadLen, uint16(0))

	if err := conn.replyPing(pingFrm); err != nil {
		t.Error(err)
		t.FailNow()
	}

	// mock server read
	conn.isServer = true
	pongFrm, err := conn.readFrame()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, pongFrm.OpCode, opCodePong)
	assert.Equal(t, pongFrm.Fin, uint16(1))
	assert.GreaterOrEqual(t, pongFrm.PayloadLen, uint16(0))
	assert.Equal(t, pongFrm.Payload, pingFrm.Payload)
	if err := conn.replyPong(pingFrm); err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func Test_Conn_PongHandler(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	pongHandlerCalledCnt := 0
	conn.SetPongHandler(func(s string) {
		pongHandlerCalledCnt++
	})

	pongFrm := constructControlFrame(opCodePing, true, []byte("pingping"))
	err := conn.replyPong(pongFrm)
	assert.Nil(t, err)
	assert.Equal(t, 1, pongHandlerCalledCnt)
}

func Test_Conn_close(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	closeErr := &CloseError{Code: CloseNormalClosure}
	closeErr.Text = closeErr.Error()

	err := conn.close(closeErr.Code)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	conn.isServer = false
	conn.State = Connected
	frm, err := conn.readFrame()
	if err != nil {
		closeErr2, ok := err.(*CloseError)
		if !ok {
			t.Error(err)
			t.FailNow()
		}

		assert.Equal(t, closeErr.Code, closeErr2.Code)
		assert.Equal(t, closeErr.Error(), closeErr2.Error())
	}
	require.NotNil(t, frm)

	assert.Equal(t, frm.OpCode, opCodeClose)
	assert.Equal(t, frm.Fin, uint16(1))
}

func Test_Conn_sendDataFrame(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	payload := []byte("goodbye")

	if err := conn.sendDataFrame(payload, opCodeContinuation); err == nil {
		t.Error("could not pass invalid opCode")
		t.FailNow()
	}

	if err := conn.sendDataFrame(payload, opCodeText); err != nil {
		t.Error(err)
		t.FailNow()
	}
}
