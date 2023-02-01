package socks5

import (
	"context"
	"net"
)

type Config struct {
	Credentials CredentialStore
	Resolver    NameResolver
	BindIP      net.IP
	Dial        func(ctx context.Context, network, addr string) (net.Conn, error)
	MsgCh       *chan []byte
}
