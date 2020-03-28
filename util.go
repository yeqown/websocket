package websocket

import (
	"crypto/rand"
	"encoding/base64"
	"io"
)

// inspired by https://github.com/gorilla/websocket/blob/master/util.go#L26
func generateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}
