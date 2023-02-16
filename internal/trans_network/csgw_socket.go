package trans_network

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// StartSocketForwardService used to forward socket data from client to correct gateway.
// Forwarding socket data requires sending auth message first, which contains auth message and connection detail,
func (n *Node) StartSocketForwardService() {
	n.logger.Sugar().Infof("Socket Forward Server: %s", n.Config.SocketForwardServiceAddr)
	listen, err := net.Listen("tcp", n.Config.SocketForwardServiceAddr)
	internal.CheckError(err)
	for {
		conn, err := listen.Accept()

		go func(conn net.Conn) {
			internal.CheckError(err)
			defer conn.Close()

			r := bufio.NewReader(conn)

			var msgLen uint64
			err = binary.Read(r, binary.BigEndian, &msgLen)
			internal.CheckError(err)

			// Read init request to get service detail
			initRequest := models.ApronSocketInitRequest{}
			n.logger.Sugar().Infof("Request size: %d", msgLen)
			initRequestBytes := make([]byte, msgLen)

			// Parse init request
			readSize, err := r.Read(initRequestBytes)
			internal.CheckError(err)
			n.logger.Sugar().Infof("Read size: %d", readSize)

			err = proto.Unmarshal(initRequestBytes, &initRequest)
			internal.CheckError(err)

			n.logger.Info("parsed apron socket init request", zap.Any("init_request", initRequest))

			n.mutex.Lock()
			servicePeerId, found := n.servicePeerMapping[initRequest.ServiceId]
			if !found {
				n.mutex.Unlock()
				n.logger.Error("CSGW: Service not found",
					zap.Any("init_request", initRequest),
					zap.Any("existing_services", n.servicePeerMapping),
				)
				conn.Write([]byte("ClientSideGateway: Service not found"))
				return
			}

			service, found := n.services[initRequest.ServiceId]
			if !found {
				n.mutex.Unlock()
				n.logger.Error("CSGW: Service data is missing", zap.Any("init_request", initRequest))
				// Service is in the peer mapping but not in services list, internal error
				conn.Write([]byte("ClientSideGateway: Service data missing, contract help"))
				return
			}
			n.mutex.Unlock()

			if len(service.Providers) < 1 {
				n.logger.Error("CSGW: Service data error", zap.Any("init_request", initRequest))
				conn.Write([]byte("ClientSideGateway: Service data error, contract help"))
				return
			}

			n.logger.Sugar().Infof("Service detail: %#v", service)

			// Find service connection detail, build data package and send to correct gateway

			// Build request ID
			requestId := fmt.Sprintf("%s.%s.%s", initRequest.UserId, service.GetId(), uuid.New().String())

			// Create ApronServiceRequest struct and send to remote with stream
			req := &models.ApronSocketServiceRequest{
				ServiceId: service.GetId(),
				RequestId: requestId,
				PeerId:    (*n.Host).ID().String(),
			}

			// Register the requestId to current node
			msgCh := make(chan []byte)
			n.requestIdChanMapping[requestId] = msgCh

			initSocketConnStream, err := (*n.Host).NewStream(context.Background(), servicePeerId, protocol.ID(ProxySocketInitReq))
			if err != nil {
				n.logger.Sugar().Errorf("forward service init request err: %+v\n", err)
				return
			}

			reqBytes, err := proto.Marshal(req)
			internal.CheckError(err)
			n.logger.Info("Init socket request", zap.Int("request_size", len(reqBytes)), zap.ByteString("req_bytes", reqBytes))
			WriteBytesViaStream(initSocketConnStream, reqBytes)

			clientSocketDataStream, err := (*n.Host).NewStream(context.Background(), servicePeerId, protocol.ID(ProxySocketDataFromClientSide))
			if err != nil {
				n.logger.Sugar().Errorf("forward service data request err: %+v\n", err)
				return
			}

			// Save client to CSGW connection in node, which will be used in stream handler function
			n.clientSocketConns[requestId] = conn

			go func() {
				for {
					reader := bufio.NewReader(conn)
					data, err := ReadOneFrameDataFromStream(reader)
					if err != nil {
						n.logger.Error("receive data from client error",
							zap.String(EntityFieldName, EntityCSGW),
							zap.Error(err),
						)
						if err.Error() == "EOF" {
							conn.Close()
							break
						}
						continue
					}

					n.logger.Info("CSGW: Received data from client",
						zap.Int("data_size", len(data)),
						zap.String(EntityFieldName, EntityCSGW),
					)
					serviceData := models.ApronServiceData{
						RequestId: requestId,
						RawData:   data,
					}

					serviceDataBytes, err := proto.Marshal(&serviceData)
					internal.CheckError(err)
					n.logger.Info("package data into ApronServiceData and send to SSGW",
						zap.String(EntityFieldName, EntityCSGW),
						zap.Int("data_size", len(serviceDataBytes)),
						zap.String("stream_name", "ProxySocketDataFromClientSide"),
					)
					WriteBytesViaStream(clientSocketDataStream, serviceDataBytes)
				}

			}()

			// Infinity loop to keep the connection
			select {}
		}(conn)
	}
}
