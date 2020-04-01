package main

import (
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

// var upgrader = websocket.FastHTTPUpgrader{
// 	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool { return true },
// }

// EchoServer .
func EchoServer(conn *websocket.Conn) {
	var message = make([]byte, 5)

	for {
		_, err := conn.Read(message)
		if err != nil {
			log.Println("read error:", err)
			break
		}
		log.Printf("recv: %s", message)
		_, err = conn.Write(message)
		if err != nil {
			log.Println("write error:", err)
			break
		}
	}
}

func main() {
	srv := websocket.Server{
		Config:  websocket.Config{},
		Handler: EchoServer,
	}

	if err := http.ListenAndServe(":8080", srv); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
