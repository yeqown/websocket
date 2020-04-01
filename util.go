package websocket

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"io"
)

var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

// inspired by https://github.com/gorilla/websocket/blob/master/util.go#L26
func generateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}

func computeAcceptKey(challengeKey string) string {
	h := sha1.New()
	h.Write([]byte(challengeKey))
	h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// // get high 16bit from uint64
// func bigendian16BitFromUint64(v uint64) uint16 {
// 	v = v >> (64 - 16)
// 	return uint16(v)
// }
