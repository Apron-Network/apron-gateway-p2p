package socks5

const (
	socks5Version              = byte(5)
	socks5AuthNone             = byte(0)
	socks5AuthUsernamePassword = byte(2)
	socks5ConnectCommand       = byte(1)
	socks5IPv4                 = byte(1)
	socks5IPv6                 = byte(4)
	socks5Domain               = byte(3)

	// socks5ServiceName indicates the service name, which will be set in data sent to Apron network and target service
	socks5ServiceName = "socks5"

	socks5ConnectMessage = 1
)
