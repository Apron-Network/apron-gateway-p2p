package test

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"testing"
	"time"
)

var ctx = context.Background()

// func TestNodeConnection(t *testing.T) {
//	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
//
//	time.Sleep(1 * time.Second)
//
//	assert.Equal(t, (*bsNodes[0].Host).Peerstore().Peers().Len(), 2)
//	assert.Equal(t, (*clientNodes[0].Host).Peerstore().Peers().Len(), 2)
// }

// func TestHttpRequestForward(t *testing.T) {
// 	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
// 	time.Sleep(1 * time.Second)
//
// 	bsNode := bsNodes[0]
// 	clientNode := clientNodes[0]
//
// 	// Add demo service. The service is registered as a service in bsNode[0],
// 	// and the test client will send request to clientNode[0],
// 	// the internal p2p network forwards the service to bsNode[0] and return the response.
// 	httpbinService := &models.ApronService{
// 		Id:      "httpbin-demo",
// 		Name:    "httpbin",
// 		BaseUrl: "httpbin.org/anything",
// 		Schema:  "https",
// 		Desc:    "httpbin test service",
// 	}
//
// 	clientNode.RegisterRemoteService((*bsNode.Host).ID(), httpbinService)
//
// 	// Create test client and send request to local gateway, the gateway should find service info in local cache and forward the request
// 	netClient := &http.Client{
// 		Timeout: time.Second * 5,
// 	}
// 	reqUrl := fmt.Sprintf("http://%s/httpbin-demo/v1/testkey/foobar", clientNode.Config.ForwardServiceAddr)
// 	fmt.Printf("Request URL: %s\n", reqUrl)
// 	resp, err := netClient.Get(reqUrl)
// 	internal.CheckError(err)
// 	defer resp.Body.Close()
//
// 	bodyBytes, err := io.ReadAll(resp.Body)
// 	internal.CheckError(err)
//
// 	fmt.Printf("Resp: %+v\n", string(bodyBytes))
//
// 	// TODO: Simulate http forward, find service node in local, and build proxy request and send to remote.
// 	// Next: Process forward request in client node, get service key and find it in remote service list, then forward it.
// }

func TestWsRequestForward(t *testing.T) {
	go StartDemoWebsocketServer()

	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
	time.Sleep(1 * time.Second)

	bsNode := bsNodes[0]
	clientNode := clientNodes[0]

	// Add demo service. The service is registered as a service in bsNode[0],
	// and the test client will send request to clientNode[0],
	// the internal p2p network forwards the service to bsNode[0] and return the response.
	httpbinService := &models.ApronService{
		Id:      "ws-demo",
		Name:    "ws-demo",
		BaseUrl: "65.181.93.134:10429",
		Schema:  "ws",
		Desc:    "Demo ws service",
	}

	clientNode.RegisterRemoteService((*bsNode.Host).ID(), httpbinService)

	reqUrl := fmt.Sprintf("ws://%s/ws-demo/v1/testkey/pms-ws/", clientNode.Config.ForwardServiceAddr)
	fmt.Printf("Request URL: %s\n", reqUrl)

	c, _, err := websocket.DefaultDialer.Dial(reqUrl, nil)
	internal.CheckError(err)
	defer c.Close()

	go func() {
		for {
			_, msg, err := c.ReadMessage()
			internal.CheckError(err)
			fmt.Printf("Receive: %s\n", msg)
		}
	}()
}
