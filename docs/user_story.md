# End to End test case

This document will show you how to build a fully Apron gateway network with 2 nodes by hand.
As a service provider, how do you register your service and join the gateway network.
As a user, how do you use the existing services.

## Apron Network with Alice (boot node) and Bob

You should prepare two machines, one is Alice and another is Bob. 

### Download docker image

Alice and Bob should do the same action. 

Please download the latest date of image from [here](https://drive.google.com/drive/folders/1W9X3BAYs9mU2VuBsnPd2axxRtPkXS9co?usp=sharing)


### Load docker image

Alice and Bob should do the same action. 

```
docker load < apron-node-2021xxxx.tar.gz
```

### Alice Starts firstly

Please download the boot configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-boot.yml)

```shell
$ ln -s docker-compose-boot.yml docker-compose.yml
$ docker-compose up -d
```
When the node starts you should see output simillar to this. You can get apron gateway boot node addr information, such as `/ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

```shell
$ docker-compose logs | grep apron-gateway
Attaching to apron-node, apron-gateway, polkadot-frontend
apron-gateway        | 2021/08/10 16:12:10 Host ID: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
apron-gateway        | 2021/08/10 16:12:10 Connect to me on:
apron-gateway        | 2021/08/10 16:12:10   /ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
apron-gateway        | 2021/08/10 16:12:10   /ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### The blockchain expoler 

In your web browser, navigate to http://<Alice_addr>:3001/?rpc=ws%3A%2F%2F<Alice_addr>%3A9944#/explorer

### Bob Joins

Please download the configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-client.yml)

#### Modify bootnodes
using the boot node addr instead of the [`xxx.xxx.xxx.xxx`](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-client.yml#L11) in `--bootnodes` of `apron-node` section.

#### Modify apron gateway bootnodes.


using the apron gateway boot node addr instead of the [`/ip4/xxx.xxx.xxx.xxx/tcp/2145/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-client.yml#L29) in `--peers` of `apron-gateway` section.

#### Bob Starts
```shell
$ ln -s docker-compose-client.yml docker-compose.yml
$ docker-compose up -d
```

### Deploy Contracts
 Get contract address 

### Deploy Nodejs tools


### Deploy Marketplace


## Service Provider

As a service provider, you set up a **httpbin** service on Bob Machine. 


### Setup Service

Please add the below into `docker-compose.yml` of Bob as a part of `services`. 

```yaml
  httpbin:
    container_name: httpbin-test
    image: kennethreitz/httpbin:latest
    ports:
      - "8088:80"
```

**Start Service**

```shell
docker-compose up -d httpbin
```

### Register service

Service is registered by RESTful request. Here is the sample request:

```shell
curl --location --request POST 'http://<Bob_addr>:8082/service' \
--header 'Content-Type: application/json' \
--data-raw '{
    "id" : "<Alice_addr>:8080",
    "domain_name": "<Bob_addr>",
    "providers": [
        {
            "id" : "httpbin_provider1",
            "name": "httpbin provider1",
            "desc": "http provider1 desc",
            "base_url": "http://<Bob_addr>:8088/anything",
            "schema": "http"
        }
    ]
}'
```
Please use your Bob machine ip address replace `<Bob_addr>`

### Query Service

The service will be published to all Apron gateway network. So you can query the above service from Alice and Bob. 

```shell
curl --location --request GET 'http://<Alice_addr>:8082/service'
```
Please use your Alice machine ip address replace `<Alice_addr>`


```shell
curl --location --request GET 'http://<Bob_addr>:8082/service'
```
Please use your Bob machine ip address replace `<Bob_addr>`

## User

As a user, you use the service from Alice machine.

### Use Service

The service on Bob machine can be accessed in this way from Alice machine gateway.

```shell
curl http://<Alice_addr>:8080/v1/testkey/anything/foobar
```

### Check Service and Usage Reponrt on Marketplace







