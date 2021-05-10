package main

import (
	"context"
	"fmt"
)

const (
	LocalServiceMgmtRPCCall = "LocalServiceMgmtRPCApi"
	UpdateServiceRPCFunc    = "PushLocalServiceToRemote"
)

type LocalServiceMgmtRPCApi struct {
	rpcService *UpdateServiceRpcCall
}

// PushLocalServiceToRemote is invoked by remote host when new services registered.
// Remote host broadcast self service list with this RPC call, and all receiver
// hosts update self service routing table
func (api *LocalServiceMgmtRPCApi) PushLocalServiceToRemote(ctx context.Context, in ServiceBroadcastMsg, out *ServiceBroadcastMsg) error {
	fmt.Printf("PushLocalServiceToRemote in params: %+v\n", in)
	*out = api.rpcService.ReceiveRemoteServiceBroadcast(&in)
	return nil
}
