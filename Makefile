.PHONY: default build install-livekit-server run-livekit-server
default: build 
build:
	go build -buildmode=c-archive -o clivekit.a .
install-livekit-server:
	GOBIN=$(CURDIR)/bin/livekit-server go install github.com/livekit/livekit-server/cmd/server@v1.9.1
	mv ./bin/livekit-server/server ./bin/server 
	rm -rf ./bin/livekit-server 
	mv ./bin/server ./bin/livekit-server 
run-livekit-server:
	./bin/livekit-server --dev 
