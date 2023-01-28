package socks5

import (
	"bufio"
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
)

type Config struct {
	AuthMethods []Authenticator
	Credentials CredentialStore
	Resolver    NameResolver
	Rules       RuleSet
	Rewriter    AddressRewriter
	BindIP      net.IP
	Logger      *zap.Logger
	Dial        func(ctx context.Context, network, addr string) (net.Conn, error)
	MsgCh       *chan []byte
}
type Server struct {
	config      *Config
	authMethods map[uint8]Authenticator
}

// New creates a new Server and potentially returns an error
func New(conf *Config) (*Server, error) {
	// Ensure we have at least one authentication method enabled
	if len(conf.AuthMethods) == 0 {
		if conf.Credentials != nil {
			conf.AuthMethods = []Authenticator{&UserPassAuthenticator{conf.Credentials}}
		} else {
			conf.AuthMethods = []Authenticator{&NoAuthAuthenticator{}}
		}
	}

	// Ensure we have a DNS resolver
	if conf.Resolver == nil {
		conf.Resolver = DNSResolver{}
	}

	// Ensure we have a rule set
	if conf.Rules == nil {
		conf.Rules = PermitAll()
	}

	// Ensure we have a log target
	if conf.Logger == nil {
		conf.Logger = zap.NewExample()
	}

	server := &Server{
		config: conf,
	}

	server.authMethods = make(map[uint8]Authenticator)

	for _, a := range conf.AuthMethods {
		server.authMethods[a.GetCode()] = a
	}

	return server, nil
}

// ListenAndServe is used to create a listener and serve on it
func (s *Server) ListenAndServe(networkType, addr string, apronMode bool) error {
	l, err := net.Listen(networkType, addr)
	if err != nil {
		return err
	}
	return s.Serve(l, apronMode)
}

// Serve is used to serve connections from a listener
func (s *Server) Serve(l net.Listener, apronMode bool) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go s.ServeConnection(conn, apronMode)
	}
}

func (s *Server) ServeConnection(conn net.Conn, apronMode bool) error {
	//if apronMode {
	//	// TODO: Apron: Send request object to csgw, add protocol type (socks5)
	//	// Send request dest IP, auth context to target service via ssgw
	//
	//	// Negotiation Phase
	//	// 1. Create socks5.ApronSocks5ConnectRequest object
	//	// 2. Package it into ExtServiceData
	//	// 3. Send to target service via Apron network
	//	// 4. Wait for response
	//
	//	// Build init request, which will be sent to target server and check whether it can be served
	//	initRequest, err := s.buildCaInitRequest(conn)
	//	internal.CheckError(err)
	//
	//	initRequestBytes, err := binary.Marshal(initRequest)
	//	internal.CheckError(err)
	//
	//	reqData := &models.ExtServiceData{
	//		ServiceName: socks5ServiceName,
	//		ContentType: uint32(InitRequest),
	//		Content:     initRequestBytes,
	//	}
	//
	//	reqBytes, err := proto.Marshal(reqData)
	//	internal.CheckError(err)
	//
	//	*s.socks5Config.MsgCh <- reqBytes
	//}
	defer conn.Close()
	request, err := s.prepareRequest(conn)
	if err != nil {
		err = fmt.Errorf("failed to prepare request: %v", err)
		s.config.Logger.Error("prepare request error", zap.Error(err))
		return err
	}

	if client, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		request.RemoteAddr = &AddrSpec{IP: client.IP, Port: client.Port}
	}

	// Process the client request
	if err := s.handleRequest(request, conn); err != nil {
		err = fmt.Errorf("failed to handle request: %v", err)
		s.config.Logger.Error("handle request error", zap.Error(err))
		return err
	}
	return nil
}

func (s *Server) prepareRequest(conn net.Conn) (*Request, error) {
	bufConn := bufio.NewReader(conn)

	// Read the version byte
	version := []byte{0}
	if _, err := bufConn.Read(version); err != nil {
		s.config.Logger.Error("socks: Failed to get version byte", zap.Error(err))
		return nil, err
	}

	// Ensure we are compatible
	if version[0] != socks5Version {
		err := fmt.Errorf("unsupported SOCKS version: %v", version)
		s.config.Logger.Error("socks error", zap.Error(err))
		return nil, err
	}

	// Authenticate the connection
	authContext, err := s.authenticate(conn, bufConn)
	if err != nil {
		err = fmt.Errorf("failed to authenticate: %v", err)
		s.config.Logger.Error("socks error", zap.Error(err))
		return nil, err
	}

	request, err := NewRequest(bufConn)
	if err != nil {
		if err == unrecognizedAddrType {
			if err := sendReply(conn, addrTypeNotSupported, nil); err != nil {
				return nil, fmt.Errorf("failed to send reply: %v", err)
			}
		}
		return nil, fmt.Errorf("failed to read destination address: %v", err)
	}

	request.AuthContext = authContext

	return request, nil
}
