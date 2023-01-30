package trans_network

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"github.com/fasthttp/websocket"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (n *Node) ProxyHttpInitRequestHandler(s network.Stream) {
	dataCh := make(chan []byte)
	go ReadBytesViaStream(s, dataCh)

	select {
	case proxyReqBytes := <-dataCh:
		proxyReq := &models.ApronServiceRequest{}
		err := proto.Unmarshal(proxyReqBytes, proxyReq)
		internal.CheckError(err)
		n.logger.Sugar().Infof("Read proxy request from stream: %s\n", proxyReq)

		httpReq, err := proxyReq.RecoverClientRequest()
		internal.CheckError(err)

		var clientReqDetail models.RequestDetail
		err = models.ExtractRequestDetailFromFasthttpRequest(httpReq, &clientReqDetail)

		n.serviceUsageRecordManager.RecordUsageFromInitHttpProxyRequest(proxyReq, &clientReqDetail)

		peerId, err := peer.Decode(proxyReq.PeerId)
		internal.CheckError(err)

		// Get service detail from local services list and fill missing fields of request
		serviceDetail := n.services[proxyReq.ServiceId]
		n.logger.Sugar().Infof("Service detail: %#v\n", serviceDetail)
		reqToService, err := proxyReq.BuildHttpRequestToService(&clientReqDetail, httpReq, &serviceDetail)
		internal.CheckError(err)
		n.logger.Sugar().Infof("Request to service: %#v\n", reqToService)
		defer fasthttp.ReleaseRequest(reqToService)

		if proxyReq.IsWsRequest {
			respStream, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyWsDataFromServiceSide))
			internal.CheckError(err)

			dialer := websocket.Dialer{
				HandshakeTimeout: 15 * time.Second,
			}

			n.logger.Sugar().Infof("ServiceSideGateway: client side req URL: %s", reqToService.URI().String())
			serviceWsConn, _, err := dialer.Dial(reqToService.URI().String(), nil)
			internal.CheckError(err)

			n.serviceWsConns[proxyReq.RequestId] = serviceWsConn

			go func() {
				for {
					_, msgBytes, err := serviceWsConn.ReadMessage()
					internal.CheckError(err)

					n.logger.Sugar().Infof("ServiceSideGateway: Received message from service: %q\n", msgBytes)

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

			err = fasthttp.Do(reqToService, serviceSideResp)
			internal.CheckError(err)
			respBody := serviceSideResp.Body()

			serviceResp := &models.ApronServiceData{
				RequestId: proxyReq.RequestId,
				RawData:   respBody,
			}

			respBytes, err := proto.Marshal(serviceResp)
			internal.CheckError(err)

			n.logger.Sugar().Infof("ServiceSideGateway: Write response data: %+q\n", respBody)
			n.logger.Sugar().Infof("resp stream is nil %+v\n", respStream == nil)
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

			n.logger.Sugar().Infof("ProxyWsDataHandler: Read proxy data from stream: %+v, %s\n", s.Protocol(), proxyData)

			if s.Protocol() == protocol.ID(ProxyWsDataFromClientSide) {
				n.logger.Sugar().Infof("ProxyDataFromClientSideHandler: Send data to service\n")
				err = n.serviceWsConns[proxyData.RequestId].WriteMessage(websocket.TextMessage, proxyData.RawData)
				internal.CheckError(err)
				n.serviceUsageRecordManager.RecordUsageHttpProxyData(proxyData, true)
			} else if s.Protocol() == protocol.ID(ProxyWsDataFromServiceSide) {
				n.logger.Sugar().Infof("ProxyDataFromServiceHandler: Send data to client\n")
				err = n.clientWsConns[proxyData.RequestId].WriteMessage(websocket.TextMessage, proxyData.RawData)
				internal.CheckError(err)
				n.serviceUsageRecordManager.RecordUsageHttpProxyData(proxyData, false)
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

			n.logger.Sugar().Infof("ProxyHttpRespHandler: Read proxy data from stream: %+v, %s\n", s.Protocol(), proxyData)
			n.serviceUsageRecordManager.RecordUsageHttpProxyData(proxyData, false)

			n.clientHttpDataChan[proxyData.RequestId] <- proxyData.RawData
		}

	}

}

// ProxySocketInitReqHandler will be used to init socket connection to service
func (n *Node) ProxySocketInitReqHandler(s network.Stream) {
	// Use sync read here since the connection to service should be created before processing proxy data
	dataBuf, err := ReadOneFrameDataFromStream(s)
	internal.CheckError(err)

	socketInitReq := &models.ApronSocketInitRequest{}
	err = proto.Unmarshal(dataBuf, socketInitReq)
	internal.CheckError(err)
	n.logger.Sugar().Infof("Read init socket request from stream: %s\n", socketInitReq)

	csgwPeerId, err := peer.Decode(socketInitReq.PeerId)
	internal.CheckError(err)

	n.logger.Sugar().Infof("ClientSideGateway PeerID: %+v\n", csgwPeerId)

	n.serviceUsageRecordManager.RecordUsageFromSocket(socketInitReq)

	// Get service detail from local services list and fill missing fields of request
	serviceDetail := n.services[socketInitReq.ServiceId]
	n.logger.Info("service detail saved", zap.Any("service_detail", serviceDetail))

	socketServiceUrl := serviceDetail.Providers[0].GetBaseUrl()
	n.logger.Info("connecting to socket service URL",
		zap.String("socket_service_url", socketServiceUrl),
		zap.Any("service_socket_conn", n.serviceSocketConns),
	)

	serviceSocketConn, err := net.Dial("tcp", socketServiceUrl)
	internal.CheckError(err)

	n.serviceSocketConns[socketInitReq.RequestId] = serviceSocketConn
	n.logger.Info("socket service connection established", zap.String("request_id", socketInitReq.RequestId))

	// create stream with CSGW for response data
	respStream, err := (*n.Host).NewStream(context.Background(), csgwPeerId, protocol.ID(ProxySocketDataFromServiceSide))

	// Create reader goroutines for socket service, and forward service data back with request id in respStream
	go func() {
		serverReader := bufio.NewReader(serviceSocketConn)
		buf := make([]byte, 4096)
		for {
			readSize, err := serverReader.Read(buf)
			internal.CheckError(err)

			n.logger.Sugar().Infof("ServiceSideGateway: Received message from service: %q", buf[:readSize])

			forwardData := &models.ApronServiceData{
				RequestId: socketInitReq.RequestId,
				RawData:   buf[:readSize],
			}

			forwardDataBytes, err := proto.Marshal(forwardData)
			internal.CheckError(err)
			WriteBytesViaStream(respStream, forwardDataBytes)
		}
	}()
	select {}
}

// ProxySocketDataHandler will be used to process socket data from client or service side
func (n *Node) ProxySocketDataHandler(s network.Stream) {
	dataCh := make(chan []byte)
	go ReadBytesViaStream(s, dataCh)

	for {
		select {
		case proxyDataBytes := <-dataCh:
			proxyData := &models.ApronServiceData{}
			err := proto.Unmarshal(proxyDataBytes, proxyData)
			internal.CheckError(err)

			//n.logger.Sugar().Infof("ProxySocketDataHandler: Read proxy data from stream: %+v, %s\n", s.Protocol(), proxyData)

			if s.Protocol() == protocol.ID(ProxySocketDataFromClientSide) {
				// Got data sent from CSGW. The data is ApronServiceData package,
				// and the RawData should be ExtServiceData, which will be processed in agent side
				n.logger.Info("ProxyDataFromClientSideHandler: got data from CSGW", zap.String("request_id", proxyData.RequestId))

				WriteBytesViaStream(n.serviceSocketConns[proxyData.RequestId], proxyData.RawData)
				n.logger.Info("data sent to service agent / service", zap.String("request_id", proxyData.RequestId))

				// TODO: Record socket usage data is not implemented yet
				//n.serviceUsageRecordManager.RecordUsageFromSocket(proxyData, true)
			} else if s.Protocol() == protocol.ID(ProxySocketDataFromServiceSide) {
				n.logger.Info("ProxyDataFromServiceHandler: Send data to client")
				_, err = n.clientSocketConns[proxyData.RequestId].Write(proxyData.RawData)
				internal.CheckError(err)

				// TODO: Record socket usage data is not implemented yet
				//n.serviceUsageRecordManager.RecordUsageHttpProxyData(proxyData, false)
			} else {
				panic(errors.New(fmt.Sprintf("wrong protocol: %s", s.Protocol())))
			}
		}
	}
}

func (n *Node) SetProxyStreamHandlers() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyHttpInitReq), n.ProxyHttpInitRequestHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyWsDataFromClientSide), n.ProxyWsDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyWsDataFromServiceSide), n.ProxyWsDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyHttpRespFromServiceSide), n.ProxyHttpRespHandler)

	// Socket related handlers
	(*n.Host).SetStreamHandler(protocol.ID(ProxySocketInitReq), n.ProxySocketInitReqHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxySocketDataFromClientSide), n.ProxySocketDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxySocketDataFromServiceSide), n.ProxySocketDataHandler)
}
