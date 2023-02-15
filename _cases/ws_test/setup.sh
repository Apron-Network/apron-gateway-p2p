#!/bin/bash

echo 'starting up ./bin/wsbin service for testing'
docker run --rm --name testwsbin -p 8978:8978 --entrypoint /app/wsbin -d apron/gateway_p2p
echo 'wsbin service has been started on localhost:8978'

echo 'registering service to alice_ssgw'
http post http://localhost:8082/service/ << EOF
{
    "id" : "bbbbbbbbbb",
    "name": "test_ws",
    "providers": [
        {
            "id" : "test_wsbin",
            "name": "test_wsbin provider",
            "desc": "test_wsbin provider",
            "base_url": "host.docker.internal:8978",
            "schema": "ws"
        }
    ]
}
EOF
echo 'service "bbbbbbbbbb" has been registered'

echo 'validating service can be found in both nodes'
http localhost:8082/service/local
http localhost:8083/service/remote

echo 'Now the testing can be done via'
echo 'websocat ws://localhost:8081/v1/bbbbbbbbbbhellouser'
