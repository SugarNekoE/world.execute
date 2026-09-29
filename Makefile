GO      ?= go
BIN     ?= bin/world.execute
PKG     ?= ./...

.PHONY: all build run test vet fmt tidy clean record help

all: build

build:
	$(GO) build -trimpath -o $(BIN) ./cmd/world.execute

run:
	$(GO) run ./cmd/world.execute

test:
	$(GO) test $(PKG)

vet:
	$(GO) vet $(PKG)

fmt:
	gofmt -l -w .

tidy:
	$(GO) mod tidy

# Dump frames instead of playing, for a quick look at the picture.
record:
	$(GO) run ./cmd/world.execute --frames 600 --record out.ansi
	@echo "wrote out.ansi, replay it with: cat out.ansi"

clean:
	rm -rf bin out.ansi world.execute.ansi

help:
	@$(GO) run ./cmd/world.execute --help
