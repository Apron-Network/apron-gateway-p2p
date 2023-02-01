package socks5

// ApronSocks5ConnectRequest saves some header info of request which will be sent to Apron network at first
type ApronSocks5ConnectRequest struct {
	Version  uint8
	Command  uint8
	DestAddr string // The dest address may be IPv4 address, IPv6 address or domain name
	DestPort int
}
