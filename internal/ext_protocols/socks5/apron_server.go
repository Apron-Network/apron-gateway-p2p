package socks5

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/kelindar/binary"
	"go.uber.org/zap"
)

type ApronServerMode uint8

const (
	ClientSideMode ApronServerMode = iota
	ServerSideMode
)

type ApronServer struct {
	config *Config
	mode   ApronServerMode
}

func NewApronServer(conf *Config, mode ApronServerMode) (*ApronServer, error) {
	// TODO: AuthMethods is ignored currently
	// TODO: Rule set is ignored currently

	// Ensure we have a DNS resolver
	if conf.Resolver == nil {
		conf.Resolver = DNSResolver{}
	}

	if conf.Logger == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}
		conf.Logger = logger
	}

	server := &ApronServer{
		config: conf,
		mode:   mode,
	}

	return server, nil
}

func (s *ApronServer) ListenAndServe(networkType, addr string) error {
	l, err := net.Listen(networkType, addr)
	if err != nil {
		s.config.Logger.Error("listen on addr error", zap.Error(err), zap.String("addr", addr))
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			s.config.Logger.Error("accept connection error", zap.Error(err), zap.String("addr", addr))
			return err
		}
		s.config.Logger.Sugar().Infof("accept connection on %s, mode: %+v, client addr: %+v", addr, s.mode, conn.RemoteAddr())

		go s.ServeConnection(conn)
	}
}

func (s *ApronServer) ServeConnection(conn net.Conn) error {
	switch s.mode {
	case ClientSideMode:
		initRequest, err := s.prepareApronInitRequest(conn)
		if err != nil {
			s.config.Logger.Error("prepare init request error", zap.Error(err))
			return err
		}

		initRequestByte, err := binary.Marshal(initRequest)
		if err != nil {
			s.config.Logger.Error("marshal init requst error", zap.Error(err))
			return err
		}

		s.config.Logger.Debug("send init request to CSGW", zap.Any("init_request", initRequest))
		*s.config.MsgCh <- initRequestByte
	case ServerSideMode:
		s.config.Logger.Debug("serve int ServerSideMode")
		defer conn.Close()
		reader := bufio.NewReader(conn)

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
			conn.Write([]byte(line))
		}

		// TODO: TODO: Check how ssgw send data to target agent, then write code here to parse init request

		// Parse ApronInitRequest
		// TODO: Listen to s.config.MsgCh, then decode the ApronInitRequest first
	}

	return nil
}

// handleGreetingPackage sent from client side, the greeting package contains version and auth method accepted by client
// Currently the server only supports NoAuth, which will be updated later
func (s *ApronServer) handleGreetingPackage(reader *bufio.Reader, conn net.Conn) error {
	err := s.validateVersion(reader)
	if err != nil {
		s.config.Logger.Error(err.Error())
		return err
	}

	numAuthMethods, err := reader.ReadByte()
	if err != nil {
		s.config.Logger.Error("read count of auth method error", zap.Error(err))
		return err
	}

	authMethods := make([]byte, numAuthMethods)
	reader.Read(authMethods)

	// TODO: change to username auth method here, and the username/password will be used to identify Apron user
	// returns no auth for now
	_, err = conn.Write([]byte{socks5Version, socks5AuthNone})
	return err
}

func (s *ApronServer) validateVersion(reader *bufio.Reader) error {
	version, err := reader.ReadByte()
	if err != nil {
		s.config.Logger.Error("read socks version error", zap.Error(err))
		return err
	}

	if version != socks5Version {
		err = fmt.Errorf("invalid socks version: %d", version)
		s.config.Logger.Error(err.Error())
		return err
	}

	return nil
}

// prepareApronInitRequest try to get version and command from request sent from socks5 client,
// then build ApronSocks5ConnectRequest and return.
func (s *ApronServer) prepareApronInitRequest(conn net.Conn) (*ApronSocks5ConnectRequest, error) {
	reader := bufio.NewReader(conn)

	err := s.handleGreetingPackage(reader, conn)
	if err != nil {
		s.config.Logger.Error("handle greeting package error", zap.Error(err))
		return nil, err
	}

	// TODO: Prepare auth package

	// Process connection request package
	// The connection request contains version, command and dest addr and port.
	// For now, only *connect* command is supported, bind and associate command is not supporting now
	buf := make([]byte, 3)
	readNum, err := reader.Read(buf)
	if err != nil {
		s.config.Logger.Error("read client connection request package error", zap.Error(err))
		return nil, err
	}

	if readNum != 3 {
		err = errors.New("read size not equals 3")
		s.config.Logger.Error("read client connection request package error", zap.Error(err))
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
