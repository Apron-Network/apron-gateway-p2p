package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/ipfs_agent"
	"apron.network/gateway-p2p/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
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

// StartForwardService used to forward service request from client to correct gateway that registered the service.
// The function first parses request sent from client to RequestDetail struct, then build ApronServiceRequest based
// on the request data.
func (n *Node) StartForwardService() {
	fasthttp.ListenAndServe(n.Config.ForwardServiceAddr, func(ctx *fasthttp.RequestCtx) {
		// Parse request URL and split service
		var rawReq bytes.Buffer
		err := models.DumpRequestToBytes(&ctx.Request, &rawReq)
		internal.CheckError(err)

		// TODO: ExtractRequestDetailFromFasthttpRequest contains extra logic which is not required here,
		// change to some functions that only extract needed info
		var clientReqDetail models.RequestDetail
		err = models.ExtractRequestDetailFromFasthttpRequest(&ctx.Request, &clientReqDetail)
		if err != nil {
			ctx.Error(fmt.Sprintf("ClientSideGateway: extract service name error: %+v", err), fasthttp.StatusInternalServerError)
		}
		serviceNameStr := string(clientReqDetail.ServiceName)

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

		requestId := uuid.New().String()

		req := &models.ApronServiceRequest{
			ServiceId:   service.GetId(),
			RequestId:   requestId,
			PeerId:      (*n.Host).ID().String(),
			IsWsRequest: websocket.FastHTTPIsWebSocketUpgrade(ctx),
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

func (n *Node) StartUploadUsageReportTask(uploadInterval int, ipfsAgent ipfs_agent.IpfsAgent) {
	for true {
		if rslt, err := n.serviceUsageRecordManager.ExportAllUsage(); err != nil {
			fmt.Printf(fmt.Errorf("error occurred while exporting usage report: %+v", err).Error())
		} else {
			tmpReportFile, err := os.CreateTemp("", "sample")
			defer os.Remove(tmpReportFile.Name())
			if err != nil {
				fmt.Printf(fmt.Errorf("error occurred while creating usage report tmp file: %+v", err).Error())
			} else {
				usageRecordsJsonByte, _ := json.Marshal(rslt)
				_, err := tmpReportFile.Write(usageRecordsJsonByte)
				if err != nil {
					fmt.Printf(fmt.Errorf("error occurred while writing usage report data: %+v", err).Error())
				} else {
					ipfsAgent.PinFile(tmpReportFile.Name())
				}
			}
		}
		time.Sleep(time.Duration(uploadInterval) * time.Second)
	}

}
