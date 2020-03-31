package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/yeqown/websocket"
)

func main() {
	var (
		conn *websocket.Conn
		err  error

		tlsconfig *tls.Config
	)

	cert, err := tls.LoadX509KeyPair("./ca.crt", "./ca.key")
	if err != nil {
		log.Fatalf("server: loadkeys: %s", err)
	}

	tlsconfig = &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	conn, err = websocket.Dial("wss://localhost:8080/echo", websocket.WithTLS(tlsconfig))
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
