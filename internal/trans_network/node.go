package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
)

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

/*******************************/
// Self connection related logic
/*******************************/

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

		log.Printf("Received msg: %+v\n", msg)
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
	log.Printf("Reg service id: %s to peer %s\n", service.Id, peerId.String())
	n.services[service.Id] = *service
	n.servicePeerMapping[service.Id] = peerId
	log.Printf("RegRemoteService: servicePeerMappings: %+v\n", n.servicePeerMapping)
}

func (n *Node) NodeAddrStr() string {
	bsIdStr := (*n.Host).ID().Pretty()
	bsNodeAddr := (*n.Host).Addrs()[0]
	return fmt.Sprintf("%s/p2p/%s", bsNodeAddr, bsIdStr)
}

// TODO: when new node join the network, it require to connect to all gw, and get services registered on the node

/*******************************/
// Proxy related logic
/*******************************/

// SetProxyRequestStreamHandler set handler to process proxy request
func (n *Node) SetProxyRequestStreamHandler() {
	(*n.Host).SetStreamHandler(protocol.ID(HttpProxyRequestStream), func(s network.Stream) {

		// TODO: This handler process ApronServiceRequest sent from client side node
		// TODO: then the node try to find the service detail in local services list, then make the request
		// TODO: So the node near client should only pass service name, account id and request params,
		// TODO: and this handler node try to find service detail and build request.
		// TODO: If there are any LB things, doing it here.

		proxyReq, err := ParseProxyReqFromStream(s)
		internal.CheckError(err)
		log.Printf("Read HTTP proxy request from stream: %s\n", proxyReq)

		// Get service detail from local services list and fill missing fields of request
		serviceDetail := n.services[proxyReq.ServiceId]

		clientSideReq := proxyReq.BuildHttpRequest(serviceDetail)
		serviceSideResp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseRequest(clientSideReq)
		defer fasthttp.ReleaseResponse(serviceSideResp)

		// log.Printf("Client side req: %+v", clientSideReq)
		// log.Printf("Service detail in remote node: %+v", serviceDetail)

		err = fasthttp.Do(clientSideReq, serviceSideResp)
		internal.CheckError(err)

		respBody := serviceSideResp.Body()

		err = WriteBytesViaStream(s, respBody)
		internal.CheckError(err)
	})

	(*n.Host).SetStreamHandler(protocol.ID(WsProxyRequestStream), func(s network.Stream) {
		proxyReq, err := ParseProxyReqFromStream(s)
		internal.CheckError(err)
		log.Printf("ServiceSideGateway: Read WS proxy request from stream: %s\n", proxyReq)

		serviceDetail := n.services[proxyReq.ServiceId]

		clientSideReq := proxyReq.BuildHttpRequest(serviceDetail)
		defer fasthttp.ReleaseRequest(clientSideReq)

		dialer := websocket.Dialer{
			HandshakeTimeout: 15 * time.Second,
		}

		log.Printf("ServiceSideGateway: client side req URL: %s", clientSideReq.URI().String())
		serviceWsConn, _, err := dialer.Dial(clientSideReq.URI().String(), nil)
		internal.CheckError(err)

		serviceToProxyErrorCh := make(chan error, 1)
		proxyToServiceErrorCh := make(chan error, 1)
		go ForwardWsMsgToInternalStream(serviceWsConn, &s, serviceToProxyErrorCh)
		go ForwardInternalStreamToWsMsg(&s, serviceWsConn, proxyToServiceErrorCh)

		// for true {
		// 	go func() {
		// 		_, serviceMsgBytes, err := serviceWsConn.ReadMessage()
		// 		internal.CheckError(err)
		// 		WriteBytesViaStream(s, serviceMsgBytes)
		// 	}()
		//
		// 	go func() {
		// 		streamBytes, err := ReadBytesViaStream(s)
		// 		log.Printf("ServiceSideGateway: data read from client side gateway stream: %q\n", streamBytes)
		// 		internal.CheckError(err)
		// 		serviceWsConn.WriteMessage(websocket.TextMessage, streamBytes)
		// 	}()
		// }

		for {
			select {
			case err := <-serviceToProxyErrorCh:
				log.Printf("Error while forwarding service data to proxy: %+v", err)
			case err := <-proxyToServiceErrorCh:
				log.Printf("Error while forwarding proxy data to service: %+v", err)
			}

		}
	})
}

// forwardRestfulRequest handles request sent from client side,
// and check whether the request should be sent to remote peer or process locally, and send respond back.
// TODO: Process other method and query params, post body, etc.
func (n *Node) forwardRestfulRequest(ctx *fasthttp.RequestCtx, peerId peer.ID, req *models.ApronServiceRequest) {
	if peerId == n.selfID {
		// TODO: function to send request to service
	} else {
		s, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(HttpProxyRequestStream))
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
		log.Printf("Resp: %s\n", string(respBytes))

		// TODO: Build structure to save response, should contains all data, includes status code, headers, etc.
		ctx.Write(respBytes)
	}
}

func (n *Node) forwardWebsocketRequest(ctx *fasthttp.RequestCtx, peerId peer.ID, req *models.ApronServiceRequest) {
	// This is handler for websocket connection from client, should do upgrade things
	upgrader := &websocket.FastHTTPUpgrader{
		ReadBufferSize:  10240,
		WriteBufferSize: 10240,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			return true
		},
	}

	// Connect to remote
	// TODO: Connect remote node with stream, and make remote peer connect to ws service and do conn things.

	err := upgrader.Upgrade(ctx, func(clientWsConn *websocket.Conn) {
		// defer clientWsConn.Close()

		// Create stream to remote peer
		s, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(WsProxyRequestStream))
		internal.CheckError(err)
		// if err != nil {
		// 	ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		// 	return
		// }

		// copy data received from clientWsConn to stream

		// Error channel
		clientToProxyErrorCh := make(chan error, 1)
		proxyToClientErrorCh := make(chan error, 1)

		go ForwardWsMsgToInternalStream(clientWsConn, &s, clientToProxyErrorCh)
		go ForwardInternalStreamToWsMsg(&s, clientWsConn, proxyToClientErrorCh)

		for {
			select {
			case err := <-clientToProxyErrorCh:
				log.Printf("Error while forwarding client data to proxy: %+v", err)
			case err := <-proxyToClientErrorCh:
				log.Printf("Error while forwarding proxy data to client: %+v", err)
			}
		}

		// go func() {
		// 	for {
		// 		log.Printf("ClientSideGateway: Try to read from client")
		// 		_, bytesReceivedFromClient, err := clientWsConn.ReadMessage()
		// 		internal.CheckError(err)
		// 		log.Printf("ClientSideGateway: Receive data from client: %q", bytesReceivedFromClient)
		// 		err = WriteBytesViaStream(s, bytesReceivedFromClient)
		// 		internal.CheckError(err)
		// 	}
		// }()
		//
		// // copy data received from stream to clientWsConn
		// go func() {
		// 	for {
		// 		log.Printf("ClientSideGateway: Try to read from remote node")
		// 		streamBytes, err := ReadBytesViaStream(s)
		// 		internal.CheckError(err)
		// 		log.Printf("ClientSideGateway: Got msg from remote node: %q", streamBytes)
		// 		err = clientWsConn.WriteMessage(websocket.TextMessage, streamBytes)
		// 		internal.CheckError(err)
		// 	}
		// }()

		// Send request detail, which makes remote node to create ws connection to service
		reqBytes, err := proto.Marshal(req)
		internal.CheckError(err)
		log.Printf("ClientSideGateway: request bytes to remote service: %q", reqBytes)
		err = WriteBytesViaStream(s, reqBytes)
		internal.CheckError(err)

		select {}
	})
	internal.CheckError(err)
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
		log.Printf("Read stream: %s\n", str)
	})
}

func (n *Node) StartMgmtApiServer() {
	fasthttp.ListenAndServe(n.Config.MgmtAddr, func(ctx *fasthttp.RequestCtx) {

	})
}

// StartForwardService used to forward service request from client to correct gateway that registered the service.
// The function first parses request sent from client to RequestDetail struct, then build ApronServiceRequest based
// on the request data.
func (n *Node) StartForwardService() {
	fasthttp.ListenAndServe(n.Config.ForwardServiceAddr, func(ctx *fasthttp.RequestCtx) {
		// Parse request URL and split service
		var rawReq bytes.Buffer
		err := models.DumpRequestToBytes(&ctx.Request, &rawReq)
		internal.CheckError(err)

		// log.Printf("Request: %s\n", ctx.Request.String())
		// log.Printf("Request size: %d\n", len(ctx.Request.String()))
		// log.Printf("Request size after processing: %d\n", rawReq.Len())

		serviceNameStr := string(internal.ServiceHostnameToIdByte(ctx.Host()))

		log.Printf("ClientSideGateway: Service name: %s\n", serviceNameStr)
		log.Printf("ClientSideGateway: Current services mapping: %+v\n", n.servicePeerMapping)

		servicePeerId, found := n.servicePeerMapping[serviceNameStr]
		if !found {
			ctx.Error("ClientSideGateway: Service not found", fasthttp.StatusNotFound)
			return
		}

		service, found := n.services[serviceNameStr]
		if !found {
			// Service is in the peer mapping but not in services list, internal error
			ctx.Error("ClientSideGateway: Service data missing, contract help", fasthttp.StatusInternalServerError)
			return
		}

		if len(service.Providers) < 1 {
			ctx.Error("ClientSideGateway: Service data error, contract help", fasthttp.StatusInternalServerError)
			return
		}

		req := &models.ApronServiceRequest{
			ServiceId:  service.GetId(),
			RespStream: "",
			RawRequest: rawReq.Bytes(),
		}

		log.Printf("ClientSideGateway: Service URL requested from : %s\n", ctx.Request.URI())
		if websocket.FastHTTPIsWebSocketUpgrade(ctx) {
			n.forwardWebsocketRequest(ctx, servicePeerId, req)
		} else {
			n.forwardRestfulRequest(ctx, servicePeerId, req)
		}
	})
}
