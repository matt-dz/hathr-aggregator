.PHONY: all build run clean

BINARY:=aggregator
BINDIR:=bin
DOCKER_TAG:=aggregator

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

all: build

build:
	@echo "🔨  Building $(BINARY)…"
	go build -o $(BINDIR)/$(BINARY) cmd/aggregator/main.go
	@echo "✓  Built $(BINDIR)/$(BINARY)"

run:
	@echo "🚀  Starting..."
	go run cmd/aggregator/main.go

docker-build:
	@echo "🔨🐳 Building docker image $(BINARY)…"
	docker build . -t $(DOCKER_TAG)
	@echo "✓  Built $(DOCKER_TAG)"

docker-run:
	@echo "🚀🐳  Starting docker image $(DOCKER_TAG)..."
	docker run --env-file .env $(DOCKER_TAG)

clean:
	@echo "🧹 Cleaning $(BINDIR)"
	rm $(BINDIR)/*
