package trans_network

type NodeConfig struct {
	BootstrapPeers           AddrList
	InternalPort             int
	ForwardServiceAddr       string
	MgmtAddr                 string
	SocketForwardServiceAddr string
	Rendezvous               string
	SecretKey                int
	PinataAPIKey             string
	PinataSecretKey          string
	ReportInterval           int
}
