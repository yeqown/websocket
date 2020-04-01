package main

import (
	"io"
	"net/http"
	"os"

	"github.com/yeqown/log"
	"github.com/yeqown/websocket"
)

var (
	upgrader websocket.Upgrader
	r        io.Reader
	err      error
)

func init() {
	if r, err = os.Open("./example.png"); err != nil {
		panic(err)
	}
}

func download(w http.ResponseWriter, req *http.Request) {

	err = upgrader.Upgrade(w, req, func(conn *websocket.Conn) {
		defer log.Info("conn finished")
		if err := conn.SendBinary(r); err != nil {
			log.Errorf("write binary error: err=%v", err)
			return
		}
		log.Info("sending file finished")
		conn.Close()
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
	http.HandleFunc("/download", download)
	if err = http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
