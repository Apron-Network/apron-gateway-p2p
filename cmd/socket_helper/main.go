package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"net"
	"os"
	"time"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/logger"
	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func main() {
	subCommand := os.Args[1]

	switch subCommand {
	case "client":
		logger.GetLogger().Info("command options",
			zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
			zap.String("cli_opts", subCommand))
		clientOpts := flag.NewFlagSet("client", flag.ExitOnError)
		csgwAddr := clientOpts.String("csgw-socket-addr", "", "Client side GW socket address")
		serviceId := clientOpts.String("service-id", "", "Service ID to be connected")
		logBaseDir := clientOpts.String("log-dir", "/var/log/", "Base log directory for application")
		logLevel := clientOpts.String("log-level", "info", "Output log level")
		err := clientOpts.Parse(os.Args[2:])
		internal.CheckError(err)

		logger.InitLogger(logger.LogConfig{
			BaseDir: *logBaseDir,
			Level:   *logLevel,
		}, "socket_helper_client")
		go startClientSideSocketAgent(*csgwAddr, *serviceId)
	case "server":
		logger.GetLogger().Info("command options",
			zap.String(trans_network.EntityFieldName, trans_network.EntitySA),
			zap.String("cli_opts", subCommand))
		serviceOpts := flag.NewFlagSet("service", flag.ExitOnError)
		serviceId := serviceOpts.String("service-id", "", "id of the service")
		listenAddr := serviceOpts.String("listen-addr", "", "Client side GW socket address")
		ssgwMgmtAddr := serviceOpts.String("ssgw-addr", "", "RESTful management API address for service side gateway")
		logBaseDir := serviceOpts.String("log-dir", "/var/log/", "Base log directory for application")
		logLevel := serviceOpts.String("log-level", "info", "Output log level")
		err := serviceOpts.Parse(os.Args[2:])
		internal.CheckError(err)

		logger.InitLogger(logger.LogConfig{
			BaseDir: *logBaseDir,
			Level:   *logLevel,
		}, "socket_helper_server")
		go startServerSideSocketAgent(*listenAddr, *ssgwMgmtAddr, *serviceId)
	}

	select {}
}

// Client related code
func startClientSideSocketAgent(csgwAddr, serviceId string) {
	logger.GetLogger().Debug("connect to CSGW",
		zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
		zap.String("csgw_addr", csgwAddr))

	conn, err := net.Dial("tcp", csgwAddr)
	internal.CheckError(err)

	// Send init request
	socketInitRequest := models.ApronSocketInitRequest{ServiceId: serviceId}
	requestBytes, _ := proto.Marshal(&socketInitRequest)
	trans_network.WriteBytesViaStream(conn, requestBytes)

	logger.GetLogger().Debug("sent socket init request to CSGW",
		zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
		zap.String("service_id", serviceId),
		zap.ByteString("init_request_bytes", requestBytes))

	time.Sleep(3 * time.Second)

	go sendAndReceiveData(conn)
}

func sendAndReceiveData(conn net.Conn) {
	requestBytes := []byte("hello")

	// Read data
	go func(c net.Conn) {
		for {
			reader := bufio.NewReader(conn)
			dataBuf, err := trans_network.ReadOneFrameDataFromStream(reader)
			internal.CheckError(err)

			logger.GetLogger().Info("got data from client",
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
				zap.Int("size", len(dataBuf)))
			logger.GetLogger().Debug("got data from client",
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
				zap.ByteString("content", dataBuf))
		}
	}(conn)

	// Write data repeatedly
	// Note: this application is used for demo so using this fake sender to emulate data sent from socket client.
	go func(c net.Conn) {
		for {
			trans_network.WriteBytesViaStream(conn, requestBytes)
			logger.GetLogger().Info("write data to client",
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
				zap.Int("size", len(requestBytes)))
			logger.GetLogger().Debug("got data from client",
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
				zap.ByteString("content", requestBytes))
			time.Sleep(time.Second)
		}
	}(conn)

}

// Server related code
func startServerSideSocketAgent(listenAddr, ssgwMgmtAddr, serviceId string) {
	registerSocketService(ssgwMgmtAddr, listenAddr, serviceId)

	l, err := net.Listen("tcp", listenAddr)
	internal.CheckError(err)

	for {
		conn, err := l.Accept()
		internal.CheckError(err)
		logger.GetLogger().Info("connection with SSGW established",
			zap.String(trans_network.EntityFieldName, trans_network.EntitySA),
		)
		go echoRequestData(conn)
	}
}

func registerSocketService(ssgwAddr, listenAddr, serviceId string) {
	logger.GetLogger().Info("prepare to register service to SSGW",
		zap.String(trans_network.EntityFieldName, trans_network.EntitySA),
		zap.String("ssgw_address", ssgwAddr),
		zap.String("listen_address", listenAddr),
		zap.String("service_id", serviceId),
	)
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	newServiceRequest := models.ApronService{
		Id:   serviceId,
		Name: serviceId,
		Providers: []*models.ApronServiceProvider{{
			Id:      serviceId,
			Name:    serviceId,
			BaseUrl: listenAddr,
			Schema:  "tcp",
		}},
	}

	logger.GetLogger().Debug("new service request",
		zap.String(trans_network.EntityFieldName, trans_network.EntitySA),
		zap.Any("new_service_request", newServiceRequest),
	)

	newServiceReqBytes, err := json.Marshal(newServiceRequest)
	internal.CheckError(err)

	req.Header.SetRequestURI(ssgwAddr)
	req.Header.SetMethod("POST")
	req.SetBody(newServiceReqBytes)

	err = fasthttp.Do(req, resp)
	internal.CheckError(err)

	respBytes := resp.Body()
	logger.GetLogger().Debug("response from SSGW",
		zap.String(trans_network.EntityFieldName, trans_network.EntitySA),
		zap.ByteString("ssgw_response", respBytes),
	)
}

func echoRequestData(conn net.Conn) {
	for {
		reader := bufio.NewReader(conn)
		dataBuf, err := trans_network.ReadOneFrameDataFromStream(reader)
		internal.CheckError(err)
		logger.GetLogger().Info("received message",
			zap.String(trans_network.EntityFieldName, trans_network.EntitySA),
			zap.Int("msg_len", len(dataBuf)),
			zap.ByteString("content", dataBuf))

		writeCnt, err := conn.Write(dataBuf)
		internal.CheckError(err)
		logger.GetLogger().Info("message sent back",
			zap.String(trans_network.EntityFieldName, trans_network.EntitySA),
			zap.Int("write_cnt", writeCnt),
			zap.ByteString("content", dataBuf))
	}
}
