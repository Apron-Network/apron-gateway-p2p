package main

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-gorpc"
)

type UpdateServiceRpcCall struct {
	rpcServer  *rpc.Server
	rpcClient  *rpc.Client
	host       host.Host
	protocolID protocol.ID
}

type ServiceBroadcastMsg struct {
	services string
}

func (rpcService *UpdateServiceRpcCall) SetupRPC() error {
	rpcAPI := LocalServiceMgmtRPCApi{rpcService: rpcService}

	rpcService.rpcServer = rpc.NewServer(rpcService.host, rpcService.protocolID)
	err := rpcService.rpcServer.Register(&rpcAPI)
	if err != nil {
		return err
	}

	rpcService.rpcClient = rpc.NewClientWithServer(rpcService.host, rpcService.protocolID, rpcService.rpcServer)
	return nil
}

//BroadcastLocalService broad self services to other nodes
func (rpcService *UpdateServiceRpcCall) BroadcastLocalService(services *[]string) error {
	var remotePeers peer.IDSlice
	for _, p := range rpcService.host.Peerstore().Peers() {
		if p != rpcService.host.ID() {
			remotePeers = append(remotePeers, p)
		}
	}

	fmt.Printf("Services in UpdateServiceRpcCall: %+v\n", services)
	fmt.Printf("Send message: %+v\n", (*services)[0])

	var replies = make([]*ServiceBroadcastMsg, len(remotePeers))
	errs := rpcService.rpcClient.MultiCall(
		Ctxts(len(remotePeers)),
		remotePeers,
		LocalServiceMgmtRPCCall,
		UpdateServiceRPCFunc,
		ServiceBroadcastMsg{services: (*services)[0]},
		ProcessBroadcastMsgReply(replies),
	)

	for i, err := range errs {
		if err != nil {
			fmt.Printf("Peer %s return error: %-v\n", remotePeers[i].Pretty(), err)
		} else {
			fmt.Printf("Peer %s return msg: %-v\n", remotePeers[i].Pretty(), replies[i])
		}
	}

	return nil
}

func (rpcService *UpdateServiceRpcCall) ReceiveRemoteServiceBroadcast(msg *ServiceBroadcastMsg) ServiceBroadcastMsg {
	fmt.Printf("Received remote broadcast Msg: %+v\n", msg)
	return ServiceBroadcastMsg{services: msg.services}
}

func Ctxts(n int) []context.Context {
	ctxs := make([]context.Context, n)
	for i := 0; i < n; i++ {
		ctxs[i] = context.Background()
	}
	return ctxs
}

func ProcessBroadcastMsgReply(in []*ServiceBroadcastMsg) []interface{} {
	fmt.Printf("Remote RPC client pushed msg: %+v\n", in)
	ifaces := make([]interface{}, len(in))
	for i := range in {
		in[i] = &ServiceBroadcastMsg{}
		ifaces[i] = in[i]
	}
	return ifaces
}
