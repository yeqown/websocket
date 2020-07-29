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
	_, _ = h.Write([]byte(challengeKey))
	_, _ = h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
