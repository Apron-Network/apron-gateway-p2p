all: gen build

.PHONY: gen clean test docker

GO_SOURCES = $(wildcard internal/*/*.go internal/*.go)
OUTPUT_BINS = $(patsubst cmd/%, bin/%, $(wildcard cmd/*))
DOCKER_IMG_NAME = apron/gateway_p2p

PB_FILES = $(wildcard proto/*.proto)
GEN_PB_GO_FILES = $(patsubst proto/%,internal/models/%,$(patsubst %.proto,%.pb.go,$(PB_FILES)))

gen: $(GEN_PB_GO_FILES)
build: $(OUTPUT_BINS)

internal/models/%.pb.go: proto/%.proto
	protoc --proto_path=proto --go_out=internal/models --go_opt=paths=source_relative $<

bin/%: ./cmd/%/main.go $(GO_SOURCES)
	go build -o $@ ./$(shell dirname $<)

docker:
	docker build -t $(DOCKER_IMG_NAME) .

test:
	go test -v -cover ./...

clean:
	-rm -rf bin/





