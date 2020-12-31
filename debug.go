package websocket

import (
	"fmt"

	"github.com/yeqown/log"
)

var (
	_debug = false
	logger *log.Logger
)

func init() {
	logger, _ = log.NewLogger()
	logger.SetLogLevel(log.LevelInfo)

	if _debug {
		logger.SetLogLevel(log.LevelDebug)
	}
}

// SetDebug . open debug mode
func SetDebug(debug bool) {
	if debug {
		_debug = debug
		logger.SetLogLevel(log.LevelDebug)
	}
}

// debugPrintEncodedFrame print []byte into bits into stdout
func debugPrintEncodedFrame(encoded []byte) {
	if !_debug {
		return
	}

	gotLen := len(encoded)
	max := gotLen / 4
	s := 0

	// println(gotLen, max)
	for i := 0; i < max; i++ {
		s = i * 4
		fmt.Printf("%08b,%08b,%08b,%08b\n", encoded[s], encoded[s+1], encoded[s+2], encoded[s+3])
		// println(s)
	}

	s += 4
	for s < gotLen {
		fmt.Printf("%08b,", encoded[s])
		s++
	}
	fmt.Println()
}

func debugErrorf(format string, args ...interface{}) {
	if !_debug {
		return
	}

	logger.Errorf(format, args...)
}

func formatUint16(v uint16) string {
	return fmt.Sprintf("%016b", v)
}

func debugPrintFrame(frm *Frame) {
	if !_debug {
		return
	}

	var frameFormat = `Frame{
    Fin:				%d,
    RSV1:				%d,
    RSV2:				%d,
    RSV3:				%d,
    OpCode:				%d,
    Mask:				%d,
    PayloadLen:			%d,
    PayloadExtendLen:	%d,
    MaskingKey:			%d,
    Payload: 			%d,
}`

	logger.Debugf(
		frameFormat,
		frm.Fin,
		frm.RSV1,
		frm.RSV2,
		frm.RSV3,
		frm.OpCode,
		frm.Mask,
		frm.PayloadLen,
		frm.PayloadExtendLen,
		frm.MaskingKey,
		len(frm.Payload),
	)
}
