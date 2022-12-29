all: gen build

.PHONY: gen clean

gen:internal/models/models.pb.go

build: bin/gateway bin/report_generator bin/socks_app bin/socks_server

SOURCES = $(wildcard internal/*/*.go internal/*.go)

internal/models/models.pb.go: proto/models.proto
	protoc --proto_path=proto --go_out=internal/models --go_opt=paths=source_relative proto/models.proto

bin/gateway: $(SOURCES) cmd/gateway/main.go
	go build -o bin/ ./cmd/gateway

bin/report_generator: $(SOURCES) cmd/report_generator/main.go
	go build -o bin/ ./cmd/report_generator

bin/socks_app: $(SOURCES) cmd/socks_app/main.go
	go build -o bin/ ./cmd/socks_app/

bin/socks_server: $(SOURCES) cmd/socks_server/main.go
	go build -o bin/ ./cmd/socks_server/

test:
	go test -v -cover ./...

clean:
	-rm -rf bin/





