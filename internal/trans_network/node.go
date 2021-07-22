package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
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

	// Mapping of request id and client ctx. After receiving request from client side,
	// the client side gateway add uniq requestID to forwarded ApronServiceRequest,
	// and the streamID and the ctx of client will be saved here for later usage
	requestIdChanMapping map[string]chan []byte
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
		Host:                 &h,
		Config:               config,
		services:             map[string]models.ApronService{},
		namedServices:        map[string][]models.ApronService{},
		servicePeerMapping:   map[string]peer.ID{},
		requestIdChanMapping: map[string]chan []byte{},
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
		if err != nil {
			log.Println("wait ServiceBroadcast err", err)
			continue
		}

		if msg.ReceivedFrom == n.selfID {
			continue
		}
		service := new(models.ApronService)
		err = proto.Unmarshal(msg.Data, service)
		if err != nil {
			log.Println("Unmarshal ApronService err", err)
			continue
		}
		log.Printf("Received service: %+v\n", service)
		n.RegisterRemoteService(msg.ReceivedFrom, service)
	}
}

// BroadcastService broad local service to the network with configured topic,
// so all nodes subscribed to the topic can update its local cache data
func (n *Node) BroadcastService(ctx context.Context, service *models.ApronService) error {
	data, err := proto.Marshal(service)
	if err != nil {
		return err
	}
	return n.broadcastServiceTopic.Publish(ctx, data)
}

func (n *Node) RegisterLocalService(service *models.ApronService) {
	log.Printf("Reg local service id: %s to peer %s\n", service.Id, n.selfID)
	n.services[service.Id] = *service
	n.servicePeerMapping[service.Id] = n.selfID
	if err := n.BroadcastService(context.Background(), service); err != nil {
		log.Println("RegisterLocalService err", err)
		panic(err)
	}
	log.Printf("Reg Service count: %+v\n", len(n.services))
}

func (n *Node) RegisterRemoteService(peerId peer.ID, service *models.ApronService) {
	log.Printf("Reg remote service id: %s to peer %s\n", service.Id, peerId.String())
	n.services[service.Id] = *service
	n.servicePeerMapping[service.Id] = peerId
	log.Printf("Reg Service: servicePeerMappings: %+v\n", n.servicePeerMapping)
	log.Printf("Reg Service count: %+v\n", len(n.services))
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

func (n *Node) ProxyRequestStreamHandler(s network.Stream) {
	proxyReq, err := ParseProxyReqFromStream(s)
	internal.CheckError(err)
	log.Printf("Read proxy request from stream: %s\n", proxyReq)

	peerId, err := peer.Decode(proxyReq.PeerId)
	internal.CheckError(err)

	respStream, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyRespStream))
	internal.CheckError(err)

	// Get service detail from local services list and fill missing fields of request
	serviceDetail := n.services[proxyReq.ServiceId]
	clientSideReq := proxyReq.BuildHttpRequest(serviceDetail)
	defer fasthttp.ReleaseRequest(clientSideReq)

	if bytes.HasPrefix(proxyReq.ClientSchema, []byte("ws")) {
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

		for {
			select {
			case err := <-serviceToProxyErrorCh:
				log.Printf("Error while forwarding service data to proxy: %+v", err)
			case err := <-proxyToServiceErrorCh:
				log.Printf("Error while forwarding proxy data to service: %+v", err)
			}

		}
	} else if bytes.HasPrefix(proxyReq.ClientSchema, []byte("http")) {
		serviceSideResp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(serviceSideResp)

		// log.Printf("Client side req: %+v", clientSideReq)
		// log.Printf("Service detail in remote node: %+v", serviceDetail)

		err = fasthttp.Do(clientSideReq, serviceSideResp)
		internal.CheckError(err)
		respBody := serviceSideResp.Body()

		// TODO: Replace with send to Resp stream
		serviceResp := &models.ApronServiceResponse{
			RequestId:   proxyReq.RequestId,
			RawResponse: respBody,
		}

		respBytes, err := proto.Marshal(serviceResp)
		internal.CheckError(err)

		log.Printf("ServiceSideGateway: Write response data: %+q\n", respBody)
		log.Printf("resp stream is nil %+v\n", respStream == nil)
		err = WriteBytesViaStream(respStream, respBytes)
		internal.CheckError(err)
	} else {
		panic(errors.New("wrong schema requested"))
	}
}

func (n *Node) ProxyResponseStreamHandler(s network.Stream) {
	proxyReq, err := ParseProxyRespFromStream(s)
	internal.CheckError(err)
	log.Printf("ProxyResponseStreamHandler: Read proxy resp from stream: %s\n", proxyReq)
	n.requestIdChanMapping[proxyReq.RequestId] <- proxyReq.RawResponse
}

func (n *Node) SetProxyStreamHandlers() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyReqStream), n.ProxyRequestStreamHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyRespStream), n.ProxyResponseStreamHandler)
}

// SetProxyRequestStreamHandler set handler to process proxy request
// TODO: This handler process ApronServiceRequest sent from client side node
// TODO: then the node try to find the service detail in local services list, then make the request
// TODO: So the node near client should only pass service name, account id and request params,
// TODO: and this handler node try to find service detail and build request.
// TODO: If there are any LB things, doing it here.

// forwardRestfulRequest handles request sent from client side,
// and check whether the request should be sent to remote peer or process locally, and send respond back.
// TODO: Process other method and query params, post body, etc.
// func (n *Node) forwardRestfulRequest(ctx *fasthttp.RequestCtx, peerId peer.ID, req *models.ApronServiceRequest) {
// 	if peerId == n.selfID {
// 		// TODO: function to send request to service
// 	} else {
// 		s, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(HttpProxyRequestStream))
// 		if err != nil {
// 			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
// 			return
// 		}
//
// 		// Build request package with passed in request
// 		reqBytes, err := proto.Marshal(req)
// 		internal.CheckError(err)
// 		err = WriteBytesViaStream(s, reqBytes)
// 		internal.CheckError(err)
//
// 		// Read response from remote
// 		respBytes, err := ReadBytesViaStream(s)
// 		internal.CheckError(err)
//
// 		if err != nil {
// 			panic(err)
// 		}
// 		log.Printf("Resp: %s\n", string(respBytes))
//
// 		// TODO: Build structure to save response, should contains all data, includes status code, headers, etc.
// 		ctx.Write(respBytes)
// 	}
// }

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

func (n *Node) StartMgmtApiServer() {
	fasthttp.ListenAndServe(n.Config.MgmtAddr, func(ctx *fasthttp.RequestCtx) {
		// TODO: Manage service:
		// TODO:   - Add local services:
		// TODO:	 1. Invoke RegisterLocalService to add service to local service list
		// TODO:	 2. Publish service changes to all network via pubsub in BroadcastServiceChannel
		// TODO:	 1. All other nodes received the pubsub message and invoke RegisterRemoteService function to build service id and peer id mapping
		// TODO:   - vice versa for removing local services
		// TODO: Question: How to handle the gap between service changed in local but not changed in remote
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

		requestId := uuid.New().String()

		req := &models.ApronServiceRequest{
			ServiceId:    service.GetId(),
			RequestId:    requestId,
			PeerId:       (*n.Host).ID().String(),
			ClientSchema: ctx.URI().Scheme(),
			RawRequest:   rawReq.Bytes(),
		}

		// Register the requestId to current node
		msgCh := make(chan []byte)
		n.requestIdChanMapping[requestId] = msgCh

		log.Printf("ClientSideGateway: Service URL requested from : %s\n", ctx.Request.URI())
		log.Printf("ClientSideGateway: servicePeedId : %s\n", servicePeerId.String())
		s, err := (*n.Host).NewStream(context.Background(), servicePeerId, protocol.ID(ProxyReqStream))
		if err != nil {
			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
			return
		}

		// Build request package with passed in request
		reqBytes, err := proto.Marshal(req)
		internal.CheckError(err)
		err = WriteBytesViaStream(s, reqBytes)
		internal.CheckError(err)

		if websocket.FastHTTPIsWebSocketUpgrade(ctx) {
			// TODO
			// Request sent from client is websocket request, upgrade the connection and prepare to forward data
			n.forwardWebsocketRequest(ctx, servicePeerId, req)
		}

		select {
		case msg := <-msgCh:
			ctx.Write(msg)
		}
	})
}
