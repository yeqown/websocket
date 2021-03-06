## WebSockt (Beta)

[![Go Report Card](https://goreportcard.com/badge/github.com/yeqown/websocket)](https://goreportcard.com/report/github.com/yeqown/websocket) [![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/yeqown/websocket)

this is an Go implementation of WebSocket protocol. just for study, **DO NOT USING IN PRODUCTION !!!**, so it will be easy to read and **SIMPLE**.

> go version: 1.14.1, os: darwin-10.15.4

### Get start

how to use this lib as WebSocket client, as the following code:

```go
// examples/use-as-client/main.go
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
```

```go
// example/use-as-server/main.go
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

### Protocol

The WebSocket Protocol enables two-way communication between a client running untrusted code in a controlled environment to a remote host that has opted-in to communications from that code.  The security model used for this is the origin-based security model commonly used by web browsers.  The protocol consists of an opening handshake followed by basic message framing, layered over TCP.  The goal of this technology is to provide a mechanism for browser-based applications that need two-way communication with servers that does not rely on opening multiple HTTP connections (e.g., using XMLHttpRequest or <\iframe>'s and long polling).

#### Frame (Core)

base frame is following:

<img align="center" src="./static/websocket-frame.svg"/>

[READ MORE](docs/Frame.md)

#### How to work

1. build connection from client
2. accept connection in server side, start ping/pong
3. send and recv message
    3.0 assemble data frame, according to the protocol by RFC6455
    3.1 handle exceptions (server panic; heartbeat loss)
4. close connection

<img align="center" src="./static/websocket-process.svg"/>

### client and server sequence

<img align="center" src="./static/websocket-seq.svg">

## References

* https://tools.ietf.org/html/RFC6455
* https://github.com/abbshr/abbshr.github.io/issues/22