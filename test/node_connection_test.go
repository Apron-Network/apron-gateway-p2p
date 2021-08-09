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
		"httpecho": {
			Id:          "demo_httpecho",
			Name:        "demo_httpecho",
			Desc:        "demo httpecho server",
			CreatedAt:   1625711065622,
			UpdatedAt:   1625711065622,
			ExtraDetail: "",
			BaseUrl:     StartDemoHttpbinServer(),
			Schema:      "http",
		},
	}

	wsProvider = map[string]*models.ApronServiceProvider{
		"echo": {
			Id:          "ws_echo",
			Name:        "ws echo",
			Desc:        "echo ws server",
			CreatedAt:   1625711065622,
			UpdatedAt:   1625711065622,
			ExtraDetail: "",
			BaseUrl:     StartDemoWebsocketServer(),
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

	httpbinService := &models.ApronService{
		Id:         clientNodes[0].Config.ForwardServiceAddr,
		DomainName: "localhost",
		Providers: []*models.ApronServiceProvider{
			httpsProvider["httpecho"],
		},
	}
	RegisterService(bsNodes[0], httpbinService, false)
	RegisterService(clientNodes[0], httpbinService, true)

	log.Printf("\nSETUP DONE\n\n")

	// Create test client and send request to local gateway, the gateway should find service info in local cache and forward the request
	netClient := &http.Client{
		Timeout: time.Second * 5,
	}

	patternStr := "apronservicetest"
	reqUrl := fmt.Sprintf("http://%s/v1/testkey/%s", clientNodes[0].Config.ForwardServiceAddr, patternStr)
	log.Printf("Client: Request URL: %s\n", reqUrl)
	resp, err := netClient.Get(reqUrl)
	internal.CheckError(err)
	defer resp.Body.Close()

	// TODO: Verify header info will also be transferred

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	internal.CheckError(err)

	log.Println(resp)

	assert.Equal(t, resp.StatusCode, 200)
	assert.Contains(t, string(bodyBytes), patternStr)

	mgmtUrl := fmt.Sprintf("http://%s/service/report", bsNodes[0].Config.MgmtAddr)
	usageReportResp, err := netClient.Get(mgmtUrl)
	internal.CheckError(err)
	defer usageReportResp.Body.Close()

	usageReportBodyBytes, err := ioutil.ReadAll(usageReportResp.Body)
	internal.CheckError(err)

	assert.Equal(t, usageReportResp.StatusCode, 200)
	assert.Contains(t, string(usageReportBodyBytes), "testkey")
	assert.Contains(t, string(usageReportBodyBytes), "usage")

	// log.Printf("usage report: %q\n", usageReportBodyBytes)
}

func TestWsEchoRequestForward(t *testing.T) {
	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
	time.Sleep(1 * time.Second)

	wsEchoService := &models.ApronService{
		Id:         fmt.Sprintf("localhost%s", clientNodes[0].Config.ForwardServiceAddr),
		DomainName: "localhost",
		Providers: []*models.ApronServiceProvider{
			wsProvider["echo"],
		},
	}

	RegisterService(bsNodes[0], wsEchoService, true)
	RegisterService(clientNodes[0], wsEchoService, false)

	log.Printf("\nSETUP DONE\n\n")

	reqUrl := fmt.Sprintf("ws://%s/v1/testkey/", wsEchoService.Id)
	log.Printf("Request URL: %s\n", reqUrl)

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	c, _, err := dialer.Dial(reqUrl, nil)
	internal.CheckError(err)

	go func() {
		for {
			time.Sleep(2 * time.Second)
			log.Printf("Client: Writing abcdefg to server from client")
			err = c.WriteMessage(websocket.TextMessage, []byte("abcdefg"))
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
