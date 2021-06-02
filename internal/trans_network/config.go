package trans_network

type TransNetworkConfig struct {
	BootstrapPeers     AddrList
	InternalPort       int
	ForwardServiceAddr string
	MgmtAddr           string
}
