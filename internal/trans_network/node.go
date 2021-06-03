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

		// Send request to service and pass response back
		netClient := &http.Client{
			Timeout: time.Second * 5,
		}

		r, err := http.NewRequest(HttpMethodInternalToStrMapping[proxyReq.HttpMethod], proxyReq.ServiceUrlWithSchema(), nil)
		resp, err := netClient.Do(r)
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		internal.CheckError(err)

		err = WriteBytesViaStream(s, bodyBytes)
		internal.CheckError(err)
	})
}

// handleHttpForwardRequest handles request sent from client side,
// and check whether the request should be sent to remote peer or process locally, and send respond back.
// TODO: Process other method and query params, post body, etc.
func (n *Node) handleHttpForwardRequest(ctx *fasthttp.RequestCtx, peerId peer.ID, req *models.ApronServiceRequest) {
	if peerId == n.selfID {

	} else {
		s, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyRequestStream))
		if err != nil {
			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
			return
		}

		// Build request package with passed in request
		reqBytes, err := proto.Marshal(req)
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

		// TODO: Build structure to save response, should contains all data, includes status code, headers, etc.
		ctx.Write(respBytes)
	}
}

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

		fmt.Printf("servies: %+v\n", n.services)
		fmt.Printf("servicePeerMappings: %+v\n", n.servicePeerMapping)
		servicePeerId, found := n.servicePeerMapping[srvKey]
		if !found {
			ctx.Error("Service not found", fasthttp.StatusNotFound)
			return
		}

		service, found := n.services[srvKey]
		if !found {
			// Service is in the peer mapping but not in services list, internal error
			ctx.Error("Service data missing, contract help", fasthttp.StatusInternalServerError)
			return
		}

		switch service.Schema {
		case "http", "https":
			req := &models.ApronServiceRequest{
				HttpMethod: HttpMethodStringToInternalMapping[string(ctx.Method())],
				ServiceUrl: fmt.Sprintf("%s/%s", service.BaseUrl, requestUrl),
				Schema:     service.Schema,
			}
			n.handleHttpForwardRequest(ctx, servicePeerId, req)
		case "ws", "wss":
			// go n.handleWebsocketForwardRequest(ctx, servicePeerId, req)
		default:
			ctx.Error(fmt.Sprintf("Unknown service schema %s", service.Schema), fasthttp.StatusInternalServerError)
			return
		}

		if servicePeerId == n.selfID {
			// Invoke service directly

		} else {
			// Send service request to remote peer with pre defined stream, and receive response
		}

		// Find service from node registered service list
	})
}
