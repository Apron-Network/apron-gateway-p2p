package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"bufio"
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"strings"
	"time"
)

var HttpMethodInternalToStrMapping = map[models.ApronServiceRequest_HttpMethod]string{
	models.ApronServiceRequest_GET:    "GET",
	models.ApronServiceRequest_PUT:    "PUT",
	models.ApronServiceRequest_POST:   "POST",
	models.ApronServiceRequest_DELETE: "DELETE",
}

var HttpMethodStringToInternalMapping = map[string]models.ApronServiceRequest_HttpMethod{
	"GET":    models.ApronServiceRequest_GET,
	"PUT":    models.ApronServiceRequest_PUT,
	"POST":   models.ApronServiceRequest_POST,
	"DELETE": models.ApronServiceRequest_DELETE,
}

type Node struct {
	Host   *host.Host
	Config *TransNetworkConfig

	ps                    *pubsub.PubSub
	broadcastServiceTopic *pubsub.Topic
	serviceBroadcastSub   *pubsub.Subscription
	selfID                peer.ID

	// Save service has no name but only key. The key for service should be uniq
	services map[string]models.ApronService

	// Save service with name, which may contain multiple service with same name, will be used for LB later
	namedServices map[string][]models.ApronService

	// Mapping of service and peer id, the key for this mapping is service key, and value is peer id, to locate service while receiving forward request
	servicePeerMapping map[string]peer.ID
}

func NewNode(ctx context.Context, config *TransNetworkConfig) (*Node, error) {
	// r := rand.Reader
	// priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	// if err != nil {
	//	return nil, err
	// }
	//
	// addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.ConnectPort))
	// h, err := libp2p.New(ctx, libp2p.ListenAddrs(addr), libp2p.Identity(priv))
	h, err := libp2p.New(context.Background(), libp2p.ListenAddrs())
	if err != nil {
		return nil, err
	}

	return &Node{
		Host:               &h,
		Config:             config,
		services:           map[string]models.ApronService{},
		namedServices:      map[string][]models.ApronService{},
		servicePeerMapping: map[string]peer.ID{},
	}, nil
}

// SetupServiceBroadcastListener set subscriber of service broadcast
func (n *Node) SetupServiceBroadcastListener(ctx context.Context) {
	var err error
	n.ps, err = pubsub.NewGossipSub(ctx, *n.Host)
	internal.CheckError(err)

	n.broadcastServiceTopic, err = n.ps.Join(BroadcastServiceChannel)
	internal.CheckError(err)

	n.serviceBroadcastSub, err = n.broadcastServiceTopic.Subscribe()
	internal.CheckError(err)

	n.selfID = (*n.Host).ID()

	go n.StartListeningOnServiceBroadcast(ctx)
}

// StartListeningOnServiceBroadcast is a infinity loop which listens to service broadcast subscriptions.
// After receiving the message, the service will be added to remote service list of this node,
// which will be queried while receiving service request
func (n *Node) StartListeningOnServiceBroadcast(ctx context.Context) {
	for {
		msg, err := n.serviceBroadcastSub.Next(ctx)
		internal.CheckError(err)

		if msg.ReceivedFrom == n.selfID {
			continue
		}

		fmt.Printf("Received msg: %+v\n", msg)
	}
}

// BroadcastService broad local service to the network with configured topic,
// so all nodes subscribed to the topic can update its local cache data
func (n *Node) BroadcastService(ctx context.Context, msg string) error {
	return n.broadcastServiceTopic.Publish(ctx, []byte(msg))
}

func (n *Node) RegisterLocalService(service *models.ApronService) {
	n.services[service.Id] = *service
	n.servicePeerMapping[service.Id] = n.selfID
}

func (n *Node) RegisterRemoteService(peerId peer.ID, service *models.ApronService) {
	n.services[service.Id] = *service
	n.servicePeerMapping[service.Id] = peerId
}

func (n *Node) NodeAddrStr() string {
	bsIdStr := (*n.Host).ID().Pretty()
	bsNodeAddr := (*n.Host).Addrs()[0]
	return fmt.Sprintf("%s/p2p/%s", bsNodeAddr, bsIdStr)
}

// TODO: when new node join the network, it require to connect to all gw, and get services registered on the node

// NewProxyRequest send proxy request to node associated with service.
// The request is encapsulated in a proto message, and after final message sent, a \n will be sent to note message done,
// this should be changed to some other policy in future.
func (n *Node) NewProxyRequest(peerId peer.ID, proxyReq *models.ApronServiceRequest) {
	s, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyRequestStream))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Stream: %+v\n", s)

	// Build request package with passed in request
	reqBytes, err := proto.Marshal(proxyReq)
	internal.CheckError(err)
	err = WriteBytesViaStream(s, reqBytes)
	internal.CheckError(err)

	// Read response from remote
	respBytes, err := ReadBytesViaStream(s)
	internal.CheckError(err)

	if err != nil {
		panic(err)
	}
	fmt.Printf("Resp: %s\n", string(respBytes))
}

// SetProxyRequestStreamHandler set handler to process proxy request
func (n *Node) SetProxyRequestStreamHandler() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyRequestStream), func(s network.Stream) {
		// TODO: Forward request to remote service, and send respond to invoker via respstream
		// Q: How to get src addr

		msgByte, err := ReadBytesViaStream(s)
		internal.CheckError(err)

		proxyReq := &models.ApronServiceRequest{}
		err = proto.Unmarshal(msgByte, proxyReq)

		fmt.Printf("Read stream: %s\n", proxyReq)

		switch proxyReq.Schema {
		case "http", "https":
			go n.forwardHttpRequest(s, proxyReq)
		case "ws", "wss":
			go n.forwardWebsocketRequest(s, proxyReq)
		default:
			panic(fmt.Errorf("wrong schema: %s", proxyReq.Schema))
		}

	})
}

func (n *Node) forwardHttpRequest(s network.Stream, req *models.ApronServiceRequest) {
	// Build http request and send
	fmt.Printf("%s: %s\n", models.ApronServiceRequest_HttpMethod_name[int32(req.HttpMethod)], req.ServiceUrlWithSchema())

	netClient := &http.Client{
		Timeout: time.Second * 5,
	}

	r, err := http.NewRequest(HttpMethodInternalToStrMapping[req.HttpMethod], req.ServiceUrlWithSchema(), nil)
	resp, err := netClient.Do(r)
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	internal.CheckError(err)

	err = WriteBytesViaStream(s, bodyBytes)
	internal.CheckError(err)
}

func (n *Node) forwardWebsocketRequest(s network.Stream, req *models.ApronServiceRequest) {
	fmt.Printf("Service URL: %s\n", req.ServiceUrlWithSchema())
	ws, _, err := websocket.DefaultDialer.Dial(req.ServiceUrlWithSchema(), nil)
	internal.CheckError(err)
	// defer ws.Close()

	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			internal.CheckError(err)

			fmt.Printf("ws recv: %q\n", msg)

			err = WriteBytesViaStream(s, msg)
			internal.CheckError(err)
		}
	}()
}

// SetProxyRespStreamHandler set handler for response returned from remote gateway
// TODO: seems not using now
func (n *Node) SetProxyRespStreamHandler() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyRespStream), func(s network.Stream) {
		buf := bufio.NewReader(s)
		str, err := buf.ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read stream: %s\n", str)
	})
}

func (n *Node) NewProxyResp(remoteNode *Node, content []byte) {
	s, err := (*n.Host).NewStream(context.Background(), (*remoteNode.Host).ID(), protocol.ID(ProxyRequestStream))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Stream: %+v\n", s)

	_, err = s.Write(content)
	internal.CheckError(err)
}

func (n *Node) StartMgmtApiServer() {
	fasthttp.ListenAndServe(n.Config.MgmtAddr, func(ctx *fasthttp.RequestCtx) {

	})
}

func (n *Node) StartForwardService() {
	fasthttp.ListenAndServe(n.Config.ForwardServiceAddr, func(ctx *fasthttp.RequestCtx) {
		// Parse request URL and split service
		fmt.Printf("Ctx path: %s\n", ctx.Path())
		urlElements := strings.Split(string(ctx.Path()), "/")
		if len(urlElements) < 4 {
			ctx.Error("Wrong request"+
				"URL format", fasthttp.StatusBadRequest)
			return
		}
		srvKey := urlElements[1]
		srvVer := urlElements[2]
		userKey := urlElements[3]
		requestUrl := strings.Join(urlElements[4:], "/")
		// TODO: Handle query params and body
		fmt.Printf("Request service: %s with key: %s, reqUrl: %s, ver: %s\n", srvKey, userKey, requestUrl, srvVer)

		servicePeerId, found := n.servicePeerMapping[srvKey]
		if !found {
			ctx.Error("Service not found", fasthttp.StatusNotFound)
			return
		}

		if servicePeerId == n.selfID {

		} else {
			// TODO: Find service from saved services list, and update request info to that object, then send proxy request
			service, found := n.services[srvKey]
			if !found {
				// Service is in the peer mapping but not in services list, internal error
				ctx.Error("Service data missing, contract help", fasthttp.StatusInternalServerError)
				return
			}

			req := &models.ApronServiceRequest{
				HttpMethod: HttpMethodStringToInternalMapping[string(ctx.Method())],
				ServiceUrl: fmt.Sprintf("%s/%s", service.BaseUrl, requestUrl),
				Schema:     service.Schema,
			}
			fmt.Printf("Req: %+v\n", req)
			n.NewProxyRequest(servicePeerId, req)
		}

		// Find service from node registered service list
	})
}
