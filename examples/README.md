# Examples

Here display some example of this repo, these code shows how it(WebSocket protocol) works.

### Client

```go
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
```

logs (with debug log), client side:

```sh
➜  use-as-client git:(master) ✗ go run main.go
2020/03/29 01:13:05 client.go:74: [DEBUG] Dial got finnal DialOption is: &{host:localhost port:8080 schema:ws path:/echo rawquery: tlsConfig:<nil> ctx:0xc00008e2a0}
2020/03/29 01:13:05 client.go:137: [DEBUG] http request url=http://localhost:8080/echo?
2020/03/29 01:13:05 client.go:157: [DEBUG] dialWithContext send requet with headers=map[Connection:[Upgrade] Host:[localhost:8080] Sec-Websocket-Key:[2ki1e2eHHmXcs8Nd8S4cdA==] Sec-Websocket-Version:[13] Upgrade:[websocket]]
2020/03/29 01:13:05 client.go:198: [DEBUG] dialWithContext got response status=101 headers=map[Connection:[Upgrade] Date:[Sat, 28 Mar 2020 17:13:04 GMT] Sec-Websocket-Accept:[E3jg6n2QCE+v02EOoBo+uwh6+FQ=] Server:[EchoExample] Upgrade:[websocket]]
2020/03/29 01:13:05 protocol.go:334: [DEBUG] init: &{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:0 PayloadExtendLen:0 MaskingKey:2596996162 Payload:[]}
2020/03/29 01:13:05 protocol.go:157: [DEBUG] Frame.setPayload got frm.Payload=[104 101 108 108 111]
2020/03/29 01:13:05 protocol.go:336: [DEBUG] with payload: &{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:0 PayloadExtendLen:0 MaskingKey:2596996162 Payload:[242 174 104 46 245]}
2020/03/29 01:13:05 protocol.go:338: [DEBUG] calc payload len: &{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:5 PayloadExtendLen:0 MaskingKey:2596996162 Payload:[242 174 104 46 245]}
2020/03/29 01:13:05 conn.go:194: [DEBUG] Conn.sendFrame with frame=&{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:5 PayloadExtendLen:0 MaskingKey:2596996162 Payload:[242 174 104 46 245]}
2020/03/29 01:13:05 conn.go:100: [DEBUG] Conn.readFrame got frmWithoutPayload=&{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:0 PayloadLen:5 PayloadExtendLen:0 MaskingKey:0 Payload:[]}
2020/03/29 01:13:05 conn.go:162: [DEBUG] c.read(5) into payload data
2020/03/29 01:13:05 conn.go:171: [DEBUG] got payload=hello
2020/03/29 01:13:05 protocol.go:157: [DEBUG] Frame.setPayload got frm.Payload=[104 101 108 108 111]
messageType=1, msg=hello
2020/03/29 01:13:08 protocol.go:334: [DEBUG] init: &{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:0 PayloadExtendLen:0 MaskingKey:4039455774 Payload:[]}
2020/03/29 01:13:08 protocol.go:157: [DEBUG] Frame.setPayload got frm.Payload=[104 101 108 108 111]
2020/03/29 01:13:08 protocol.go:336: [DEBUG] with payload: &{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:0 PayloadExtendLen:0 MaskingKey:4039455774 Payload:[152 160 88 114 159]}
2020/03/29 01:13:08 protocol.go:338: [DEBUG] calc payload len: &{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:5 PayloadExtendLen:0 MaskingKey:4039455774 Payload:[152 160 88 114 159]}
2020/03/29 01:13:08 conn.go:194: [DEBUG] Conn.sendFrame with frame=&{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:5 PayloadExtendLen:0 MaskingKey:4039455774 Payload:[152 160 88 114 159]}
2020/03/29 01:13:08 conn.go:100: [DEBUG] Conn.readFrame got frmWithoutPayload=&{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:0 PayloadLen:5 PayloadExtendLen:0 MaskingKey:0 Payload:[]}
2020/03/29 01:13:08 conn.go:162: [DEBUG] c.read(5) into payload data
2020/03/29 01:13:08 conn.go:171: [DEBUG] got payload=hello
2020/03/29 01:13:08 protocol.go:157: [DEBUG] Frame.setPayload got frm.Payload=[104 101 108 108 111]
messageType=1, msg=hello
^Csignal: interrupt
➜  use-as-client git:(master) ✗ 
```

logs, server-side:
```sh
➜  websocket git:(master) ✗ go run third_ws_server.go
conn done
recv: hello
recv: hello
read error: websocket: close 1006 (abnormal closure): unexpected EOF
```

### Server 

```go
package main

import (
	"net/http"

	"github.com/yeqown/log"
	"github.com/yeqown/websocket"
)

var upgrader websocket.Upgrader

func echo(w http.ResponseWriter, req *http.Request) {
	err := upgrader.Upgrade(w, req, func(conn *websocket.Conn) {
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				if closeErr, ok := err.(*websocket.CloseError); ok {
					log.Warnf("conn closed, beacuse=%v", closeErr)
					break
				}
				log.Errorf("read error, err=%v", err)
				continue
			}
			log.Infof("recv: mt=%d, msg=%s", mt, message)
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

func main() {
	http.HandleFunc("/echo", echo)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
```

```log
➜  use-as-server git:(master) ✗ go run main.go
2020/03/30 14:17:34 server.go:104: [DEBUG] Upgrader.Upgrade hackHandshakeResponse finished
2020/03/30 14:17:34 main.go:38: [INFO] conn upgrade done
2020/03/30 14:17:36 conn.go:118: [DEBUG] Conn.readFrame got frmWithoutPayload=&{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:1 PayloadLen:12 PayloadExtendLen:0 MaskingKey:0 Payload:[]}
2020/03/30 14:17:36 conn.go:179: [DEBUG] c.read(12) into payload data
2020/03/30 14:17:36 conn.go:188: [DEBUG] got payload=w�Dl�R
                                                           s�X
2020/03/30 14:17:36 protocol.go:203: [DEBUG] Frame.setPayload got frm.Payload=[14 119 243 68 19 108 227 82 11 115 246 88]
2020/03/30 14:17:36 main.go:21: [INFO] recv: mt=1, msg=thisisyeqllo
2020/03/30 14:17:36 protocol.go:203: [DEBUG] Frame.setPayload got frm.Payload=[116 104 105 115 105 115 121 101 113 108 108 111]
2020/03/30 14:17:36 conn.go:217: [DEBUG] Conn.sendFrame with frame=&{Fin:1 RSV1:0 RSV2:0 RSV3:0 OpCode:1 Mask:0 PayloadLen:12 PayloadExtendLen:0 MaskingKey:0 Payload:[116 104 105 115 105 115 121 101 113 108 108 111]}
^Csignal: interrupt
➜  use-as-server git:(master) ✗
```