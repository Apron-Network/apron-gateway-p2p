package test

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"context"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

var (
	ctx           = context.Background()
	httpsProvider = map[string]*models.ApronServiceProvider{
		"httpbin": {
			Id:          "httpbin_test",
			Name:        "httpbin",
			Desc:        "httpbin server",
			CreatedAt:   1625711065622,
			UpdatedAt:   1625711065622,
			ExtraDetail: "",
			BaseUrl:     "https://httpbin.org/anything",
			Schema:      "https",
		},
	}

	wsProvider = map[string]*models.ApronServiceProvider{
		"echo": {
			Id:          "ws_echo",
			Name:        "ws echo",
			Desc:        "echo ws server from jmalloc/echo-server",
			CreatedAt:   1625711065622,
			UpdatedAt:   1625711065622,
			ExtraDetail: "",
			BaseUrl:     "ws://localhost:10000",
			Schema:      "ws",
		},
		"stream": {},
	}
)

func TestNodeConnection(t *testing.T) {
	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)

	time.Sleep(1 * time.Second)

	assert.Equal(t, (*bsNodes[0].Host).Peerstore().Peers().Len(), 2)
	assert.Equal(t, (*clientNodes[0].Host).Peerstore().Peers().Len(), 2)
}

func TestHttpRequestForward(t *testing.T) {
	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
	time.Sleep(1 * time.Second)

	bsNode := bsNodes[0]
	clientNode := clientNodes[0]
	remoteNode := bsNodes[0]

	// Add demo service. The service is registered as a service in bsNode[0],
	// and the test client will send request to clientNode[0],
	// the internal p2p network forwards the service to bsNode[0] and return the response.
	httpbinService := &models.ApronService{
		Id:         clientNode.Config.ForwardServiceAddr,
		DomainName: "localhost",
		Providers: []*models.ApronServiceProvider{
			httpsProvider["httpbin"],
		},
	}

	// Adding service here is for testing, should be replaced w/ normal adding service request and internal sync
	clientNode.RegisterRemoteService((*bsNode.Host).ID(), httpbinService)
	remoteNode.RegisterLocalService(httpbinService)

	fmt.Printf("\nSETUP DONE\n\n")

	// Create test client and send request to local gateway, the gateway should find service info in local cache and forward the request
	netClient := &http.Client{
		Timeout: time.Second * 5,
	}

	reqUrl := fmt.Sprintf("http://%s/v1/testkey/babvaasfasrfa", clientNode.Config.ForwardServiceAddr)
	fmt.Printf("Request URL: %s\n", reqUrl)
	resp, err := netClient.Get(reqUrl)
	internal.CheckError(err)
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	internal.CheckError(err)

	fmt.Printf("Status code: %d\n", resp.StatusCode)
	fmt.Printf("Resp in client: %q\n", bodyBytes)
	// TODO: Test post body, query params, form params, header
}

func TestWsEchoRequestForward(t *testing.T) {
	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
	time.Sleep(1 * time.Second)

	bsNode := bsNodes[0]
	clientNode := clientNodes[0]
	remoteNode := bsNodes[0]

	serverStr := fmt.Sprintf("localhost%s", clientNode.Config.ForwardServiceAddr)

	// Add demo service. The service is registered as a service in bsNode[0],
	// and the test client will send request to clientNode[0],
	// the internal p2p network forwards the service to bsNode[0] and return the response.
	wsEchoService := &models.ApronService{
		Id:         serverStr,
		DomainName: "localhost",
		Providers: []*models.ApronServiceProvider{
			wsProvider["echo"],
		},
	}

	clientNode.RegisterRemoteService((*bsNode.Host).ID(), wsEchoService)
	remoteNode.RegisterLocalService(wsEchoService)

	fmt.Printf("\nSETUP DONE\n\n")

	reqUrl := fmt.Sprintf("ws://%s/v1/testkey/", serverStr)
	fmt.Printf("Request URL: %s\n", reqUrl)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	log.Println("Client: Before dial")
	c, _, err := dialer.Dial(reqUrl, nil)
	log.Println("Client: Dial done")
	internal.CheckError(err)

	go func() {
		for {
			time.Sleep(2 * time.Second)
			log.Printf("\n\n\n")
			log.Printf("Client: Writing abcdefg to server from client")
			err = c.WriteMessage(websocket.TextMessage, []byte("abcdefg"))
			internal.CheckError(err)

			err = c.WriteMessage(websocket.TextMessage, []byte("hijklmn"))
			internal.CheckError(err)

			time.Sleep(3 * time.Second)
		}
	}()

	go func() {
		for {
			_, msgBytes, err := c.ReadMessage()
			internal.CheckError(err)
			log.Printf("Client: Receive msg: %q\n", msgBytes)
		}
	}()

	select {}
}

func Test_RegisterLocalService(t *testing.T) {

	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
	time.Sleep(1 * time.Second)

	// bsNode := bsNodes[0]
	clientNode := clientNodes[0]
	remoteNode := bsNodes[0]

	// Add demo service. The service is registered as a service in bsNode[0],
	// and the test client will send request to clientNode[0],
	// the internal p2p network forwards the service to bsNode[0] and return the response.
	serverStr := fmt.Sprintf("localhost%s", clientNode.Config.ForwardServiceAddr)
	wsEchoService := &models.ApronService{
		Id:         serverStr,
		DomainName: "localhost",
		Providers: []*models.ApronServiceProvider{
			wsProvider["echo"],
		},
	}

	serverStr2 := fmt.Sprintf("localhost2%s", clientNode.Config.ForwardServiceAddr)
	wsEchoService2 := &models.ApronService{
		Id:         serverStr2,
		DomainName: "localhost2",
		Providers: []*models.ApronServiceProvider{
			wsProvider["echo2"],
		},
	}

	// clientNode.RegisterRemoteService((*bsNode.Host).ID(), wsEchoService)
	clientNode.RegisterLocalService(wsEchoService)
	remoteNode.RegisterLocalService(wsEchoService2)

	time.Sleep(2 * time.Second)
	fmt.Printf("\nSETUP DONE\n\n")
}
