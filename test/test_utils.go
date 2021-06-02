package test

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/trans_network"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
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
		config := &trans_network.TransNetworkConfig{
			BootstrapPeers:     nil,
			InternalPort:       port,
			MgmtAddr:           fmt.Sprintf(":%d", port+1),
			ForwardServiceAddr: fmt.Sprintf(":%d", port+2),
		}

		node, err := trans_network.NewNode(ctx, config)
		if err != nil {
			panic(err)
		}

		node.SetupServiceBroadcastListener(ctx)
		node.SetupServiceBroadcastListener(ctx)
		node.SetProxyRequestStreamHandler()
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
		config := &trans_network.TransNetworkConfig{
			BootstrapPeers:     bsPeers,
			InternalPort:       port,
			MgmtAddr:           fmt.Sprintf(":%d", port+1),
			ForwardServiceAddr: fmt.Sprintf(":%d", port+2),
		}

		node, err := trans_network.NewNode(ctx, config)
		if err != nil {
			panic(err)
		}

		node.SetupServiceBroadcastListener(ctx)
		node.SetProxyRequestStreamHandler()
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

func DemoWsEcho(w http.ResponseWriter, r *http.Request) {

}
func DemoWsStream(w http.ResponseWriter, r *http.Request) {

}

func StartDemoWebsocketServer() string {
	var upgrader = websocket.Upgrader{}

	// TODO: simulate stream and echo in different path
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		internal.CheckError(err)
		defer c.Close()

		for {
			tick := time.Now().Nanosecond()
			err = c.WriteMessage(websocket.TextMessage, []byte(strconv.Itoa(tick)))
			internal.CheckError(err)
			time.Sleep(time.Second)
		}
	}))
	defer s.Close()

	return "ws" + strings.TrimPrefix(s.URL, "http")
}
