# Deployment

## Download docker image

Please download the latest date of image from [here](https://drive.google.com/drive/folders/1W9X3BAYs9mU2VuBsnPd2axxRtPkXS9co?usp=sharing)


## Load docker image

```
docker load < apron-node-2021xxxx.tar.gz
```

## Dev mode


Please download the dev configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-dev.yml)

```shell 
$ ln -s docker-compose-dev.yml docker-compose.yml
$ docker-compose up -d
```

## A Private Apron Network with Boot and Client Node

### Boot Node Starts firstly

Please download the boot configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-boot.yml)

```shell
$ ln -s docker-compose-boot.yml docker-compose.yml
$ docker-compose up -d
```
When the node starts you should see output simillar to this. You can get apron gateway boot node addr information, such as `/ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

```
$ docker-compose logs | grep apron-gateway
Attaching to apron-node, apron-gateway, polkadot-frontend
apron-gateway        | 2021/08/10 16:12:10 Host ID: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
apron-gateway        | 2021/08/10 16:12:10 Connect to me on:
apron-gateway        | 2021/08/10 16:12:10   /ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
apron-gateway        | 2021/08/10 16:12:10   /ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### The blockchain expoler 

In your web browser, navigate to http://<boot_addr>:3001/?rpc=ws%3A%2F%2F<boot_addr>%3A9944#/explorer

### Client Joins

Please download the configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-client.yml)

#### Modify apron node bootnodes
using the boot node addr instead of the [`xxx.xxx.xxx.xxx`](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-client.yml#L11) in `--bootnodes` of `apron-node` section.

#### Modify apron gateway bootnodes.


using the apron gateway boot node addr instead of the [`/ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-client.yml#L29) in `--peers` of `apron-gateway` section.

#### Client Starts
```shell
$ ln -s docker-compose-client.yml docker-compose.yml
$ docker-compose up -d
```

