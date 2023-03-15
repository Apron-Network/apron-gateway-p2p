package trans_network

import "apron.network/gateway-p2p/internal/logger"

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
	LogConfig                logger.LogConfig
}
