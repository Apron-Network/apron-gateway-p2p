all: gen build

.PHONY: gen clean

gen:internal/models/models.pb.go

build: gateway report_generator

SOURCES = $(wildcard internal/*/*.go internal/*.go)

internal/models/models.pb.go: proto/models.proto
	protoc --proto_path=proto --go_out=internal/models --go_opt=paths=source_relative proto/models.proto

gateway: $(SOURCES) cmd/gateway/main.go
	go build ./cmd/gateway

report_generator: $(SOURCES) cmd/report_generator/main.go
	go build ./cmd/report_generator

test:
	go test -v -cover ./...

clean:
	-rm gateway





