package main

import (
	"flag"
	"fmt"
	"os"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/ext_protocols/socks5"
	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction()
	if len(os.Args) < 2 {
		printMainUsage()
		os.Exit(1)
	}
	subCommamd := os.Args[1]
	logger.Sugar().Infof("sub command: %s", subCommamd)

	socks5Config := &socks5.Config{
		Logger: logger,
	}

	switch subCommamd {
	case "ca":
		logger.Sugar().Infof("client side agent")
		clientOpts := flag.NewFlagSet("client", flag.ExitOnError)
		agentConfig := socks5.ApronAgentServerConfig{
			Mode: socks5.ClientAgentMode,
		}
		clientOpts.StringVar(&agentConfig.RemoteSocketAddr, "csgw-socket-addr", "", "Socket address of service")
		clientOpts.StringVar(&agentConfig.ListenAddr, "listen-addr", "", "Client side GW socket address")
		clientOpts.Parse(os.Args[2:])

		server, err := socks5.NewApronAgentServer(socks5Config, &agentConfig, logger)
		internal.CheckError(err)

		go func() {
			err := server.ListenAndServe("tcp")
			internal.CheckError(err)
		}()
	case "sa":
		logger.Sugar().Infof("service side agent")
		serviceOpts := flag.NewFlagSet("service", flag.ExitOnError)
		agentConfig := socks5.ApronAgentServerConfig{
			Mode: socks5.ServerAgentMode,
		}

		serviceOpts.StringVar(&agentConfig.AgentId, "agent-id", "", "id of the service agent")
		serviceOpts.StringVar(&agentConfig.ListenAddr, "listen-addr", "", "Client side GW socket address")
		serviceOpts.StringVar(&agentConfig.RestMgmtAddr, "ssgw-addr", "", "RESTful management API address for service side gateway")
		serviceOpts.StringVar(&agentConfig.RemoteSocketAddr, "service-addr", "", "Socket address of service")
		serviceOpts.Parse(os.Args[2:])

		server, err := socks5.NewApronAgentServer(socks5Config, &agentConfig, logger)
		internal.CheckError(err)

		go func() {
			err := server.ListenAndServe("tcp")
			internal.CheckError(err)
		}()
	default:
		printMainUsage()
		os.Exit(1)
	}

	select {}
}

func printMainUsage() {
	fmt.Println("Provide subcommand `ca` or `sa` to get detail usage")
}
