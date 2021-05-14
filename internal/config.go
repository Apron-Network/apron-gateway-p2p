package internal

import "apron.network/gateway-p2p/internal/trans_network"

type GatewayConfig struct {
	BootstrapPeers trans_network.AddrList
	ConnectPort    int32
	ProxyPort      int32
	MgmtAddr       string
}
