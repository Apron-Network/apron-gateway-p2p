# Test Case Set

The base network and some required service are defeined in `docker-compose.yml` file,
which are split by *profiles*. For using this features, please upgrade the docker compose to **1.28.0** or higher.

Before starting the tests, the docker image `apron/gateway_p2p` should be existing in the system.

## HTTP test

Shows http forwarding with Apron network.

### Prepare

```bash
cd $TOP/_cases
docker compose --profile http_test up -d
cd http_test
./setup.sh
```

### Test

Check the result manually

```bash
http post 'localhost:8081/v1/aaaaaaaaaahello/lala?foo=bar'
http get localhost:8081/v1/aaaaaaaaaahellouser/testget
```


## Websocket test

Forward websocket data stream via Apron network

### Prepare

```bash
cd $TOP/_cases
docker compose --profile ws_test up -d
cd ws_test
./setup.sh
```

### Test

The command below will open a websocket connection with CSGW, and the CSGW will connect to test websocket server *wsbin* accordingly.
The *wsbin* service will send current unix milliseconds every 2 seconds, and it can also echo user input back.

```bash
websocat ws://localhost:8081/v1/bbbbbbbbbbhellouser
```

## Socket test

Forward socket data stream via Apron network

### Prepare

```bash
cd $TOP/_cases
docker compose --profile socket_test up -d
```

### Test

Testing forwarding socket package requires built-in binary `socket_helper`. In client mode, the application sends hello every 2 seconds and gets echoed result.

```bash
./bin/socket_helper client -csgw-socket-addr localhost:9981 -service-id apn_socket_service -log-dir ./_cases/logs/
```

## Socks5 test

### Prepare

Forward socks5 request via Apron network to remote socks5 service.
Before doing this test, open docker-compose.yml file and replace the string `"replace.me.to.socks5.server.address"` with the real socks5 address.
For example, the common socks5 service is listening on `localhost:1080`.

```bash
docker-compose --profile socks5_test up
```

4 contains will be started after the command executed successfully.
And the service side agent will be registered to SSGW by default.

### Test

After containers started successfully, there will be a socket service listening on `localhost:10801`,
here are some testing commands:

```bash
# Test http request with curl
curl -x socks5://127.0.0.1:10801 ifconfig.co

# Test websocket connection with socks5 proxy
websocat --socks5=127.0.0.1:10801 wss://stream.binance.com:9443/ws/btcusdt@trade
```



