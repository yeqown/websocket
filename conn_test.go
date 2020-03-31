package websocket

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	// "github.com/stretchr/testify"
)

func mockConn(rw io.ReadWriter) *Conn {
	return &Conn{
		conn: nil,

		bufRD: bufio.NewReader(rw),
		bufWR: bufio.NewWriter(rw),

		State: Connected,
	}
}

func Test_Conn_SendAndRead(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	conn := mockConn(buf)
	srcFrm := mockFrame()

	err := conn.sendFrame(srcFrm)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

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
	frms := mockFragmentFrames()

	wantPayload := make([]byte, 0, 18)
	wantMT := MessageType(frms[0].OpCode)

	for _, v := range frms {
		err := conn.sendFrame(v)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		wantPayload = append(wantPayload, v.Payload...)
	}

	t.Logf("want mt=%d, want payload=%s", wantMT, wantPayload)
	mt, payload, err := conn.ReadMessage()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, wantMT, mt)
	assert.Equal(t, wantPayload, payload)
}
