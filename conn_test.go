package websocket

import (
	"bufio"
	"io"
	// "github.com/stretchr/testify"
)

func mockConn(wr io.Writer, rd io.Reader) *Conn {
	return &Conn{
		conn: nil,

		bufRD: bufio.NewReader(rd),
		bufWR: bufio.NewWriter(wr),

		State: Connected,
	}
}

// TODO: more testcase of Conn
