package main

import (
	"context"
	"flag"
	"log"
	"sync"

	"apron.network/gateway-p2p/internal/ipfs_agent"
	"apron.network/gateway-p2p/internal/logger"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/trans_network"
)

func main() {
	var wg sync.WaitGroup
	ctx := context.Background()

	config := &trans_network.NodeConfig{}
	flag.Var(&config.BootstrapPeers, "peers", "Bootstrap Peers")
	flag.IntVar(&config.InternalPort, "p2p-port", 2145, "Internal Port Used by p2p network")
	flag.StringVar(&config.ForwardServiceAddr, "service-addr", ":8080", "Service addr used for proxy")
	flag.StringVar(&config.SocketForwardServiceAddr, "socket-service-addr", "localhost:9980", "Socket service addr used for proxy")
	flag.StringVar(&config.MgmtAddr, "mgmt-addr", ":8082", "API base for management")
	flag.StringVar(&config.Rendezvous, "rendezvous", "ApronServiceNetwork", "Rendezvous to build DHT network")
	flag.IntVar(&config.SecretKey, "secret-key", 0, "Secret key to specified host id")
	flag.IntVar(&config.ReportInterval, "report-interval", 15, "Upload usage report interval second")
	flag.StringVar(&config.LogConfig.BaseDir, "log-dir", "/var/log/", "Base log directory for application")
	flag.StringVar(&config.LogConfig.Level, "log-level", "info", "Output log level")
	loggerName := flag.String("logger-name", "gateway", "logger name")
	apiKey := flag.String("ipfs-key", "", "Api key for IPFS agent")
	apiSecret := flag.String("ipfs-secret", "", "Api secret for IPFS agent")
	flag.Parse()

	logger.InitLogger(config.LogConfig, *loggerName)
	node, err := trans_network.NewNode(ctx, config)
	internal.CheckError(err)

	log.Printf("Host ID: %s", (*node.Host).ID().String())
	log.Printf("Connect to me on:")
	for _, addr := range (*node.Host).Addrs() {
		log.Printf("  %s/p2p/%s", addr, (*node.Host).ID().String())
	}

	// Setup listener for service broadcast
	node.SetupServiceBroadcastListener(ctx)

	// Start monitor for peers.
	go node.UpdatePeers()

	// Setup stream handler
	node.SetProxyStreamHandlers()

	kdht, err := trans_network.NewKDHT(ctx, *node.Host, config.BootstrapPeers, &wg)
	internal.CheckError(err)

	// Start discover goroutines
	go trans_network.Discover(ctx, node, kdht, config.Rendezvous)

	// Setup listener for management service
	go node.StartMgmtApiServer()

	// Setup proxy request handler
	go node.StartForwardService()

	go node.StartSocketForwardService()

	// Upload log file to IPFS
	// TODO: Replace with real IPFS agent
	agent := ipfs_agent.LocalFileAgent{
		APIKey:    *apiKey,
		APISecret: *apiSecret,
	}
	go node.StartUploadUsageReportTask(config.ReportInterval, &agent)

	select {}
}
