package socks5

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
	"github.com/google/uuid"
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
	case socks5IPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadAtLeast(reader, addr, len(addr)); err != nil {
			return nil, err
		}
		connRequest.DestAddr = net.IP(addr).String()

	case socks5IPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadAtLeast(reader, addr, len(addr)); err != nil {
			return nil, err
		}
		connRequest.DestAddr = net.IP(addr).String()

	case socks5Domain:
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
		return nil, unknownAddressType
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
func (s *ApronAgentServer) connectToCsgwAndSendInitRequest() (string, error) {
	// Connect to CSGW and send ApronSocketInitRequest
	// TODO: Parse client request and get real service ID
	s.logger.Info(
		"got init request from client",
		zap.String("TODO", "get serviceId and userId from request"),
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
		buf := make([]byte, 40960)
		for {
			// Read client message
			reader := bufio.NewReader(clientC)
			readCnt, err := reader.Read(buf)
			if err != nil {
				s.logger.Error("read data from client error",
					zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
					zap.Error(err),
				)
				if err.Error() == "EOF" {
					break
				} else {
					continue
				}
			}

			s.logger.Info("CA: got client data", zap.Int("data_size", readCnt))

			// Package data to CSGW socket connection
			serviceData := &models.ExtServiceData{
				ServiceName: socks5ServiceName,
				ContentType: socks5DataMessage,
				RequestId:   requestId,
				MsgId:       uuid.NewString(),
				Content:     buf[:readCnt],
			}
			dataBytes, err := proto.Marshal(serviceData)
			if err != nil {
				s.logger.Error("marshal ExtServiceData failed",
					zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
					zap.Error(err),
				)
				if err.Error() == "EOF" {
					break
				} else {
					continue
				}
			}

			s.logger.Info("CA: package client request in ExtServiceData and send to CSGW", zap.Int("data_size", len(dataBytes)))

			trans_network.WriteBytesViaStream(csgwC, dataBytes)
		}
	}(clientConn, s.agentConfig.RemoteSocketConn)

	// Proxy data from CSGW to Client
	go func(clientC, csgwC net.Conn) {
		for {
			// Read ExtServiceData sent from CSGW
			reader := bufio.NewReader(csgwC)
			csgwDataBytes, err := trans_network.ReadOneFrameDataFromStream(reader)
			if err != nil {
				s.logger.Error("read data from CSGW error",
					zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
					zap.Error(err),
				)
				if err.Error() == "EOF" {
					break
				} else {
					continue
				}
			}

			serviceData := models.ExtServiceData{}
			err = proto.Unmarshal(csgwDataBytes, &serviceData)
			if err != nil {
				s.logger.Error("unmarshal ExtServiceData bytes from CSGW error",
					zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
					zap.ByteString("received_data", csgwDataBytes),
					zap.Error(err),
				)
				if err.Error() == "EOF" {
					break
				} else {
					continue
				}
			}

			s.logger.Info("CA: got CSGW data",
				zap.Int("data_size", len(csgwDataBytes)),
				zap.Any("service_data", serviceData),
			)

			// Write raw content to client
			writeCnt, err := clientC.Write(serviceData.Content)
			if err != nil {
				s.logger.Error("write data to client error",
					zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
					zap.ByteString("data_to_be_returned", serviceData.Content),
					zap.Error(err),
				)
				if err.Error() == "EOF" {
					break
				} else {
					continue
				}
			}
			s.logger.Info("written data to client", zap.Int("written_data_size", writeCnt))
		}
	}(clientConn, s.agentConfig.RemoteSocketConn)
}
