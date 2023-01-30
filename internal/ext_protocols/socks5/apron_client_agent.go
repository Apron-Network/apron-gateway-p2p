package socks5

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"os"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// This file contains client related functions of agent

// handleClientGreetingPackage sent from client side, the greeting package contains version and auth method accepted by client
// Currently the server only supports NoAuth, which will be updated later
func (s *ApronAgentServer) handleClientGreetingPackage(reader *bufio.Reader, conn net.Conn) error {
	err := s.validateVersion(reader)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	numAuthMethods, err := reader.ReadByte()
	if err != nil {
		s.logger.Error("read count of auth method error", zap.Error(err))
		return err
	}

	authMethods := make([]byte, numAuthMethods)
	reader.Read(authMethods)

	// TODO: change to username auth method here, and the username/password will be used to identify Apron user
	// returns no auth for now
	_, err = conn.Write([]byte{socks5Version, socks5AuthNone})
	return err
}

// buildSocks5ConnectRequest builds init request from client agent based on greeting package from client, which contains those logics:
// * Validate version for the request and return error to client if not match
// * Return NoAuth
// try to get version and command from request sent from socks5 client,
// then build ApronSocks5ConnectRequest and return.
func (s *ApronAgentServer) buildSocks5ConnectRequest(conn net.Conn) (*ApronSocks5ConnectRequest, error) {
	reader := bufio.NewReader(conn)

	err := s.handleClientGreetingPackage(reader, conn)
	if err != nil {
		s.logger.Error("handle greeting package error", zap.Error(err))
		return nil, err
	}

	// TODO: Prepare auth package

	// Process connection request package
	// The connection request contains version, command and dest addr and port.
	// For now, only *connect* command is supported, bind and associate command is not supporting now
	buf := make([]byte, 3)
	readNum, err := reader.Read(buf)
	if err != nil {
		s.logger.Error("read client connection request package error", zap.Error(err))
		return nil, err
	}

	if readNum != 3 {
		err = errors.New("read size not equals 3")
		s.logger.Error("read client connection request package error", zap.Error(err))
		return nil, err
	}

	version := buf[0]
	command := buf[1]

	connRequest := &ApronSocks5ConnectRequest{
		Version: version,
		Command: command,
	}

	// Read dest address in connection request
	addrType := []byte{0}
	if _, err := reader.Read(addrType); err != nil {
		return nil, err
	}

	switch addrType[0] {
	case ipv4Address:
		addr := make([]byte, 4)
		if _, err := io.ReadAtLeast(reader, addr, len(addr)); err != nil {
			return nil, err
		}
		connRequest.DestAddr = net.IP(addr).String()

	case ipv6Address:
		addr := make([]byte, 16)
		if _, err := io.ReadAtLeast(reader, addr, len(addr)); err != nil {
			return nil, err
		}
		connRequest.DestAddr = net.IP(addr).String()

	case fqdnAddress:
		if _, err := reader.Read(addrType); err != nil {
			return nil, err
		}
		addrLen := int(addrType[0])
		fqdn := make([]byte, addrLen)
		if _, err := io.ReadAtLeast(reader, fqdn, addrLen); err != nil {
			return nil, err
		}
		connRequest.DestAddr = string(fqdn)

	default:
		return nil, unrecognizedAddrType
	}

	// dest port
	// Read the port
	port := []byte{0, 0}
	if _, err := io.ReadAtLeast(reader, port, 2); err != nil {
		return nil, err
	}
	connRequest.DestPort = (int(port[0]) << 8) | int(port[1])

	return connRequest, nil
}

// connectToCsgwAndSendInitRequest builds init ApronSocketInitRequest and sends to CSGW
// Currently the only params sent is *serviceId*, which will be used to find related SSGW and establish connection
func (s *ApronAgentServer) connectToCsgwAndSendInitRequest() error {
	// Connect to CSGW and send ApronSocketInitRequest
	// TODO: Parse client request and get real service ID
	s.logger.Info(
		"got init request from client",
		zap.String("TODO", "get service id from request"),
	)
	var serviceId string
	serviceId, found := os.LookupEnv("APRON_SOCKS_SERVICE_ID")
	if !found {
		serviceId = "apron_hello_socks5"
	}

	csgwConn, err := net.Dial("tcp", s.agentConfig.RemoteSocketAddr)
	if err != nil {
		return "", err
	}
	s.logger.Debug("connection to CSGW established", zap.Any("csgw_conn", csgwConn))
	s.agentConfig.RemoteSocketConn = csgwConn

	requestId := fmt.Sprintf("%s.%s.%s", "shouldBeUid", serviceId, uuid.New().String())
	socketInitRequest := models.ApronSocketInitRequest{ServiceId: serviceId, RequestId: requestId}
	requestBytes, _ := proto.Marshal(&socketInitRequest)
	trans_network.WriteBytesViaStream(csgwConn, requestBytes)

	return requestId, nil
}

func (s *ApronAgentServer) proxyDataFromClient(clientConn net.Conn, requestId string) {
	s.logger.Info("entering proxyDataFromClient function")

	// Proxy data from client to CSGW
	go func(clientC, csgwC net.Conn) {
		buf := make([]byte, 4096)
		for {
			// Read client message
			reader := bufio.NewReader(clientC)
			readCnt, err := reader.Read(buf)
			internal.CheckError(err)

			s.logger.Info("CA: got client data", zap.ByteString("resp_content", buf[:readCnt]))
		}
	}(clientConn, s.agentConfig.RemoteSocketConn)

	// Proxy data from CSGW to Client
	go func(clientC, csgwC net.Conn) {
		buf := make([]byte, 4096)
		for {
			// Read client message
			reader := bufio.NewReader(csgwC)
			readCnt, err := reader.Read(buf)
			internal.CheckError(err)

			serviceData := models.ExtServiceData{}
			err = proto.Unmarshal(buf[:readCnt], &serviceData)
			internal.CheckError(err)

			s.logger.Info("CA: got CSGW data", zap.ByteString("resp_content_byte", buf[:readCnt]), zap.Any("service_data", serviceData))
			log.Printf("content bytes: %+v\n", serviceData.Content)

			writeCnt, err := clientC.Write(serviceData.Content)
			internal.CheckError(err)
			s.logger.Info("written data to client", zap.Int("written_data_size", writeCnt))
		}
	}(clientConn, s.agentConfig.RemoteSocketConn)

}
