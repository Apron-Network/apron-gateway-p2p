package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"net"
	"os"
	"time"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var logger *zap.SugaredLogger

func main() {
	logger = zap.NewExample().Sugar()
	subCommamd := os.Args[1]
	logger.Infof("sub command: %s", subCommamd)

	switch subCommamd {
	case "client":
		logger.Infof("client side agent")
		clientOpts := flag.NewFlagSet("client", flag.ExitOnError)
		csgwAddr := clientOpts.String("csgw-socket-addr", "", "Client side GW socket address")
		serviceId := clientOpts.String("service-id", "", "Service ID to be connected")
		clientOpts.Parse(os.Args[2:])

		go startSocketClient(*csgwAddr, *serviceId)
	case "server":
		logger.Infof("service side agent")
		serviceOpts := flag.NewFlagSet("service", flag.ExitOnError)
		serviceId := serviceOpts.String("service-id", "", "id of the service")
		listenAddr := serviceOpts.String("listen-addr", "", "Client side GW socket address")
		ssgwMgmtAddr := serviceOpts.String("ssgw-addr", "", "RESTful management API address for service side gateway")
		serviceOpts.Parse(os.Args[2:])

		go startSocketServer(*listenAddr, *ssgwMgmtAddr, *serviceId)
	}

	select {}
}

// Client related code
func startSocketClient(csgwAddr, serviceId string) {
	conn, err := net.Dial("tcp", csgwAddr)
	internal.CheckError(err)

	// Send init request
	socketInitRequest := models.ApronSocketInitRequest{ServiceId: serviceId}
	requestBytes, _ := proto.Marshal(&socketInitRequest)
	trans_network.WriteBytesViaStream(conn, requestBytes)
	time.Sleep(3 * time.Second)

	go sendAndReceiveData(conn)
}

func sendAndReceiveData(conn net.Conn) {
	buf := make([]byte, 40960)

	// Read data
	go func(c net.Conn) {
		for {
			reader := bufio.NewReader(conn)
			readCnt, err := reader.Read(buf)
			internal.CheckError(err)

			logger.Infof("Client: Got response: %q", buf[:readCnt])
		}
	}(conn)

	// Write data repeatedly
	go func(c net.Conn) {
		for {
			writeCnt, err := c.Write([]byte("hello"))
			internal.CheckError(err)
			logger.Infof("Client: sent data, size: %d", writeCnt)
			time.Sleep(time.Second)
		}
	}(conn)

}

// Server related code
func startSocketServer(listenAddr, ssgwMgmtAddr, serviceId string) {
	registerSocketService(ssgwMgmtAddr, listenAddr, serviceId)

	l, err := net.Listen("tcp", listenAddr)
	internal.CheckError(err)

	for {
		conn, err := l.Accept()
		internal.CheckError(err)
		logger.Infof("Connection established: %+v", conn)
		go echoRequestData(conn)
	}
}

func registerSocketService(ssgwAddr, listenAddr, serviceId string) {
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

	newServiceReqBytes, err := json.Marshal(newServiceRequest)
	internal.CheckError(err)

	req.Header.SetRequestURI(ssgwAddr)
	req.Header.SetMethod("POST")
	req.SetBody(newServiceReqBytes)

	err = fasthttp.Do(req, resp)
	internal.CheckError(err)

	respBytes := resp.Body()
	logger.Infof("Resp: %q\n", respBytes)
}

func echoRequestData(conn net.Conn) {
	for {
		buf := make([]byte, 4096)
		reader := bufio.NewReader(conn)

		readCnt, err := reader.Read(buf)
		internal.CheckError(err)
		logger.Infof("Server: Read %d bytes message, detail: %q", readCnt, buf[:readCnt])

		writeCnt, err := conn.Write(buf[:readCnt])
		internal.CheckError(err)
		logger.Infof("Server: Write %d bytes message", writeCnt)
	}
}
