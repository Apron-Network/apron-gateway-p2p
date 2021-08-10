# The P2P Gateway for Apron Project

## Build

### Standalone

> Note: `protoc` and `protoc-gen-go` are required if want to regenerate protobuf model.

```shell
$ make gen  # Generate protobuf model
$ make build # Build gateway binary
```

## Environment setup

### Standalone

#### Bootstrap Node
The bootstrap node can be started with this command.

```
./gateway
```

The default values are the below. 

```
% ./gateway -h
Usage of ./gateway:
  -mgmt-addr string
        API base for management (default "localhost:8082")
  -p2p-port int
        Internal Port Used by p2p network (default 2145)
  -peers value
        Bootstrap Peers
  -service-addr string
        Service addr used for service forward (default ":8080")
 ```



### Client Node

Bootstrap peer can be got from bootstrap node log. Just like the below. 

```
2021/07/31 18:12:13 Connect to me on:
2021/07/31 18:12:13   /ip4/9.200.45.133/tcp/58536/p2p/QmURiNzjDDReotRe3tFxoBJsy6X56gcQg9th3xJTrDsmjQ
2021/07/31 18:12:13   /ip4/127.0.0.1/tcp/58536/p2p/QmURiNzjDDReotRe3tFxoBJsy6X56gcQg9th3xJTrDsmjQ
2021/07/31 18:12:13   /ip6/::1/tcp/58537/p2p/QmURiNzjDDReotRe3tFxoBJsy6X56gcQg9th3xJTrDsmjQ
```

```
./gateway -peers /ip4/127.0.0.1/tcp/52552/p2p/QmSHwNRPEvhkaiYjVgsHPTks8zGsGLaXkNUDvwu2rM84wZ -p2p-port 2143 -service-addr :8090 -mgmt-addr localhost:8081
```



