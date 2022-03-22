package test

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

type HttpbinResponse struct {
	Args    map[string]string `json:"args"`
	Data    string            `json:"data"`
	Files   map[string]string `json:"files"`
	Form    map[string]string `json:"form"`
	Headers map[string]string `json:"headers"`
	Json    map[string]string `json:"json"`
	Method  string            `json:"method"`
	Origin  string            `json:"origin"`
	Url     string            `json:"url"`
}

var (
	ctx = context.Background()

	httpProvider = map[string]*models.ApronServiceProvider{
		"httpecho": {
			BaseUrl: StartDemoHttpbinServer(),
			Schema:  "http",
		},
	}

	wsProvider = map[string]*models.ApronServiceProvider{
		"echo": {
			BaseUrl: StartDemoWebsocketServer(),
			Schema:  "ws",
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
		Id:     "testdemohttpservice",
		Name:   "testDemoHttpService",
		Desc:   "A demo test http service",
		Usage:  "Demo usage",
		UserId: "0x1234567890",
		Providers: []*models.ApronServiceProvider{
			httpProvider["httpecho"],
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
	reqUrl := fmt.Sprintf("http://%s/v1/testkey/%s?a=1", clientNodes[0].Config.ForwardServiceAddr, patternStr)
	log.Printf("Client: Request URL: %s\n", reqUrl)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set("Test-Header", "test-header-value")
	resp, err := netClient.Do(req)
	internal.CheckError(err)
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	internal.CheckError(err)

	httpbinResp := &HttpbinResponse{}
	err = json.Unmarshal(bodyBytes, httpbinResp)
	internal.CheckError(err)

	assert.Equal(t, resp.StatusCode, 200)
	assert.Contains(t, httpbinResp.Url, patternStr)
	assert.Contains(t, httpbinResp.Args, "a")
	assert.Equal(t, httpbinResp.Args["a"], "1")
	assert.Contains(t, httpbinResp.Headers, "Test-Header")
	assert.Equal(t, httpbinResp.Headers["Test-Header"], "test-header-value")

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
		Id:     "testdemowsservice",
		Name:   "testDemoWsService",
		Desc:   "A demo test ws service",
		Usage:  "Demo usage",
		UserId: "0x1234567890",
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
	wsEchoService := &models.ApronService{
		Id:     "testdemowsservice",
		Name:   "testDemoWsService",
		Desc:   "A demo test ws service",
		Usage:  "Demo usage",
		UserId: "0x1234567890",
		Providers: []*models.ApronServiceProvider{
			wsProvider["echo"],
		},
	}

	wsEchoService2 := &models.ApronService{
		Id:     "testdemowsservice2",
		Name:   "testDemoWsService",
		Desc:   "A demo test ws service",
		Usage:  "Demo usage",
		UserId: "0x1234567890",
		Providers: []*models.ApronServiceProvider{
			wsProvider["echo2"],
		},
	}

	// clientNode.RegisterRemoteService((*bsNode.Host).ID(), wsEchoService)
	clientNode.RegisterLocalService(wsEchoService)
	remoteNode.RegisterLocalService(wsEchoService2)

	time.Sleep(2 * time.Second)
}
