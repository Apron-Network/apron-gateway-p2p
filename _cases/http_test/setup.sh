#!/bin/bash

echo 'starting up httpbin service for testing'
docker run --rm --name testhttpbin -p 8923:80 -d kennethreitz/httpbin
echo 'httpbin service has been started on localhost:8923'

echo 'registering service to alice_ssgw'
http post http://localhost:8082/service/ << EOF
{
    "id" : "aaaaaaaaaa",
    "name": "foo_test",
    "providers": [
        {
            "id" : "test_httpbin",
            "name": "test_httpbin provider",
            "desc": "test_httpbin provider",
            "base_url": "host.docker.internal:8923/anything",
            "schema": "http"
        }
    ]
}
EOF
echo 'service "aaaaaaaaaa" has been registered'

echo 'validating service can be found in both nodes'
http localhost:8082/service/local
http localhost:8083/service/remote

echo 'Now the testing can be done via'
echo 'http get localhost:8081/v1/aaaaaaaaaahellouser/testget'
echo 'http post localhost:8081/v1/aaaaaaaaaahellouser/testpost'
