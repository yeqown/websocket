package main

import (
	"fmt"
	"os"

	"github.com/yeqown/websocket"
)

func main() {
	var (
		conn  *websocket.Conn
		err   error
		fd, _ = os.OpenFile("./example-dl.png", os.O_CREATE|os.O_RDWR, 0644)
	)
	conn, err = websocket.Dial("ws://localhost:8080/download")
	if err != nil {
		panic(err)
	}

	for {
		mt, payload, err := conn.ReadMessage()
		if err != nil {
			if ce, ok := err.(*websocket.CloseError); ok {
				fmt.Printf("close err=%d, %s\n", ce.Code, ce.Text)
				break
			}
			fmt.Printf("recv failed, err=%v\n", err)
		}

		fmt.Printf("messageType=%d\n", mt)
		if mt == websocket.BinaryMessage {
			fmt.Println("writing file into disk")
			if _, err := fd.Write(payload); err != nil {
				fmt.Printf("writing file failed, err=%v\n", err)
				continue
			}
			fmt.Println("writing file finished 23333")
		}
	}
}
