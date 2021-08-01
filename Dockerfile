# Build stage
FROM golang:1.16-buster AS build-env

WORKDIR /src
ADD go.mod /src
ADD go.sum /src
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

ADD . /src
RUN cd /src && make build

# Delivery stage
FROM debian:buster
WORKDIR /app
COPY --from=build-env /src/gateway /app/
ENTRYPOINT ["/app/gateway"]
