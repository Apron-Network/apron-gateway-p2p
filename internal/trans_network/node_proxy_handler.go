package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"context"
	"errors"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
)

func (n *Node) ProxyRequestStreamHandler(s network.Stream) {
	dataCh := make(chan []byte)
	go ReadBytesViaStream(s, dataCh)

	select {
	case proxyReqBytes := <-dataCh:
		proxyReq := &models.ApronServiceRequest{}
		err := proto.Unmarshal(proxyReqBytes, proxyReq)
		internal.CheckError(err)
		log.Printf("Read proxy request from stream: %s\n", proxyReq)

		httpReq, err := proxyReq.RecoverClientRequest()
		internal.CheckError(err)

		var clientReqDetail models.RequestDetail
		err = models.ExtractRequestDetailFromFasthttpRequest(httpReq, &clientReqDetail)

		n.serviceUsageRecordManager.RecordUsageFromProxyRequest(proxyReq, &clientReqDetail)

		peerId, err := peer.Decode(proxyReq.PeerId)
		internal.CheckError(err)

		// Get service detail from local services list and fill missing fields of request
		serviceDetail := n.services[proxyReq.ServiceId]
		log.Printf("Service detail: %#v\n", serviceDetail)
		reqToService := proxyReq.BuildHttpRequestToService(&clientReqDetail, httpReq, &serviceDetail)
		log.Printf("Request to service: %#v\n", reqToService)
		defer fasthttp.ReleaseRequest(reqToService)

		if proxyReq.IsWsRequest {
			respStream, err := (*n.Host).NewStream(context.Background(), peerId, protocol.ID(ProxyWsDataFromServiceSide))
			internal.CheckError(err)

			dialer := websocket.Dialer{
				HandshakeTimeout: 15 * time.Second,
			}

			log.Printf("ServiceSideGateway: client side req URL: %s", reqToService.URI().String())
			serviceWsConn, _, err := dialer.Dial(reqToService.URI().String(), nil)
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

			err = fasthttp.Do(reqToService, serviceSideResp)
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
				n.serviceUsageRecordManager.RecordUsageFromProxyData(proxyData, true)
			} else if s.Protocol() == protocol.ID(ProxyWsDataFromServiceSide) {
				log.Printf("ProxyDataFromServiceHandler: Send data to client\n")
				err = n.clientWsConns[proxyData.RequestId].WriteMessage(websocket.TextMessage, proxyData.RawData)
				internal.CheckError(err)
				n.serviceUsageRecordManager.RecordUsageFromProxyData(proxyData, false)
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
			n.serviceUsageRecordManager.RecordUsageFromProxyData(proxyData, false)

			n.clientHttpDataChan[proxyData.RequestId] <- proxyData.RawData
		}

	}

}

// ProxySocketInitReqHandler will be used to init socket connection to service
func (n *Node) ProxySocketInitReqHandler(s network.Stream) {

}

// ProxySocketDataHandler will be used to process socket data from client or service side
func (n *Node) ProxySocketDataHandler(s network.Stream) {

}

func (n *Node) SetProxyStreamHandlers() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyReqStream), n.ProxyRequestStreamHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyWsDataFromClientSide), n.ProxyWsDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyWsDataFromServiceSide), n.ProxyWsDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxyHttpRespFromServiceSide), n.ProxyHttpRespHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxySocketInitReq), n.ProxySocketInitReqHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxySocketDataFromClientSide), n.ProxySocketDataHandler)
	(*n.Host).SetStreamHandler(protocol.ID(ProxySocketDataFromServiceSide), n.ProxySocketDataHandler)
}
