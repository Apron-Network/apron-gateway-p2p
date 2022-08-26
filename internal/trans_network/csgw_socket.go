package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/protocol"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
)

// StartSocketForwardService used to forward socket data from client to correct gateway.
// Forwarding socket data requires sending auth message first, which contains auth message and connection detail,
func (n *Node) StartSocketForwardService() {
	log.Printf("Socket Forward Server: %s\n", n.Config.SocketForwardServiceAddr)
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
			log.Printf("Request size: %d\n", msgLen)
			initRequestBytes := make([]byte, msgLen)

			// Parse init request
			readSize, err := r.Read(initRequestBytes)
			internal.CheckError(err)
			log.Printf("Read size: %d\n", readSize)

			err = proto.Unmarshal(initRequestBytes, &initRequest)
			internal.CheckError(err)

			n.mutex.Lock()
			servicePeerId, found := n.servicePeerMapping[initRequest.ServiceId]
			if !found {
				n.mutex.Unlock()
				conn.Write([]byte("ClientSideGateway: Service not found"))
				return
			}

			service, found := n.services[initRequest.ServiceId]
			if !found {
				n.mutex.Unlock()
				// Service is in the peer mapping but not in services list, internal error
				conn.Write([]byte("ClientSideGateway: Service data missing, contract help"))
				return
			}
			n.mutex.Unlock()

			if len(service.Providers) < 1 {
				conn.Write([]byte("ClientSideGateway: Service data error, contract help"))
				return
			}

			log.Printf("Service detail: %#v\n", service)

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
				log.Printf("forward service init request err: %+v\n", err)
				return
			}

			// Build request package with passed in request
			reqBytes, err := proto.Marshal(req)
			internal.CheckError(err)
			WriteBytesViaStream(initSocketConnStream, reqBytes)

			clientSocketDataStream, err := (*n.Host).NewStream(context.Background(), servicePeerId, protocol.ID(ProxySocketDataFromClientSide))
			if err != nil {
				log.Printf("forward service data request err: %+v\n", err)
				return
			}

			// Save client to CSGW connection in node, which will be used in stream handler function
			n.clientSocketConns[requestId] = &conn

			clientDataReader := bufio.NewReader(conn)
			clientDataBuf := make([]byte, 4096)

			go func() {
				for {
					msgSize, err := clientDataReader.Read(clientDataBuf)
					internal.CheckError(err)

					log.Printf("ClientSideGateway: Received %d byte message from client: %q\n", msgSize, clientDataBuf)

					forwardData := &models.ApronServiceData{
						RequestId: req.RequestId,
						RawData:   clientDataBuf,
					}

					forwardDataBytes, err := proto.Marshal(forwardData)
					internal.CheckError(err)
					WriteBytesViaStream(clientSocketDataStream, forwardDataBytes)
					log.Println("ClientSideGateway: socket data written to stream, wait for next client ws msg")
				}
			}()

			// Infinity loop to keep the connection
			select {}
		}(conn)
	}
}
