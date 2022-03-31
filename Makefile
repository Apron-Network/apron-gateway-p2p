all: gen build

build: gw report_generator

SOURCES = $(wildcard internal/*/*.go internal/*.go cmd/*/*.go)


gen: proto/models.proto
	protoc --proto_path=proto --go_out=internal/models --go_opt=paths=source_relative proto/models.proto

gw: $(SOURCES)
	go build ./cmd/gateway

report_generator: $(SOURCES)
	go build ./cmd/report_generator

test:
	go test -v -cover ./...

clean:
	-rm gateway


.PHONY: gen clean



