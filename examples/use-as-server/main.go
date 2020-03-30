package main

import (
	"net/http"

	"github.com/yeqown/log"
	"github.com/yeqown/websocket"
)

var upgrader websocket.Upgrader

func echo(w http.ResponseWriter, req *http.Request) {
	err := upgrader.Upgrade(w, req, func(conn *websocket.Conn) {
		defer conn.Close()
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				log.Errorf("read error, err=%v", err)
				break
			}
			log.Infof("recv: mt=%d, msg=%s", mt, message)
			err = conn.SendMessage(string(message))
			if err != nil {
				log.Errorf("write error: err=%v", err)
				break
			}
		}
	})

	if err != nil {
		log.Errorf("upgrade error, err=%v", err)
		// if _, ok := err.(websocket.HandshakeError); ok {
		// 	log.Errorf(err)
		// }
		return
	}

	log.Infof("conn upgrade done")
}

func main() {
	http.HandleFunc("/echo", echo)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
