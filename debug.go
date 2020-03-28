package websocket

import (
	"fmt"

	"github.com/yeqown/log"
)

var (
	debugMode = true
	logger    = log.NewLogger()
)

func init() {
	logger.SetLogLevel(log.LevelInfo)

	if debugMode {
		logger.SetLogLevel(log.LevelDebug)
	}
}

// print []byte into bits into stdout
func debugPrintEncodedFrame(encoded []byte) {
	if !debugMode {
		return
	}

	gotLen := len(encoded)
	max := (gotLen / 4)
	s := 0

	// println(gotLen, max)
	for i := 0; i < max; i++ {
		s = (i * 4)
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
	if !debugMode {
		return
	}

	logger.Errorf(format, args...)
}

func formatUint16(v uint16) string {
	return fmt.Sprintf("%016b", v)
}
