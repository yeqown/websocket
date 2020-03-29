package main

import (
	"fmt"
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
			if ce, ok := err.(*websocket.CloseError); ok {
				fmt.Printf("close err=%d, %s", ce.Code, ce.Text)
				break
			}
			fmt.Printf("recv failed, err=%v\n", err)
			time.Sleep(1 * time.Second)
		}
		fmt.Printf("messageType=%d, msg=%s\n", mt, msg)
	}
}
