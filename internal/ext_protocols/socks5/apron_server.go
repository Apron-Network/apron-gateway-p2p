package socks5

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"apron.network/gateway-p2p/internal/trans_network"
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
	SocketAddr   string

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

func NewApronAgentServer(socksConf *Config, agentConf *ApronAgentServerConfig, logger *zap.Logger) (*ApronAgentServer, error) {
	// TODO: AuthMethods is ignored currently
	// TODO: Rule set is ignored currently

	// Ensure we have a DNS resolver
	if socksConf.Resolver == nil {
		socksConf.Resolver = DNSResolver{}
	}

	if socksConf.Logger == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}
		socksConf.Logger = logger
	}

	server := &ApronAgentServer{
		socks5Config: socksConf,
		agentConfig:  agentConf,
		logger:       logger,
	}

	return server, nil
}

func (s *ApronAgentServer) ListenAndServe(networkType string) error {
	l, err := net.Listen(networkType, s.agentConfig.ListenAddr)
	if err != nil {
		s.logger.Error("listen on addr error", zap.Error(err), zap.String("addr", s.agentConfig.ListenAddr))
		return err
	}

	for {
		connWithClientOrService, err := l.Accept()
		if err != nil {
			s.logger.Error("accept connection error", zap.Error(err), zap.String("addr", s.agentConfig.ListenAddr))
			return err
		}
		s.logger.Sugar().Infof("accept connection on %s, mode: %+v, client addr: %+v", s.agentConfig.ListenAddr, s.agentConfig.Mode, connWithClientOrService.RemoteAddr())

		err = s.prepareAgent(s.agentConfig.ListenAddr)
		internal.CheckError(err)

		go func() {
			err := s.serveConnection(connWithClientOrService)
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
		s.logger.Sugar().Info("server agent prepare task: register agent to SSGW")
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
// 1. Read greeting message, and send NoAuth response (TODO: embed serviceId and userId in request URL)
// 2. Get `version` and `cmd` in next request, verify those are supported flags
// 3. Create ApronInitRequest
func (s *ApronAgentServer) serveConnection(connWithClientOrService net.Conn) error {
	switch s.agentConfig.Mode {
	case ClientAgentMode:
		initRequest, err := s.buildCaInitRequest(connWithClientOrService)
		if err != nil {
			s.logger.Error("prepare init request error", zap.Error(err))
			return err
		}

		// Connect to CSGW with ApronSocketInitRequest
		// TODO: Parse client request and get real service ID
		var serviceId string
		serviceId, found := os.LookupEnv("APRON_SOCKS_SERVICE_ID")
		if !found {
			serviceId = "apron_hello_socks5"
		}

		connWithCsgw, err := net.Dial("tcp", s.agentConfig.SocketAddr)
		if err != nil {
			return err
		}

		socketInitRequest := models.ApronSocketInitRequest{ServiceId: serviceId}
		requestBytes, _ := proto.Marshal(&socketInitRequest)
		trans_network.WriteBytesViaStream(connWithCsgw, requestBytes)

		// Build initRequest for socks5 connection
		// Note: the following package will be sent via SocketData stream, which will be parsed at service agent side,
		// so protocol specified types are requied here.

		s.logger.Debug("send init request to CSGW", zap.Any("init_request", initRequest))

	case ServerAgentMode:
		s.logger.Debug("Agent int ServerAgentMode")
		defer connWithClientOrService.Close()
		reader := bufio.NewReader(connWithClientOrService)

		// TODO: echo server for testing
		for {
			// read client request data
			bytes, err := reader.ReadBytes(byte('\n'))
			if err != nil {
				if err != io.EOF {
					fmt.Println("failed to read data, err:", err)
				}
				return err
			}
			fmt.Printf("request: %q", bytes)

			line := fmt.Sprintf("response: %q", bytes)
			connWithClientOrService.Write([]byte(line))
		}

		// TODO: TODO: Check how ssgw send data to target agent, then write code here to parse init request

		// Parse ApronInitRequest
		// TODO: Listen to s.socks5Config.MsgCh, then decode the ApronInitRequest first
	}

	return nil
}

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

// buildCaInitRequest builds init request from client agent based on greeting package from client, which contains those logics:
// * Validate version for the request and return error to client if not match
// * Return NoAuth
// try to get version and command from request sent from socks5 client,
// then build ApronSocks5ConnectRequest and return.
func (s *ApronAgentServer) buildCaInitRequest(conn net.Conn) (*ApronSocks5ConnectRequest, error) {
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
