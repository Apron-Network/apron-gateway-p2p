package socks5

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"apron.network/gateway-p2p/internal"
	"github.com/kelindar/binary"
)

const (
	socks5Version = uint8(5)
)

type Config struct {
	AuthMethods []Authenticator
	Credentials CredentialStore
	Resolver    NameResolver
	Rules       RuleSet
	Rewriter    AddressRewriter
	BindIP      net.IP
	Logger      *log.Logger
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
		conf.Logger = log.New(os.Stdout, "", log.LstdFlags)
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
	defer conn.Close()
	request, err := s.prepareRequest(conn)
	if err != nil {
		err = fmt.Errorf("failed to prepare request: %v", err)
		s.config.Logger.Printf("[ERR] socks: %v", err)
		return err
	}

	if apronMode {
		// TODO: Apron: Send request object to csgw

		encodedBytes, err := binary.Marshal(request)
		internal.CheckError(err)
		*s.config.MsgCh <- encodedBytes
	} else {
		if client, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
			request.RemoteAddr = &AddrSpec{IP: client.IP, Port: client.Port}
		}

		// Process the client request
		if err := s.handleRequest(request, conn); err != nil {
			err = fmt.Errorf("failed to handle request: %v", err)
			s.config.Logger.Printf("[ERR] socks: %v", err)
			return err
		}
	}

	return nil

}

func (s *Server) prepareRequest(conn net.Conn) (*Request, error) {
	bufConn := bufio.NewReader(conn)

	// Read the version byte
	version := []byte{0}
	if _, err := bufConn.Read(version); err != nil {
		s.config.Logger.Printf("[ERR] socks: Failed to get version byte: %v", err)
		return nil, err
	}

	// Ensure we are compatible
	if version[0] != socks5Version {
		err := fmt.Errorf("unsupported SOCKS version: %v", version)
		s.config.Logger.Printf("[ERR] socks: %v", err)
		return nil, err
	}

	// Authenticate the connection
	authContext, err := s.authenticate(conn, bufConn)
	if err != nil {
		err = fmt.Errorf("failed to authenticate: %v", err)
		s.config.Logger.Printf("[ERR] socks: %v", err)
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
