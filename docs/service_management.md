# Service Management 

## Usage demo

> In this section, the service management API address is *http://localhost:8082*

### Create / Update a service

> If the service exists, the service will be updated using the latest value. 

*POST /service/*

| Param    | Type   | Desc                                                         | Sample value           |
| -------- | ------ | ------------------------------------------------------------ | ---------------------- |
| name     | string | Name of service,          | `test_httpbin_service` |
| id | string | Uniq id for service | httpbin/               |
| providers   | Array | Providers for this service Schema for building service, support http, https, ws, wss    |                    |


#### Provider 
| Param    | Type   | Desc                                                         | Sample value           |
| -------- | ------ | ------------------------------------------------------------ | ---------------------- |
| name     | string | Name of service provider,          |  |
| id | string | Uniq id for service provider |              |
| desc   | string |  The description of service provider  |   
| base_url   | string | Base url or name for service provider, all request will be forwarded to this   |  
| schema   | string | Providers for this service Schema for building service, support http, https, ws, wss    |  


```shell
curl --location --request POST 'http://localhost:8082/service' \
--header 'Content-Type: application/json' \
--data-raw '{
    "id" : "test",
    "domain_name": "localhost",
    "providers": [
        {
            "id" : "test_provider1",
            "name": "test_provider1 http provider1",
            "desc": "test http provider1 desc",
            "base_url": "localhost:8080",
            "schema": "ws"
        }

    ]
```

### List all services

*GET /service/*

```shell
curl --location --request GET 'http://localhost:8082/service'
```

### List Local Service from current gateway
*GET /service/local*

```shell
curl --location --request GET 'http://localhost:8082/service/local'
```

### List Remote Service from other gateways
*GET /service/remote*

```shell
curl --location --request GET 'http://localhost:8082/service/remote'
```

### List Service and its P2P peers

> if a peer disconnected, the related services will be removed automaticall by p2p network.

*GET service/peers*

```shell
curl --location --request GET 'http://localhost:8082/service/peers'
```

### Delete Service
*DELETE /service*

| Param    | Type   | Desc                                                         | Sample value           |
| -------- | ------ | ------------------------------------------------------------ | ---------------------- |
| name     | string | Name of service provider,          |  |
| id | string | Uniq id for service provider |              |

```shell
curl --location --request DELETE 'http://localhost:8082/service' \
--header 'Content-Type: application/json' \
--data-raw '{
    "id" : "test"
}'
```