package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"apron.network/gateway-p2p/internal"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

var upgrader = &websocket.FastHTTPUpgrader{
	ReadBufferSize:  10240,
	WriteBufferSize: 10240,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

func main() {
	listenAddr := flag.String("listen-addr", ":8978", "listen address of service")
	mode := flag.String("mode", "both", "echo, publish, both")

	fmt.Printf("Listening on %s, mode %s\n", *listenAddr, *mode)
	fasthttp.ListenAndServe(*listenAddr, func(ctx *fasthttp.RequestCtx) {
		if !websocket.FastHTTPIsWebSocketUpgrade(ctx) {
			ctx.Error("Expected WebSocket connection", fasthttp.StatusBadRequest)
			return
		}

		err := upgrader.Upgrade(ctx, func(clientWsConn *websocket.Conn) {
			switch *mode {
			case "echo":
				go startEchoService(clientWsConn)
			case "publish":
				go startPublisherService(clientWsConn)
			case "both":
				go startEchoService(clientWsConn)
				go startPublisherService(clientWsConn)
			default:
				fmt.Println("only echo, publish and both are supported in mode")
				os.Exit(1)
			}
			select {}
		})

		internal.CheckError(err)
	})
}

func startEchoService(conn *websocket.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("WebSocket connection closed:", r)
		}
		conn.Close()
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		fmt.Printf("%+v\n", err)

		if err != nil {
			log.Println("WebSocket read error:", err)
			return
		}

		// write to the WebSocket connection
		err = conn.WriteMessage(messageType, p)
		if err != nil {
			log.Println("WebSocket read error:", err)
			return
		}
	}
}

func startPublisherService(conn *websocket.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("WebSocket connection closed:", r)
		}
		conn.Close()
	}()
	for {
		err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("tick: %d", time.Now().UnixMilli())))
		if err != nil {
			log.Println("WebSocket read error:", err)
			return
		}
		time.Sleep(2 * time.Second)
	}
}
