package websocket

import "testing"

func Test_debugPrintEncodedFrame(t *testing.T) {
	debugPrintEncodedFrame([]byte{1, 2, 212, 31, 3, 4, 5, 6, 7, 8})
	// output:
	// 00000001,00000010,11010100,00011111
	// 00000011,00000100,00000101,00000110
	// 00000111,00001000,
}

func Test_formatUint16(t *testing.T) {
	t.Logf("input 12, got=%s, want=00000000,00001100", formatUint16(12))
}
