package socks5

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	builtinBinary "encoding/binary"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/logger"
	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
	"github.com/google/uuid"
	"github.com/kelindar/binary"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// TODO: Currently the client side agent and service side agent are put in single struct, will refactor to separated structs

type ApronServerMode uint8

const (
	ClientAgentMode ApronServerMode = iota
	ServerAgentMode
)

type ApronAgentServerConfig struct {
	Mode ApronServerMode

	// Saves client or service side gateway address based on different mode
	RestMgmtAddr string

	// RemoteSocketAddr saves remote service this agent will connect to
	// in client agent mode, this address is CSGW socket proxy listen address
	// in server agent mode, this address is socket service listen address
	RemoteSocketAddr string

	// RemoteSocketConn saves connection between RemoteSocketAddr
	RemoteSocketConn net.Conn

	// Unify id for the agent, for service side agent, this will be used to register service to SSGW
	AgentId string

	// Agent service listen address
	ListenAddr string
}

type ApronAgentServer struct {
	socks5Config *Config
	agentConfig  *ApronAgentServerConfig

	logger *zap.Logger
}

func NewApronAgentServer(socksConf *Config, agentConf *ApronAgentServerConfig) (*ApronAgentServer, error) {
	// TODO: AuthMethods is ignored currently
	// TODO: Rule set is ignored currently

	// Ensure we have a DNS resolver
	if socksConf.Resolver == nil {
		socksConf.Resolver = DNSResolver{}
	}

	server := &ApronAgentServer{
		socks5Config: socksConf,
		agentConfig:  agentConf,
		logger:       logger.GetLogger(),
	}

	return server, nil
}

func (s *ApronAgentServer) ListenAndServe(networkType string) error {
	err := s.prepareAgent(s.agentConfig.ListenAddr)
	internal.CheckError(err)

	l, err := net.Listen(networkType, s.agentConfig.ListenAddr)
	if err != nil {
		s.logger.Error("listen on addr error", zap.Error(err), zap.String("addr", s.agentConfig.ListenAddr))
		return err
	}

	for {
		// connWithClientOrSsgw saves the socket connection for CA or SA
		// For CA, the other end of the connection is client
		// For SA, the other end of the connection is SSGW
		connWithClientOrSsgw, err := l.Accept()
		if err != nil {
			s.logger.Error("accept connection error", zap.Error(err), zap.String("addr", s.agentConfig.ListenAddr))
			return err
		}
		s.logger.Info("accept connection",
			zap.String("listen_addr", s.agentConfig.ListenAddr),
			zap.Any("mode", s.agentConfig.Mode),
			zap.Any("client_addr", connWithClientOrSsgw.RemoteAddr()),
		)

		go func() {
			err := s.serveConnection(connWithClientOrSsgw)
			internal.CheckError(err)
		}()
	}
}

// prepareAgent prepares required steps for starting agent service
// ClientSideAgent:
// ServerSideAgent:
//   - Register serverSideAgent to SSGW
func (s *ApronAgentServer) prepareAgent(listenAddr string) error {
	switch s.agentConfig.Mode {
	case ClientAgentMode:
		s.logger.Sugar().Info("client agent has no prepare task currently")
	case ServerAgentMode:
		newServiceRequest := models.ApronService{
			Id:   s.agentConfig.AgentId,
			Name: s.agentConfig.AgentId,
			Providers: []*models.ApronServiceProvider{{
				Id:      s.agentConfig.AgentId,
				Name:    s.agentConfig.AgentId,
				BaseUrl: listenAddr,
				Schema:  "tcp",
			}},
		}
		s.logger.Info("server agent prepare task: register agent to SSGW", zap.Any("req_request", newServiceRequest))
		respBytes, err := internal.RegisterServiceToSSGW(s.agentConfig.RestMgmtAddr, newServiceRequest)
		if err != nil {
			s.logger.Panic("Register service error", zap.Error(err))
			return err
		} else {
			s.logger.Sugar().Infof("Register service resp: %q", respBytes)
		}

	}

	return nil
}

// serveConnection is invoked after agent is prepared.
// For client side mode, the request sent from client contains socks5 header data, and those steps will be executed sequentially:
// 1. Read greeting message, and send socks5AuthNone response (TODO: embed serviceId and userId in request URL)
// 2. Get `version` and `cmd` in next request, verify those are supported flags
// 3. Create ApronInitRequest
func (s *ApronAgentServer) serveConnection(connWithClientOrSsgw net.Conn) error {
	switch s.agentConfig.Mode {
	case ClientAgentMode:
		// Build initRequest for socks5 connection
		requestId, err := s.connectToCsgwAndSendInitRequest()
		if err != nil {
			s.logger.Panic("connect to CSGW failed",
				zap.Error(err),
				zap.Any("agent_config", s.agentConfig),
				zap.Any("socks_config", s.socks5Config),
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
			)
			return err
		}
		s.logger.Info("connection to CSGW established and ApronSocketInitRequest message sent")

		// TODO: the sleep here is to avoid processing data request in advanced of init request,
		// TODO: since the init request handler should establish socket connection between service / agent side.
		// TODO: should be replaced with some notification strategies
		time.Sleep(100 * time.Microsecond)

		socksConnectRequest, err := s.buildSocks5ConnectRequest(connWithClientOrSsgw)
		if err != nil {
			s.logger.Panic("build socks5 connect request failed",
				zap.Error(err),
				zap.Any("agent_config", s.agentConfig),
				zap.Any("socks_config", s.socks5Config),
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
			)
			return err
		}
		s.logger.Info("socks connect request created", zap.Any("socks_connect_request", socksConnectRequest))

		// The following packages will be sent via SocketData stream, which will be parsed at service agent side.
		// The first package sent to CSGW should contain info about socks5 data,
		// then SSGW can recovery the socks5 request and send to service
		socksConnectRequestBytes, err := binary.Marshal(socksConnectRequest)
		if err != nil {
			s.logger.Panic("marshal socks5 connect request failed",
				zap.Error(err),
				zap.Any("agent_config", s.agentConfig),
				zap.Any("socks_config", s.socks5Config),
				zap.Any("connect_request", socksConnectRequest),
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
			)
			return err
		}

		packedSocks5ConnectMessage := models.ExtServiceData{
			ServiceName: socks5ServiceName,
			RequestId:   requestId,
			MsgId:       uuid.NewString(),
			ContentType: socks5ConnectMessage,
			Content:     socksConnectRequestBytes,
		}
		requestSentToCsgwBytes, err := proto.Marshal(&packedSocks5ConnectMessage)
		if err != nil {
			s.logger.Panic("marshal socks5 connect request to ext service data failed",
				zap.Error(err),
				zap.Any("agent_config", s.agentConfig),
				zap.Any("socks_config", s.socks5Config),
				zap.Any("connect_request", socksConnectRequest),
				zap.Any("ext_service_data", packedSocks5ConnectMessage),
				zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
			)
			return err
		}

		s.logger.Info("prepare sending data to CSGW",
			zap.Int("connect_request_size", len(socksConnectRequestBytes)),
			zap.Int("packed_socks5_connect_message_size", len(requestSentToCsgwBytes)),
			zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
		)

		trans_network.WriteBytesViaStream(s.agentConfig.RemoteSocketConn, requestSentToCsgwBytes)
		if err != nil {
			s.logger.Panic("send packed service data failed",
				zap.Error(err),
				zap.Any("agent_config", s.agentConfig),
				zap.Any("socks_config", s.socks5Config),
				zap.Any("connect_request", socksConnectRequest),
				zap.Any("ext_service_data", packedSocks5ConnectMessage),
			)
			return err
		}

		s.logger.Debug("socks5ConnectMessage sent to CSGW",
			zap.String("request_id", requestId),
			zap.String("msg_id", packedSocks5ConnectMessage.MsgId),
			zap.ByteString("data_content", requestSentToCsgwBytes),
			zap.String(trans_network.EntityFieldName, trans_network.EntityCA),
		)

		// Write success to client for getting more data
		// TODO: Change to use some message sent from SSGW/SA
		connWithClientOrSsgw.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		time.Sleep(100 * time.Microsecond)

		go s.proxyDataFromClient(connWithClientOrSsgw, requestId)
	case ServerAgentMode:
		s.logger.Debug("ServerAgentMode")
		reader := bufio.NewReader(connWithClientOrSsgw)

		dataCh := make(chan []byte)
		errCh := make(chan error)
		go trans_network.ReadBytesViaStream(reader, dataCh, errCh)

		serviceConnections := make(map[string]net.Conn)

		for {
			select {
			case serviceDataBytes := <-dataCh:
				s.logger.Info("got service data", zap.Int("data_size", len(serviceDataBytes)), zap.ByteString("data", serviceDataBytes))
				serviceData := &models.ExtServiceData{}
				err := proto.Unmarshal(serviceDataBytes, serviceData)
				internal.CheckError(err)

				s.logger.Info("got ext service data", zap.Any("service_data", serviceData))

				if serviceData.ServiceName != socks5ServiceName {
					s.logger.Panic("wrong service name", zap.Any("service_data", serviceData))
					return errors.New("incorrect service name")
				}

				switch serviceData.ContentType {
				case socks5ConnectMessage:
					s.logger.Info("received socks5ConnectMessage")
					socksConnectRequest := ApronSocks5ConnectRequest{}
					err := binary.Unmarshal(serviceData.Content, &socksConnectRequest)
					internal.CheckError(err)
					s.logger.Info("parsed connect message detail", zap.Any("connect_request", socksConnectRequest))

					connWithSocks5Service, err := s.connectToSocks5Service(&socksConnectRequest)
					serviceConnections[serviceData.RequestId] = connWithSocks5Service
					internal.CheckError(err)

					// proxy service data to SSGW
					go func(conn net.Conn) {
						buf := make([]byte, 40960)
						for {
							reader := bufio.NewReader(conn)

							readerCnt, err := reader.Read(buf)
							internal.CheckError(err)
							s.logger.Info("got response from service",
								zap.Binary("socks5_resp", buf[:readerCnt]),
								zap.Int("read_size", readerCnt),
							)
							log.Printf("service resp: %+v\n", buf[:readerCnt])

							// Package read data into ExtServiceData package and send back to SSGW
							respData := models.ExtServiceData{
								ServiceName: socks5ServiceName,
								ContentType: socks5DataMessage,
								Content:     buf[:readerCnt],
								MsgId:       uuid.NewString(),
								RequestId:   serviceData.RequestId,
							}
							respDataBytes, err := proto.Marshal(&respData)
							internal.CheckError(err)

							connWithClientOrSsgw.Write(respDataBytes)
						}
					}(connWithSocks5Service)
				case socks5DataMessage:
					s.logger.Info("received socks5DataMessage")
					serviceConnections[serviceData.RequestId].Write(serviceData.Content)
				}
			case errMsg := <-errCh:
				s.logger.Panic("got error message while reading data from stream", zap.Error(errMsg))
			}
		}
	}

	return nil
}

func (s *ApronAgentServer) validateVersion(reader *bufio.Reader) error {
	version, err := reader.ReadByte()
	if err != nil {
		s.logger.Error("read socks version error", zap.Error(err))
		return err
	}

	if version != socks5Version {
		err = fmt.Errorf("invalid socks version: %d", version)
		s.logger.Error(err.Error())
		return err
	}

	return nil
}

func (s *ApronAgentServer) connectToSocks5Service(apronSocksConnectRequest *ApronSocks5ConnectRequest) (net.Conn, error) {
	// Connect to service
	conn, err := net.Dial("tcp", s.agentConfig.RemoteSocketAddr)
	if err != nil {
		s.logger.Panic("failed to connect service", zap.String("service_addr", s.agentConfig.RemoteSocketAddr))
		return nil, err
	}

	// Send SOCKS5 authentication methods
	authMethods := []uint8{socks5AuthNone}
	if _, err := conn.Write([]byte{socks5Version, uint8(len(authMethods)), socks5AuthNone}); err != nil {
		s.logger.Panic("failed to send socks5AuthNone data")
		return nil, err
	}

	// Receive SOCKS5 server response
	authResponse := make([]byte, 2)
	if _, err := io.ReadFull(conn, authResponse); err != nil {
		s.logger.Panic("failed to receive authentication response", zap.ByteString("service_resp", authResponse))
		return nil, errors.New("failed to receive authentication response")
	}

	if authResponse[0] != socks5Version || authResponse[1] != socks5AuthNone {
		s.logger.Panic("unexpected authentication response", zap.ByteString("service_resp", authResponse))
		return nil, errors.New("unexpected authentication response")
	}

	// Send SOCKS5 connect request
	request := &bytes.Buffer{}
	request.WriteByte(socks5Version)
	request.WriteByte(socks5ConnectCommand)
	request.WriteByte(0) // Reserved

	// Domain name
	request.WriteByte(socks5Domain)
	request.WriteByte(uint8(len(apronSocksConnectRequest.DestAddr)))
	request.WriteString(apronSocksConnectRequest.DestAddr)

	// Port
	builtinBinary.Write(request, builtinBinary.BigEndian, uint16(apronSocksConnectRequest.DestPort))
	if _, err := conn.Write(request.Bytes()); err != nil {
		s.logger.Panic("Failed to send connect request", zap.Error(err))
		return nil, err
	}

	// Receive socks5 server response
	response := make([]byte, 256)
	serviceRespReader := bufio.NewReader(conn)
	readCnt, err := serviceRespReader.Read(response)
	if err != nil {
		s.logger.Panic("failed to receive connect response header", zap.Error(err))
		return nil, err
	}
	s.logger.Info("got data from service", zap.Int("data_size", readCnt))

	if response[0] != socks5Version || response[1] != 0 {
		s.logger.Panic("unexpected connect response:", zap.ByteString("resp_header", response[:2]))
		return nil, err
	}

	addrType := response[3]
	var addr string
	switch addrType {
	case socks5IPv4:
		ip := net.IP(response[4:8])
		addr = ip.String()
	case socks5Domain:
		domainLen := int(response[4])
		addr = string(response[5 : 5+domainLen])
	case socks5IPv6:
		ip := net.IP(response[4:20])
		addr = ip.String()
	}
	port := binary.BigEndian.Uint16(response[len(response)-2:])
	s.logger.Info("connection established", zap.String("addr", addr), zap.Uint16("port", port))

	return conn, nil
}
