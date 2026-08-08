.PHONY: test build gui android docker-build docker-check docker-traffic-check docker-up docker-down clean

GO ?= go
DOCKER ?= docker

test:
	$(GO) test ./...
	cd desktop-gui && $(GO) test ./...

build:
	mkdir -p bin
	$(GO) build -trimpath -o bin/socks2vpn ./cmd/socks2vpn

gui:
	mkdir -p bin
	cd desktop-gui && $(GO) build -trimpath -o ../bin/socks2vpn-gui .

android:
	cd android && ./gradlew assembleDebug

docker-build:
	$(DOCKER) build --build-arg VERSION=dev -t go-socks2vpn:local .

docker-check:
	$(DOCKER) compose --profile check run --build --rm socks4-check

docker-traffic-check:
	@status=0; $(DOCKER) compose --profile traffic run --build --rm traffic-check || status=$$?; $(DOCKER) compose down; exit $$status

docker-up:
	$(DOCKER) compose up --build socks2vpn

docker-down:
	$(DOCKER) compose down

clean:
	$(GO) clean
	cd desktop-gui && $(GO) clean
	cd android && ./gradlew clean
