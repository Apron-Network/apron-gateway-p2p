package test

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync"
	"time"
)

func BuildKdhtNetwork(ctx context.Context, bsNodeCount int, clientCount int) ([]*trans_network.Node, []*trans_network.Node) {
	bsNodes := make([]*trans_network.Node, bsNodeCount)
	clientNodes := make([]*trans_network.Node, clientCount)
	bsPeers := trans_network.AddrList{}

	wg := sync.WaitGroup{}
	rendezvous := "foobar"

	startPort := 12345
	port := startPort

	// Build bootstrap nodes
	for i := 0; i < bsNodeCount; i++ {
		config := &trans_network.NodeConfig{
			BootstrapPeers:     nil,
			InternalPort:       port,
			MgmtAddr:           fmt.Sprintf("localhost:%d", port+1),
			ForwardServiceAddr: fmt.Sprintf("localhost:%d", port+2),
		}

		node, err := trans_network.NewNode(ctx, config)
		if err != nil {
			panic(err)
		}

		log.Printf("Setup bs nodes")
		node.SetupServiceBroadcastListener(ctx)
		node.SetProxyStreamHandlers()
		bsNodes[i] = node

		bsPeers.Set(node.NodeAddrStr())
		port += 3

		kdht, err := trans_network.NewKDHT(ctx, *node.Host, nil, &wg)
		if err != nil {
			panic(err)
		}
		go trans_network.Discover(ctx, node, kdht, rendezvous)
		go node.StartMgmtApiServer()
		go node.StartForwardService()
	}

	// Build client nodes
	for i := 0; i < clientCount; i++ {
		fmt.Printf("bootstrap peers: %+v\n", bsPeers)
		config := &trans_network.NodeConfig{
			BootstrapPeers:     bsPeers,
			InternalPort:       port,
			MgmtAddr:           fmt.Sprintf("localhost:%d", port+1),
			ForwardServiceAddr: fmt.Sprintf("localhost:%d", port+2),
		}

		node, err := trans_network.NewNode(ctx, config)
		if err != nil {
			panic(err)
		}

		log.Printf("Setup client nodes")
		node.SetupServiceBroadcastListener(ctx)
		node.SetProxyStreamHandlers()
		clientNodes[i] = node
		port += 3

		kdht, err := trans_network.NewKDHT(ctx, *node.Host, bsPeers, &wg)
		if err != nil {
			panic(err)
		}
		go trans_network.Discover(ctx, node, kdht, rendezvous)
		go node.StartMgmtApiServer()
		go node.StartForwardService()
	}

	return bsNodes, clientNodes
}

func RegisterService(node *trans_network.Node, service *models.ApronService, registerServiceAsRemoteService bool) {
	if registerServiceAsRemoteService {
		node.RegisterRemoteService((*node.Host).ID(), service)
	} else {
		node.RegisterLocalService(service)
	}
}

func StartDemoHttpbinServer() string {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqDumpBytes, err := httputil.DumpRequest(r, true)
		internal.CheckError(err)
		_, err = fmt.Fprintf(w, fmt.Sprintf("Req: %q\n", reqDumpBytes))
		internal.CheckError(err)
	}))

	return s.URL
}

func StartDemoWebsocketServer() string {
	var upgrader = websocket.Upgrader{}

	// TODO: simulate stream and echo in different path
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		internal.CheckError(err)
		defer c.Close()

		// Simulate stream function
		go func() {
			for {
				tick := time.Now().Nanosecond()
				err = c.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(tick)))
				internal.CheckError(err)
				time.Sleep(time.Second)
			}
		}()

		// Simulate echo function
		go func() {
			for {
				_, msgBytes, err := c.ReadMessage()
				internal.CheckError(err)

				respMsg := []byte("ECHO: ")
				respMsg = append(respMsg, msgBytes...)

				err = c.WriteMessage(websocket.TextMessage, respMsg)
			}
		}()

		select {}
	}))
	return "ws" + strings.TrimPrefix(s.URL, "http")
}
