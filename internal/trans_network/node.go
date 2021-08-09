package trans_network

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"github.com/fasthttp/router"
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
)

type Node struct {
	Host   *host.Host
	Config *NodeConfig

	ps                    *pubsub.PubSub
	broadcastServiceTopic *pubsub.Topic
	serviceBroadcastSub   *pubsub.Subscription
	selfID                peer.ID

	mutex *sync.Mutex
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

	clientWsConns      map[string]*websocket.Conn
	serviceWsConns     map[string]*websocket.Conn
	clientHttpDataChan map[string]chan []byte

	serviceUsageRecordManager models.AggregatedAccessRecordManager
}

func NewNode(ctx context.Context, config *NodeConfig) (*Node, error) {
	h, err := libp2p.New(context.Background(), libp2p.ListenAddrs())
	if err != nil {
		return nil, err
	}

	aggrAccessRecordManager := models.AggregatedAccessRecordManager{}
	aggrAccessRecordManager.Init()

	return &Node{
		Host:                      &h,
		Config:                    config,
		services:                  map[string]models.ApronService{},
		namedServices:             map[string][]models.ApronService{},
		servicePeerMapping:        map[string]peer.ID{},
		mutex:                     &sync.Mutex{},
		requestIdChanMapping:      map[string]chan []byte{},
		clientWsConns:             map[string]*websocket.Conn{},
		serviceWsConns:            map[string]*websocket.Conn{},
		clientHttpDataChan:        map[string]chan []byte{},
		serviceUsageRecordManager: aggrAccessRecordManager,
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
		log.Printf("StartListeningOnServiceBroadcast")
		msg, err := n.serviceBroadcastSub.Next(ctx)
		if err != nil {
			log.Println("wait ServiceBroadcast err", err)
			continue
		}

		log.Printf("ReceivedFrom: %+s\n", msg.ReceivedFrom.Pretty())

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
	log.Printf("Broadcast service: %+v\n", service)
	data, err := proto.Marshal(service)
	if err != nil {
		return err
	}
	return n.broadcastServiceTopic.Publish(ctx, data)
}

func (n *Node) RegisterLocalService(service *models.ApronService) {
	log.Printf("Reg local service id: %s to peer %s\n", service.Id, n.selfID)
	n.mutex.Lock()
	if service.IsDeleted {
		log.Printf("Deleted Local service id: %s\n", service.Id)
		delete(n.services, service.Id)
		delete(n.servicePeerMapping, service.Id)
	} else {
		n.services[service.Id] = *service
		n.servicePeerMapping[service.Id] = n.selfID
	}

	log.Printf("Reg Service count: %+v\n", len(n.services))
	n.mutex.Unlock()
	if err := n.BroadcastService(context.Background(), service); err != nil {
		log.Println("RegisterLocalService err", err)
		panic(err)
	}
}

func (n *Node) RegisterRemoteService(peerId peer.ID, service *models.ApronService) {
	log.Printf("Reg remote service id: %s to peer %s\n", service.Id, peerId.String())
	n.mutex.Lock()

	if service.IsDeleted {
		log.Printf("Deleted Remote Service id: %s\n", service.Id)
		delete(n.services, service.Id)
		delete(n.servicePeerMapping, service.Id)
	} else {
		n.services[service.Id] = *service
		n.servicePeerMapping[service.Id] = peerId
	}

	log.Printf("Reg Service: servicePeerMappings: %+v\n", n.servicePeerMapping)
	log.Printf("Reg Service count: %+v\n", len(n.services))
	n.mutex.Unlock()
}

func (n *Node) NodeAddrStr() string {
	bsIdStr := (*n.Host).ID().Pretty()
	bsNodeAddr := (*n.Host).Addrs()[0]
	return fmt.Sprintf("%s/p2p/%s", bsNodeAddr, bsIdStr)
}

/*******************************/
// Proxy related logic
/*******************************/

func (n *Node) ProxyRequestStreamHandler(s network.Stream) {
	dataCh := make(chan []byte)
	go ReadBytesViaStream(s, dataCh)

	select {
	case proxyReqBytes := <-dataCh:
		proxyReq := &models.ApronServiceRequest{}
		err := proto.Unmarshal(proxyReqBytes, proxyReq)
		internal.CheckError(err)
		log.Printf("Read proxy request from stream: %s\n", proxyReq)

		n.serviceUsageRecordManager.IncUsage(proxyReq.ServiceId, proxyReq.AccountId)

		peerId, err := peer.Decode(proxyReq.PeerId)
		internal.CheckError(err)

		// Get service detail from local services list and fill missing fields of request
		serviceDetail := n.services[proxyReq.ServiceId]
		clientSideReq := proxyReq.BuildHttpRequest(&serviceDetail)
		defer fasthttp.ReleaseRequest(clientSideReq)

		if proxyReq.IsWsRequest {
			respStream, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyWsDataFromServiceSide))
			internal.CheckError(err)

			dialer := websocket.Dialer{
				HandshakeTimeout: 15 * time.Second,
			}

			log.Printf("ServiceSideGateway: client side req URL: %s", clientSideReq.URI().String())
			serviceWsConn, _, err := dialer.Dial(clientSideReq.URI().String(), nil)
			internal.CheckError(err)

			n.serviceWsConns[proxyReq.RequestId] = serviceWsConn

			go func() {
				for {
					_, msgBytes, err := serviceWsConn.ReadMessage()
					internal.CheckError(err)

					log.Printf("ServiceSideGateway: Received message from service: %q\n", msgBytes)

					forwardData := &models.ApronServiceData{
						RequestId: proxyReq.RequestId,
						RawData:   msgBytes,
					}

					forwardDataBytes, err := proto.Marshal(forwardData)
					internal.CheckError(err)
					WriteBytesViaStream(respStream, forwardDataBytes)
				}
			}()
			select {}
		} else {
			respStream, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyHttpRespFromServiceSide))
			internal.CheckError(err)

			serviceSideResp := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(serviceSideResp)

			err = fasthttp.Do(clientSideReq, serviceSideResp)
			internal.CheckError(err)
			respBody := serviceSideResp.Body()

			serviceResp := &models.ApronServiceData{
				RequestId: proxyReq.RequestId,
				RawData:   respBody,
			}

			respBytes, err := proto.Marshal(serviceResp)
			internal.CheckError(err)

			log.Printf("ServiceSideGateway: Write response data: %+q\n", respBody)
			log.Printf("resp stream is nil %+v\n", respStream == nil)
			WriteBytesViaStream(respStream, respBytes)
		}
	}
}

func (n *Node) ProxyWsDataHandler(s network.Stream) {
	dataCh := make(chan []byte)
	go ReadBytesViaStream(s, dataCh)

	for {
		select {
		case proxyDataBytes := <-dataCh:
			proxyData := &models.ApronServiceData{}
			err := proto.Unmarshal(proxyDataBytes, proxyData)
			internal.CheckError(err)

			log.Printf("ProxyWsDataHandler: Read proxy data from stream: %+v, %s\n", s.Protocol(), proxyData)

			if s.Protocol() == protocol.ID(ProxyWsDataFromClientSide) {
				log.Printf("ProxyDataFromClientSideHandler: Send data to service\n")
				err = n.serviceWsConns[proxyData.RequestId].WriteMessage(websocket.TextMessage, proxyData.RawData)
				internal.CheckError(err)
			} else if s.Protocol() == protocol.ID(ProxyWsDataFromServiceSide) {
				log.Printf("ProxyDataFromServiceHandler: Send data to client\n")
				err = n.clientWsConns[proxyData.RequestId].WriteMessage(websocket.TextMessage, proxyData.RawData)
				internal.CheckError(err)
			} else {
				panic(errors.New(fmt.Sprintf("wrong protocol: %s", s.Protocol())))
			}
		}
	}
}

func (n *Node) ProxyHttpRespHandler(s network.Stream) {
	dataCh := make(chan []byte)
	go ReadBytesViaStream(s, dataCh)

	for {
		select {
		case proxyDataBytes := <-dataCh:
			proxyData := &models.ApronServiceData{}
			err := proto.Unmarshal(proxyDataBytes, proxyData)
			internal.CheckError(err)

			log.Printf("ProxyHttpRespHandler: Read proxy data from stream: %+v, %s\n", s.Protocol(), proxyData)

			n.clientHttpDataChan[proxyData.RequestId] <- proxyData.RawData
		}

	}

}

func (n *Node) SetProxyStreamHandlers() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyReqStream), n.ProxyRequestStreamHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyWsDataFromClientSide), n.ProxyWsDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyWsDataFromServiceSide), n.ProxyWsDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyHttpRespFromServiceSide), n.ProxyHttpRespHandler)
}

// TODO: Support LB.

// serveWebsocketRequest servers websocket request sent from client, and process data received from service
func (n *Node) serveWebsocketRequest(ctx *fasthttp.RequestCtx, peerId peer.ID, req *models.ApronServiceRequest) {
	// This is handler for websocket connection from client, should do upgrade things
	upgrader := &websocket.FastHTTPUpgrader{
		ReadBufferSize:  10240,
		WriteBufferSize: 10240,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			return true
		},
	}

	dataStream, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyWsDataFromClientSide))

	err = upgrader.Upgrade(ctx, func(clientWsConn *websocket.Conn) {
		n.clientWsConns[req.RequestId] = clientWsConn

		// Forward client data to stream, the data will be packet to ApronServiceData struct
		// TODO: Handle error
		go func() {
			for {
				_, msgBytes, err := clientWsConn.ReadMessage()
				internal.CheckError(err)

				log.Printf("ClientSideGateway: dataStream conn: %+v\n", dataStream.Conn())

				log.Printf("ClientSideGateway: Received message from client: %q\n", msgBytes)

				forwardData := &models.ApronServiceData{
					RequestId: req.RequestId,
					RawData:   msgBytes,
				}

				forwardDataBytes, err := proto.Marshal(forwardData)
				internal.CheckError(err)
				WriteBytesViaStream(dataStream, forwardDataBytes)
				log.Println("ClientSideGateway: data written to stream, wait for next client ws msg")
			}
		}()
		select {
		// TODO: Add error handler
		}
	})
	internal.CheckError(err)
}

func (n *Node) serveHttpRequest(ctx *fasthttp.RequestCtx, streamToServiceGW network.Stream, req *models.ApronServiceRequest) {
	n.clientHttpDataChan[req.RequestId] = make(chan []byte)
	select {
	case respData := <-n.clientHttpDataChan[req.RequestId]:
		_, err := ctx.Write(respData)
		internal.CheckError(err)

		delete(n.clientHttpDataChan, req.RequestId)
	}
}

// UpdatePeers : if a peer disconnected and wasn't found in topic, all services related to it will be removed.
func (n *Node) UpdatePeers() {
	peerRefreshTicker := time.NewTicker(time.Second)
	defer peerRefreshTicker.Stop()
	log.Printf("UpdatePeers")
	for {
		<-peerRefreshTicker.C
		availablePeers := n.ps.ListPeers(BroadcastServiceChannel)
		log.Printf("availablePeers %++v", availablePeers)
		invaildService := make([]string, 0)
		n.mutex.Lock()
		for k, v := range n.servicePeerMapping {
			if v == n.selfID {
				continue
			}
			found := false
			for _, p := range availablePeers {
				if v == p {
					found = true
				}
			}

			if !found {
				invaildService = append(invaildService, k)
			}

		}

		// remove related services
		for _, service := range invaildService {
			log.Printf("deleted")
			delete(n.services, service)
			delete(n.servicePeerMapping, service)
		}
		n.mutex.Unlock()

	}

}

func (n *Node) StartMgmtApiServer() {
	// Init routers
	router := router.New()

	// Service related
	serviceRouter := router.Group("/service")
	serviceRouter.GET("/", n.listServiceHandler)
	serviceRouter.POST("/", n.newOrUpdateServiceHandler)
	serviceRouter.DELETE("/", n.deleteServiceHandler)
	serviceRouter.GET("/local", n.listLocalServiceHandler)
	serviceRouter.GET("/remote", n.listRemoteServiceHandler)
	serviceRouter.GET("/report", n.allUsageReportHandler)

	log.Printf("Management API Server: %s\n", n.Config.MgmtAddr)
	fasthttp.ListenAndServe(n.Config.MgmtAddr, router.Handler)
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

		serviceNameStr := string(internal.ServiceHostnameToIdByte(ctx.Host()))

		log.Printf("ClientSideGateway: Service name: %s\n", serviceNameStr)
		log.Printf("ClientSideGateway: Current services mapping: %+v\n", n.servicePeerMapping)

		n.mutex.Lock()
		servicePeerId, found := n.servicePeerMapping[serviceNameStr]
		if !found {
			n.mutex.Unlock()
			ctx.Error("ClientSideGateway: Service not found", fasthttp.StatusNotFound)
			return
		}

		service, found := n.services[serviceNameStr]
		if !found {
			n.mutex.Unlock()
			// Service is in the peer mapping but not in services list, internal error
			ctx.Error("ClientSideGateway: Service data missing, contract help", fasthttp.StatusInternalServerError)
			return
		}
		n.mutex.Unlock()

		if len(service.Providers) < 1 {
			ctx.Error("ClientSideGateway: Service data error, contract help", fasthttp.StatusInternalServerError)
			return
		}

		reqDetail, err := models.ExtractRequestDetailFromFasthttpRequest(&ctx.Request)

		requestId := uuid.New().String()

		req := &models.ApronServiceRequest{
			ServiceId:   service.GetId(),
			RequestId:   requestId,
			PeerId:      (*n.Host).ID().String(),
			IsWsRequest: websocket.FastHTTPIsWebSocketUpgrade(ctx),
			AccountId: string(reqDetail.UserKey),
			RawRequest:  rawReq.Bytes(),
		}

		// Register the requestId to current node
		msgCh := make(chan []byte)
		n.requestIdChanMapping[requestId] = msgCh

		log.Printf("ClientSideGateway: Service URL requested from : %s\n", ctx.Request.URI())
		log.Printf("ClientSideGateway: servicePeerId : %s\n", servicePeerId.String())
		s, err := (*n.Host).NewStream(context.Background(), servicePeerId, protocol.ID(ProxyReqStream))
		if err != nil {
			log.Printf("forward service request err: %+v\n", err)
			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
			return
		}

		// Build request package with passed in request
		reqBytes, err := proto.Marshal(req)
		internal.CheckError(err)
		WriteBytesViaStream(s, reqBytes)

		if websocket.FastHTTPIsWebSocketUpgrade(ctx) {
			n.serveWebsocketRequest(ctx, servicePeerId, req)
		} else {
			n.serveHttpRequest(ctx, s, req)
		}
	})
}
