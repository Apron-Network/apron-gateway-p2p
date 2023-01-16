package main

import (
	"flag"
	"log"
	"net"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/ext_protocols/socks5"
	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// socks_user_agent stands between csgw and user app, which provides same interface with apron demo grpc service,
// For connecting service via Apron network, client should enable something like Apron mode
// In the apron mode, client will try to pass

func sendSocketInitRequest(serviceId string, csgwConn net.Conn) {
	socketInitRequest := models.ApronSocketInitRequest{ServiceId: serviceId}

	// Send init socket request
	requestBytes, _ := proto.Marshal(&socketInitRequest)
	trans_network.WriteBytesViaStream(csgwConn, requestBytes)
}

// TODO: Apron: Add client msg channel to server config and pass to new socks5 server
func startApronSocks5UserSideAgent(clientMsgCh *chan []byte) {
	logger, err := zap.NewProduction()
	internal.CheckError(err)

	// Socks5 service
	conf := &socks5.Config{
		MsgCh: clientMsgCh,
		Logger: logger,
	}

	server, err := socks5.NewApronServer(conf, socks5.ClientSideMode)
	if err != nil {
		logger.Error("new socks5 server error", zap.Error(err))
		panic(err)
	}

	if err := server.ListenAndServe("tcp", "127.0.0.1:9876"); err != nil {
		panic(err)
	}
}

func connectToCsgw(csgwAddr string, serviceId string, msgCh *chan []byte) {
	csgwConn, err := net.Dial("tcp", csgwAddr)
	internal.CheckError(err)

	// ServiceId should be passed via socket connect URI, (u or p field)
	// After receiving connect request, app will first send socket init request with parsed service id
	// After connection between csgw (and service) established, app response to client that connection is okay
	// Then client can send data to app, app will forward data to service in apron way

	// TODO: Prod: Parse URI and get ServiceID from u or p field

	sendSocketInitRequest(serviceId, csgwConn)

	// Create go routines for send and receive messages
	buf := make([]byte, 4096)
	go func() {
		for {
			msgSize, err := csgwConn.Read(buf)
			internal.CheckError(err)
			log.Printf("Server resp: %q\n", buf[:msgSize])
		}
	}()

	// coroutine to send data to server
	go func() {
		for {
			select {
			case msgToServer := <-*msgCh:
				log.Printf("Received msg %q\n", msgToServer)
				csgwConn.Write(msgToServer)
			}
		}
	}()
}

func main() {
	csgwAddr := flag.String("server-addr", "localhost:8083", "Server addr with host:port format")

	// TODO: serviceId will be parsed from client connect address, mostly like u or p field.
	serviceId := flag.String("service-id", "socks_hello_apron", "Service ID for using service")
	flag.Parse()

	msgCh := make(chan []byte)

	go connectToCsgw(*csgwAddr, *serviceId, &msgCh)
	go startApronSocks5UserSideAgent(&msgCh)

	select {}
}
