package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/yeqown/websocket"
)

func main() {
	// I want to use like this:
	//
	//
	var (
		conn *websocket.Conn
		err  error
	)
	conn, err = websocket.Dial("ws://localhost:8080/echo")
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			if err = conn.SendMessage("hello"); err != nil {
				fmt.Printf("send failed, err=%v\n", err)
			}
			time.Sleep(3 * time.Second)
		}
	}()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			fmt.Printf("recv failed, err=%v\n", err)
		}
		fmt.Printf("messageType=%d, msg=%s\n", mt, msg)
	}
}
