package main

import (
	"log"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	conn, err := websocket.Dial("ws://localhost:8080/echo", "", "http://localhost")
	if err != nil {
		log.Fatal(err)
	}

	message := []byte("hello")

	for {
		_, err = conn.Write(message)
		if err != nil {
			log.Fatal(err)
			continue
		}

		_, err = conn.Read(message)
		if err != nil {
			log.Println("read error:", err)
			break
		}
		log.Printf("recv: %s", message)
		time.Sleep(2 * time.Second)
	}
}
