package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/yeqown/log"
	"github.com/yeqown/websocket"
)

func runServer() {
	var upgrader websocket.Upgrader
	var echo = func(w http.ResponseWriter, req *http.Request) {
		err := upgrader.Upgrade(w, req, func(conn *websocket.Conn) {
			for {
				mt, message, err := conn.ReadMessage()
				if err != nil {
					if closeErr, ok := err.(*websocket.CloseError); ok {
						log.Warnf("conn closed, because=%v", closeErr)
						break
					}
					log.Errorf("read error, err=%v", err)
					continue
				}
				log.Infof("receive: mt=%d, len(msg)=%d", mt, len(message))
				err = conn.SendMessage(string(message))
				if err != nil {
					log.Errorf("write error: err=%v", err)
					break
				}
			}

			log.Info("conn finished")
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

	http.HandleFunc("/echo", echo)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func runClient() {
	var (
		conn *websocket.Conn
		err  error
	)
	conn, err = websocket.Dial("ws://localhost:8080/echo")
	if err != nil {
		panic(err)
	}

	message := strings.Repeat("haha", 65535) // 4 * 65535

	go func() {
		for {
			if err = conn.SendMessage(message); err != nil {
				log.Infof("send failed, err=%v", err)
			}
			time.Sleep(3 * time.Second)
		}
	}()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			if ce, ok := err.(*websocket.CloseError); ok {
				log.Errorf("close err=%d, %s", ce.Code, ce.Text)
				break
			}
			log.Errorf("receive failed, err=%v", err)
			time.Sleep(1 * time.Second)
		}

		log.Infof("[Client]: messageType=%d, len(msg)=%d, want=%d", mt, len(msg), 4*65535)
	}
}

func main() {
	go runServer()
	go runClient()

	select {}
}
