
BASE := $(shell pwd)
BIN ?= $(BASE)/_output

CC=go build
GCFLAGS=
LDFLAGS='-w -s'
SRC=$(sell find . -name "*.go")

$(BIN)/jstio: $(SRC)
	CGO_ENABLED=0 $(CC) -ldflags=$(LDFLAGS) -o $@ $(BASE)/cmd/jstio/main.go

$(BIN)/xdsclient: $(SRC)
	CGO_ENABLED=0 $(CC) -ldflags=$(LDFLAGS) -o $@ $(BASE)/cmd/xdsclient/main.go

$(BIN)/envoy-init: $(SRC)
	CGO_ENABLED=0 $(CC) -ldflags=$(LDFLAGS) -o $@ $(BASE)/cmd/init-container/main.go

$(BIN)/logserver: $(SRC)
	CGO_ENABLED=0 $(CC) -ldflags=$(LDFLAGS) -o $@ $(BASE)/cmd/logserver/cmd/main.go

$(BIN)/metrics-server: $(SRC)
	CGO_ENABLED=0 $(CC) -ldflags=$(LDFLAGS) -o $@ $(BASE)/cmd/metrics-server/cmd/main.go

all: $(BIN)/jstio $(BIN)/xdsclient $(BIN)/envoy-init $(BIN)/logserver $(BIN)/metrics-server

.PHONY: clean
clean:
	rm -rf ${BIN}
